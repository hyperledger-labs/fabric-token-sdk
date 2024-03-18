/*
Copyright IBM Corp. All Rights Reserved.

SPDX-License-Identifier: Apache-2.0
*/

package idemix

import (
	"encoding/json"
	"os"
	"path/filepath"
	"sync"

	"github.com/IBM/idemix"
	"github.com/IBM/idemix/bccsp/keystore"
	"github.com/IBM/idemix/bccsp/types"
	"github.com/IBM/idemix/idemixmsp"
	math3 "github.com/IBM/mathlib"
	"github.com/hyperledger-labs/fabric-smart-client/pkg/utils/proto"
	"github.com/hyperledger-labs/fabric-smart-client/platform/view/services/flogging"
	"github.com/hyperledger-labs/fabric-smart-client/platform/view/services/hash"
	"github.com/hyperledger-labs/fabric-smart-client/platform/view/view"
	"github.com/hyperledger-labs/fabric-token-sdk/token/driver"
	"github.com/hyperledger-labs/fabric-token-sdk/token/driver/config"
	driver3 "github.com/hyperledger-labs/fabric-token-sdk/token/services/db/driver"
	"github.com/hyperledger-labs/fabric-token-sdk/token/services/identity/deserializer"
	"github.com/hyperledger-labs/fabric-token-sdk/token/services/identity/msp/common"
	config2 "github.com/hyperledger-labs/fabric-token-sdk/token/services/identity/msp/config"
	"github.com/hyperledger/fabric-protos-go/msp"
	"github.com/pkg/errors"
	"go.uber.org/zap/zapcore"
)

const (
	SignerConfigFull          = "SignerConfigFull"
	IdentityConfigurationType = "idemix"
)

var logger = flogging.MustGetLogger("token-sdk.services.identity.msp.idemix")

type PublicParametersWithIdemixSupport interface {
	IdemixCurve() math3.CurveID
}

type LocalMembership struct {
	config                 config2.Config
	defaultNetworkIdentity view.Identity
	signerService          common.SigService
	deserializerManager    deserializer.Manager
	identityDB             driver3.IdentityDB
	keystore               keystore.KVS
	mspID                  string
	cacheSize              int

	resolversMutex          sync.RWMutex
	resolvers               []*common.Resolver
	resolversByName         map[string]*common.Resolver
	resolversByEnrollmentID map[string]*common.Resolver
	curveID                 math3.CurveID
	identities              []*config.Identity
	// ignoreVerifyOnlyWallet when set to true, for each wallet the service will force the load of the secrets
	ignoreVerifyOnlyWallet bool
}

func NewLocalMembership(
	config config2.Config,
	defaultNetworkIdentity view.Identity,
	signerService common.SigService,
	deserializerManager deserializer.Manager,
	identityDB driver3.IdentityDB,
	keystore keystore.KVS,
	mspID string,
	cacheSize int,
	curveID math3.CurveID,
	identities []*config.Identity,
	ignoreVerifyOnlyWallet bool,
) *LocalMembership {
	return &LocalMembership{
		config:                  config,
		defaultNetworkIdentity:  defaultNetworkIdentity,
		signerService:           signerService,
		deserializerManager:     deserializerManager,
		identityDB:              identityDB,
		keystore:                keystore,
		mspID:                   mspID,
		cacheSize:               cacheSize,
		resolversByEnrollmentID: map[string]*common.Resolver{},
		resolversByName:         map[string]*common.Resolver{},
		curveID:                 curveID,
		identities:              identities,
		ignoreVerifyOnlyWallet:  ignoreVerifyOnlyWallet,
	}
}

func (lm *LocalMembership) DefaultNetworkIdentity() view.Identity {
	return lm.defaultNetworkIdentity
}

func (lm *LocalMembership) IsMe(id view.Identity) bool {
	return lm.signerService.IsMe(id)
}

func (lm *LocalMembership) GetIdentifier(id view.Identity) (string, error) {
	lm.resolversMutex.RLock()
	defer lm.resolversMutex.RUnlock()

	label := string(id)
	if logger.IsEnabledFor(zapcore.DebugLevel) {
		logger.Debugf("get anonymous identity info by label [%s]", hash.Hashable(label))
	}
	r := lm.getResolver(label)
	if r == nil {
		if logger.IsEnabledFor(zapcore.DebugLevel) {
			logger.Debugf("anonymous identity info not found for label [%s][%v]", hash.Hashable(label), lm.resolversByName)
		}
		return "", errors.New("not found")
	}
	return r.Name, nil
}

func (lm *LocalMembership) GetDefaultIdentifier() string {
	for _, resolver := range lm.resolvers {
		if resolver.Default {
			return resolver.Name
		}
	}
	return ""
}

func (lm *LocalMembership) GetIdentityInfo(label string, auditInfo []byte) (driver.IdentityInfo, error) {
	lm.resolversMutex.RLock()
	defer lm.resolversMutex.RUnlock()

	if logger.IsEnabledFor(zapcore.DebugLevel) {
		logger.Debugf("get anonymous identity info by label [%s]", hash.Hashable(label))
	}
	r := lm.getResolver(label)
	if r == nil {
		return nil, errors.Errorf("anonymous identity info not found for label [%s][%v]", hash.Hashable(label), lm.resolversByName)
	}

	return common.NewIdentityInfo(
		r.Name,
		r.EnrollmentID,
		r.Remote,
		func() (view.Identity, []byte, error) {
			return r.GetIdentity(&common.IdentityOptions{
				EIDExtension: true,
				AuditInfo:    auditInfo,
			})
		},
	), nil
}

func (lm *LocalMembership) RegisterIdentity(id string, path string) error {
	lm.resolversMutex.Lock()
	defer lm.resolversMutex.Unlock()

	if err := lm.identityDB.AddConfiguration(driver3.IdentityConfiguration{
		ID:   id,
		Type: IdentityConfigurationType,
		URL:  path,
	}); err != nil {
		return err
	}
	return lm.registerIdentity(config.Identity{ID: id, Path: path, Default: lm.GetDefaultIdentifier() == ""}, lm.curveID)
}

func (lm *LocalMembership) IDs() ([]string, error) {
	var ids []string
	for _, resolver := range lm.resolvers {
		ids = append(ids, resolver.Name)
	}
	return ids, nil
}

func (lm *LocalMembership) Reload(pp driver.PublicParameters) error {
	logger.Debugf("Reload Idemix Wallets for [%+q]", lm.identities)
	idemixPP, ok := pp.(PublicParametersWithIdemixSupport)
	if !ok {
		return errors.Errorf("public params do not support idemix")
	}
	// set curve id from the public parameters
	lm.curveID = idemixPP.IdemixCurve()

	logger.Debugf("Load Idemix Wallets with the respect to curve [%d], [%+q]", lm.curveID, lm.identities)

	lm.resolversMutex.Lock()
	defer lm.resolversMutex.Unlock()

	// cleanup all resolvers
	lm.resolvers = make([]*common.Resolver, 0)
	lm.resolversByName = make(map[string]*common.Resolver)
	lm.resolversByEnrollmentID = make(map[string]*common.Resolver)

	// load identities from configuration
	for _, identityConfig := range lm.identities {
		logger.Debugf("load wallet for identity [%+v]", identityConfig)
		if err := lm.registerIdentity(*identityConfig, lm.curveID); err != nil {
			return errors.WithMessage(err, "failed to load identity")
		}
		logger.Debugf("load wallet for identity [%+v] done.", identityConfig)
	}

	// load identity from KVS
	logger.Debugf("load identity from KVS")
	if err := lm.loadFromStorage(); err != nil {
		return errors.Wrapf(err, "failed to load identity from identityDB")
	}
	logger.Debugf("load identity from KVS done")

	// if no default identity, use the first one
	defaultIdentifier := lm.GetDefaultIdentifier()
	if len(defaultIdentifier) == 0 {
		logger.Warnf("no default identity, use the first one available")
		if len(lm.resolvers) > 0 {
			logger.Warnf("set default identity to %s", lm.resolvers[0].Name)
			lm.resolvers[0].Default = true
		} else {
			logger.Warnf("cannot set default identity, no identity available")
		}
	} else {
		logger.Debugf("default identifier is [%s]", defaultIdentifier)
	}

	return nil
}

func (lm *LocalMembership) registerIdentity(identity config.Identity, curveID math3.CurveID) error {
	// Try to register the MSP provider
	identity.Path = lm.config.TranslatePath(identity.Path)
	if err := lm.registerProvider(identity, curveID); err != nil {
		logger.Warnf("failed to load idemix msp provider at [%s]:[%s]", identity.Path, err)
		// Does path correspond to a holder containing multiple MSP identities?
		if err := lm.registerProviders(identity, curveID); err != nil {
			return errors.WithMessage(err, "failed to register MSP provider")
		}
	}
	return nil
}

func (lm *LocalMembership) registerProvider(identity config.Identity, curveID math3.CurveID) error {
	conf, err := GetLocalMspConfigWithType(identity.Path, lm.mspID, lm.ignoreVerifyOnlyWallet)
	if err != nil {
		logger.Debugf("failed reading idemix msp configuration from [%s]: [%s], try adding 'msp'...", identity.Path, err)
		// Try with "msp"
		conf, err = GetLocalMspConfigWithType(filepath.Join(identity.Path, "msp"), lm.mspID, lm.ignoreVerifyOnlyWallet)
		if err != nil {
			return errors.Wrapf(err, "failed reading idemix msp configuration from [%s] and with 'msp'", identity.Path)
		}
	}
	cryptoProvider, err := NewKVSBCCSP(lm.keystore, curveID)
	if err != nil {
		return errors.WithMessage(err, "failed to instantiate crypto provider")
	}
	provider, err := NewProvider(conf, lm.signerService, types.EidNymRhNym, cryptoProvider)
	if err != nil {
		return errors.Wrapf(err, "failed instantiating idemix msp provider from [%s]", identity.Path)
	}

	cacheSize, err := lm.cacheSizeForID(identity.ID)
	if err != nil {
		return err
	}

	var getIdentityFunc func(opts *common.IdentityOptions) (view.Identity, []byte, error)
	lm.deserializerManager.AddDeserializer(provider)
	if provider.IsRemote() {
		getIdentityFunc = func(opts *common.IdentityOptions) (view.Identity, []byte, error) {
			return nil, nil, errors.Errorf("cannot invoke this function, remote must register pseudonyms")
		}
	} else {
		getIdentityFunc = NewIdentityCache(provider.Identity, cacheSize, &common.IdentityOptions{}).Identity
	}
	lm.addResolver(identity.ID, provider.EnrollmentID(), provider.IsRemote(), identity.Default, getIdentityFunc)
	logger.Debugf("added idemix resolver for id [%s] with cache of size [%d], remote [%v]", identity.ID+"@"+provider.EnrollmentID(), cacheSize, provider.IsRemote())
	return nil
}

func (lm *LocalMembership) registerProviders(identity config.Identity, curveID math3.CurveID) error {
	entries, err := os.ReadDir(identity.Path)
	if err != nil {
		logger.Warnf("failed reading from [%s]: [%s]", identity.Path, err)
		return nil
	}
	found := 0
	for _, entry := range entries {
		if !entry.IsDir() {
			continue
		}
		id := entry.Name()
		if err := lm.registerProvider(config.Identity{ID: id, Path: filepath.Join(identity.Path, id), Default: false}, curveID); err != nil {
			logger.Errorf("failed registering msp provider [%s]: [%s]", id, err)
			continue
		}
		found++
	}
	if found == 0 {
		return errors.Errorf("no valid identities found in [%s]", identity.Path)
	}
	return nil
}

func (lm *LocalMembership) addResolver(Name string, EnrollmentID string, remote bool, defaultID bool, IdentityGetter common.GetIdentityFunc) {
	resolver := &common.Resolver{
		Name:         Name,
		Default:      defaultID,
		EnrollmentID: EnrollmentID,
		GetIdentity:  IdentityGetter,
		Remote:       remote,
	}
	lm.resolversByName[Name] = resolver
	if len(EnrollmentID) != 0 {
		lm.resolversByEnrollmentID[EnrollmentID] = resolver
	}
	lm.resolvers = append(lm.resolvers, resolver)
}

func (lm *LocalMembership) getResolver(label string) *common.Resolver {
	if logger.IsEnabledFor(zapcore.DebugLevel) {
		logger.Debugf("get anonymous identity info by label [%s]", hash.Hashable(label))
	}
	r, ok := lm.resolversByName[label]
	if ok {
		return r
	}

	if logger.IsEnabledFor(zapcore.DebugLevel) {
		logger.Debugf("anonymous identity info not found for label [%s][%v]", hash.Hashable(label), lm.resolversByName)
	}
	return nil
}

func (lm *LocalMembership) cacheSizeForID(id string) (int, error) {
	cacheSize, err := lm.config.CacheSizeForOwnerID(id)
	if err != nil {
		return 0, errors.WithMessage(err, "failed to obtain token management system instances")
	}
	if cacheSize == -1 {
		logger.Debugf("cache size for %s not configured, using default (%d)", id, lm.cacheSize)
		cacheSize = lm.cacheSize
	}
	return cacheSize, nil
}

func (lm *LocalMembership) loadFromStorage() error {
	it, err := lm.identityDB.IteratorConfigurations(IdentityConfigurationType)
	if err != nil {
		return errors.WithMessage(err, "failed to get registered identities from kvs")
	}
	defer it.Close()
	for it.HasNext() {
		entry, err := it.Next()
		if err != nil {
			return errors.WithMessagef(err, "failed to get next registered identities from kvs")
		}

		id := entry.ID
		if lm.getResolver(id) != nil {
			continue
		}
		if err := lm.registerIdentity(config.Identity{ID: id, Path: entry.URL, Default: lm.GetDefaultIdentifier() == ""}, lm.curveID); err != nil {
			return err
		}
	}
	return nil
}

func GetLocalMspConfigWithType(dir string, id string, ignoreVerifyOnlyWallet bool) (*msp.MSPConfig, error) {
	mspConfig, err := GetIdemixMspConfigWithType(dir, id, ignoreVerifyOnlyWallet)
	if err != nil {
		// load it using the fabric-ca format
		mspConfig2, err2 := GetFabricCAIdemixMspConfig(dir, id)
		if err2 != nil {
			return nil, errors.Wrapf(err2, "cannot get idemix msp config from [%s]: [%s]", dir, err)
		}
		mspConfig = mspConfig2
	}
	return mspConfig, nil
}

// GetIdemixMspConfigWithType returns the configuration for the Idemix MSP of the specified type
func GetIdemixMspConfigWithType(dir string, ID string, ignoreVerifyOnlyWallet bool) (*msp.MSPConfig, error) {
	ipkBytes, err := ReadFile(filepath.Join(dir, idemix.IdemixConfigDirMsp, idemix.IdemixConfigFileIssuerPublicKey))
	if err != nil {
		return nil, errors.Wrapf(err, "failed to read issuer public key file")
	}

	revocationPkBytes, err := ReadFile(filepath.Join(dir, idemix.IdemixConfigDirMsp, idemix.IdemixConfigFileRevocationPublicKey))
	if err != nil {
		return nil, errors.Wrapf(err, "failed to read revocation public key file")
	}

	idemixConfig := &idemixmsp.IdemixMSPConfig{
		Name:         ID,
		Ipk:          ipkBytes,
		RevocationPk: revocationPkBytes,
	}

	signerConfigPath := filepath.Join(dir, idemix.IdemixConfigDirUser, idemix.IdemixConfigFileSigner)
	if ignoreVerifyOnlyWallet {
		logger.Debugf("check the existence of SignerConfigFull")
		// check if `SignerConfigFull` exists, if yes, use that file
		path := filepath.Join(dir, idemix.IdemixConfigDirUser, SignerConfigFull)
		_, err := os.Stat(path)
		if err == nil {
			logger.Debugf("SignerConfigFull found, use it")
			signerConfigPath = path
		}
	}
	signerBytes, err := os.ReadFile(signerConfigPath)
	if err == nil {
		signerConfig := &idemixmsp.IdemixMSPSignerConfig{}
		err = proto.Unmarshal(signerBytes, signerConfig)
		if err != nil {
			return nil, err
		}
		idemixConfig.Signer = signerConfig
	}

	confBytes, err := proto.Marshal(idemixConfig)
	if err != nil {
		return nil, err
	}

	return &msp.MSPConfig{Config: confBytes, Type: int32(idemix.IDEMIX)}, nil
}

// GetFabricCAIdemixMspConfig returns the configuration for the Idemix MSP generated by Fabric-CA
func GetFabricCAIdemixMspConfig(dir string, ID string) (*msp.MSPConfig, error) {
	path := filepath.Join(dir, ConfigFileIssuerPublicKey)
	ipkBytes, err := ReadFile(path)
	if err != nil {
		return nil, errors.Wrapf(err, "failed to read issuer public key file at [%s]", path)
	}

	path = filepath.Join(dir, ConfigFileRevocationPublicKey)
	revocationPkBytes, err := ReadFile(path)
	if err != nil {
		return nil, errors.Wrapf(err, "failed to read revocation public key file at [%s]", path)
	}

	idemixConfig := &idemixmsp.IdemixMSPConfig{
		Name:         ID,
		Ipk:          ipkBytes,
		RevocationPk: revocationPkBytes,
	}

	path = filepath.Join(dir, ConfigDirUser, ConfigFileSigner)
	signerBytes, err := ReadFile(path)
	if err == nil {
		// signerBytes is a json structure, convert it to protobuf
		si := &SignerConfig{}
		if err := json.Unmarshal(signerBytes, si); err != nil {
			return nil, errors.Wrapf(err, "failed to json unmarshal signer config read at [%s]", path)
		}

		signerConfig := &idemixmsp.IdemixMSPSignerConfig{
			Cred:                            si.Cred,
			Sk:                              si.Sk,
			OrganizationalUnitIdentifier:    si.OrganizationalUnitIdentifier,
			Role:                            int32(si.Role),
			EnrollmentId:                    si.EnrollmentID,
			CredentialRevocationInformation: si.CredentialRevocationInformation,
			RevocationHandle:                si.RevocationHandle,
		}
		idemixConfig.Signer = signerConfig
	} else {
		if !os.IsNotExist(errors.Cause(err)) {
			return nil, errors.Wrapf(err, "failed to read the content of signer config at [%s]", path)
		}
	}

	confBytes, err := proto.Marshal(idemixConfig)
	if err != nil {
		return nil, err
	}

	return &msp.MSPConfig{Config: confBytes, Type: int32(idemix.IDEMIX)}, nil
}
