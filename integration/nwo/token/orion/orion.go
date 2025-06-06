/*
Copyright IBM Corp. All Rights Reserved.

SPDX-License-Identifier: Apache-2.0
*/

package orion

import (
	"bytes"
	"fmt"
	"io"
	"os"
	"text/template"
	"time"

	math3 "github.com/IBM/mathlib"
	api2 "github.com/hyperledger-labs/fabric-smart-client/integration/nwo/api"
	"github.com/hyperledger-labs/fabric-smart-client/integration/nwo/fsc"
	sfcnode "github.com/hyperledger-labs/fabric-smart-client/integration/nwo/fsc/node"
	orion2 "github.com/hyperledger-labs/fabric-smart-client/integration/nwo/orion"
	"github.com/hyperledger-labs/fabric-smart-client/pkg/utils/errors"
	"github.com/hyperledger-labs/fabric-smart-client/platform/view/services/db/driver"
	common2 "github.com/hyperledger-labs/fabric-token-sdk/integration/nwo/token/common"
	"github.com/hyperledger-labs/fabric-token-sdk/integration/nwo/token/generators"
	"github.com/hyperledger-labs/fabric-token-sdk/integration/nwo/token/generators/dlog"
	"github.com/hyperledger-labs/fabric-token-sdk/integration/nwo/token/generators/fabtoken"
	topology2 "github.com/hyperledger-labs/fabric-token-sdk/integration/nwo/token/topology"
	"github.com/hyperledger-labs/fabric-token-sdk/token"
	"github.com/hyperledger-labs/fabric-token-sdk/token/services/logging"
	"github.com/hyperledger-labs/fabric-token-sdk/token/services/network/common/rws/keys"
	"github.com/hyperledger-labs/fabric-token-sdk/token/services/network/common/rws/translator"
	"github.com/hyperledger-labs/fabric-token-sdk/token/services/network/fabric/tcc"
	"github.com/hyperledger-labs/orion-sdk-go/pkg/bcdb"
	"github.com/hyperledger-labs/orion-sdk-go/pkg/config"
	logger2 "github.com/hyperledger-labs/orion-server/pkg/logger"
	"github.com/onsi/gomega"
	"gopkg.in/yaml.v2"
)

var logger = logging.MustGetLogger()

type Entry struct {
	TMS     *topology2.TMS
	Wallets map[string]*topology2.Wallets
}

type NetworkHandler struct {
	common2.NetworkHandler
	Entries map[string]*Entry
}

func NewNetworkHandler(tokenPlatform common2.TokenPlatform, builder api2.Builder) *NetworkHandler {
	return &NetworkHandler{
		NetworkHandler: common2.NetworkHandler{
			TokenPlatform:     tokenPlatform,
			EventuallyTimeout: 10 * time.Minute,
			CryptoMaterialGenerators: map[string]generators.CryptoMaterialGenerator{
				"fabtoken": fabtoken.NewCryptoMaterialGenerator(tokenPlatform, builder),
				"dlog":     dlog.NewCryptoMaterialGenerator(tokenPlatform, math3.BN254, builder),
			},
		},
		Entries: map[string]*Entry{},
	}
}

func (p *NetworkHandler) GenerateArtifacts(tms *topology2.TMS) {
	entry := p.GetEntry(tms)

	// Generate crypto material
	cmGenerator := p.CryptoMaterialGenerators[tms.Driver]
	gomega.Expect(cmGenerator).NotTo(gomega.BeNil(), "Crypto material generator for driver %s not found", tms.Driver)

	// - Setup
	root, err := cmGenerator.Setup(tms)
	gomega.Expect(err).NotTo(gomega.HaveOccurred())

	// - Generate crypto material for each FSC node
	fscTopology := p.TokenPlatform.GetContext().TopologyByName(fsc.TopologyName).(*fsc.Topology)
	for _, node := range fscTopology.Nodes {
		p.GenerateCryptoMaterial(cmGenerator, tms, node)
	}

	// Generate public parameters
	var ppRaw []byte
	ppGenerator := p.TokenPlatform.GetPublicParamsGenerators(tms.Driver)
	gomega.Expect(ppGenerator).NotTo(gomega.BeNil(), "No public params generator for driver %s", tms.Driver)
	args := []interface{}{root}
	for _, arg := range tms.PublicParamsGenArgs {
		args = append(args, arg)
	}

	logger.Debugf("Generating public parameters for [%s:%s] with args [%+v]", tms.ID(), args)
	wallets := &topology2.Wallets{}
	for _, w := range entry.Wallets {
		wallets.Issuers = append(wallets.Issuers, w.Issuers...)
		wallets.Auditors = append(wallets.Auditors, w.Auditors...)
		wallets.Certifiers = append(wallets.Certifiers, w.Certifiers...)
	}
	ppRaw, err = ppGenerator.Generate(tms, wallets, args...)
	gomega.Expect(err).ToNot(gomega.HaveOccurred())

	// - Store pp
	gomega.Expect(os.MkdirAll(p.TokenPlatform.PublicParametersDir(), 0766)).ToNot(gomega.HaveOccurred())
	gomega.Expect(os.WriteFile(p.TokenPlatform.PublicParametersFile(tms), ppRaw, 0766)).ToNot(gomega.HaveOccurred())

	gomega.Expect(os.MkdirAll(p.TokenPlatform.TokenDir(), 0766)).To(gomega.Succeed())
	gomega.Expect(p.AddInitConfig(tms)).To(gomega.Succeed())
}

func (p *NetworkHandler) AddInitConfig(tms *topology2.TMS) error {
	orion, ok := p.TokenPlatform.GetContext().PlatformByName(tms.BackendTopology.Name()).(*orion2.Platform)
	if !ok {
		return fmt.Errorf("target topology %s not found", tms.BackendTopology.Name())
	}

	var c HelperConfig
	path := p.HelperConfigPath()
	if conf, err := ReadHelperConfig(path); err == nil {
		c = *conf
	} else if !os.IsNotExist(err) {
		return err
	}

	if len(c.PPInitConfigs) == 0 {
		c.PPInitConfigs = []*PPInitConfig{}
	}
	custodianID := tms.BackendParams[Custodian].(string)
	c.PPInitConfigs = append(c.PPInitConfigs, &PPInitConfig{
		TMSID: token.TMSID{
			Network:   tms.Network,
			Channel:   tms.Channel,
			Namespace: tms.Namespace,
		},
		PPPath:                  p.TokenPlatform.PublicParametersFile(tms),
		CustodianID:             custodianID,
		CustodianCertPath:       orion.PemPath(custodianID),
		CustodianPrivateKeyPath: orion.KeyPath(custodianID),
		CACertPath:              orion.PemPath("CA"),
		ServerUrl:               orion.ServerUrl(),
		ServerID:                orion.ServerID(),
	})

	i, err := yaml.Marshal(c)
	if err != nil {
		return err
	}
	return os.WriteFile(p.HelperConfigPath(), i, 0766)
}

func (p *NetworkHandler) GenerateExtension(tms *topology2.TMS, node *sfcnode.Node, uniqueName string) string {
	gomega.Expect(os.MkdirAll(p.TTXDBSQLDataSourceDir(uniqueName), 0775)).ToNot(gomega.HaveOccurred(), "failed to create [%s]", p.TTXDBSQLDataSourceDir(uniqueName))
	gomega.Expect(os.MkdirAll(p.TokensDBSQLDataSourceDir(uniqueName), 0775)).ToNot(gomega.HaveOccurred(), "failed to create [%s]", p.TokensDBSQLDataSourceDir(uniqueName))
	gomega.Expect(os.MkdirAll(p.AuditDBSQLDataSourceDir(uniqueName), 0775)).ToNot(gomega.HaveOccurred(), "failed to create [%s]", p.AuditDBSQLDataSourceDir(uniqueName))
	gomega.Expect(os.MkdirAll(p.IdentityDBSQLDataSourceDir(uniqueName), 0775)).ToNot(gomega.HaveOccurred(), "failed to create [%s]", p.IdentityDBSQLDataSourceDir(uniqueName))

	persistenceNames := fsc.GetPersistenceNames(node.Options, common2.AllPrefixes...)

	t, err := template.New("peer").Funcs(template.FuncMap{
		"IsCustodian": func() bool {
			custodianNode, ok := tms.BackendParams[Custodian]
			if !ok {
				return false
			}
			return custodianNode.(string) == node.Name
		},
		"CustodianID": func() string {
			return tms.BackendParams[Custodian].(string)
		},
		"TMSID":                func() string { return tms.ID() },
		"TMS":                  func() *topology2.TMS { return tms },
		"Wallets":              func() *topology2.Wallets { return p.GetEntry(tms).Wallets[node.Name] },
		"TokenPersistence":     func() driver.PersistenceName { return persistenceNames[common2.TokenKey] },
		"IdentityPersistence":  func() driver.PersistenceName { return persistenceNames[common2.IdentityKey] },
		"TokenLockPersistence": func() driver.PersistenceName { return persistenceNames[common2.TokenLockKey] },
		"AuditTxPersistence":   func() driver.PersistenceName { return persistenceNames[common2.AuditTransactionKey] },
		"OwnerTxPersistence":   func() driver.PersistenceName { return persistenceNames[common2.OwnerTransactionKey] },
	}).Parse(Extension)
	gomega.Expect(err).NotTo(gomega.HaveOccurred())

	ext := bytes.NewBufferString("")
	err = t.Execute(io.MultiWriter(ext), p)
	gomega.Expect(err).NotTo(gomega.HaveOccurred())

	return ext.String()
}

func (p *NetworkHandler) PostRun(load bool, tms *topology2.TMS) {
	if load {
		return
	}

	c, err := ReadHelperConfig(p.HelperConfigPath())
	gomega.Expect(err).NotTo(gomega.HaveOccurred())
	ppConfig := c.GetByTMSID(token.TMSID{Network: tms.Network, Channel: tms.Channel, Namespace: tms.Namespace})
	gomega.Expect(ppConfig).NotTo(gomega.BeNil())
	gomega.Expect(ppConfig.Init()).To(gomega.Succeed())
}

func (p *NetworkHandler) Cleanup() {
}

func (p *NetworkHandler) UpdatePublicParams(tms *topology2.TMS, ppRaw []byte) {
	c, err := ReadHelperConfig(p.HelperConfigPath())
	gomega.Expect(err).NotTo(gomega.HaveOccurred())
	ppConfig := c.GetByTMSID(token.TMSID{Network: tms.Network, Channel: tms.Channel, Namespace: tms.Namespace})
	gomega.Expect(ppConfig).NotTo(gomega.BeNil())
	gomega.Expect(ppConfig.InitWithPPRaw(ppRaw)).To(gomega.Succeed())
}

func (p *NetworkHandler) GenIssuerCryptoMaterial(tms *topology2.TMS, nodeID string, walletID string) string {
	cmGenerator := p.CryptoMaterialGenerators[tms.Driver]
	gomega.Expect(cmGenerator).NotTo(gomega.BeNil(), "Crypto material generator for driver %s not found", tms.Driver)

	fscTopology := p.TokenPlatform.GetContext().TopologyByName(fsc.TopologyName).(*fsc.Topology)
	for _, node := range fscTopology.Nodes {
		if node.ID() == nodeID {
			ids := cmGenerator.GenerateIssuerIdentities(tms, node, walletID)
			return ids[0].Path
		}
	}
	gomega.Expect(false).To(gomega.BeTrue(), "cannot find FSC node [%s:%s]", tms.Network, nodeID)
	return ""
}

func (p *NetworkHandler) GenOwnerCryptoMaterial(tms *topology2.TMS, nodeID string, walletID string, useCAIfAvailable bool) (res token.IdentityConfiguration) {
	cmGenerator := p.CryptoMaterialGenerators[tms.Driver]
	gomega.Expect(cmGenerator).NotTo(gomega.BeNil(), "Crypto material generator for driver %s not found", tms.Driver)

	fscTopology := p.TokenPlatform.GetContext().TopologyByName(fsc.TopologyName).(*fsc.Topology)
	for _, node := range fscTopology.Nodes {
		if node.ID() == nodeID {
			ids := cmGenerator.GenerateOwnerIdentities(tms, node, walletID)
			res.ID = ids[0].ID
			res.URL = ids[0].Path
			res.Raw = ids[0].Raw
			return
		}
	}
	gomega.Expect(false).To(gomega.BeTrue(), "cannot find FSC node [%s:%s]", tms.Network, nodeID)
	return
}

func (p *NetworkHandler) SetCryptoMaterialGenerator(driver string, generator generators.CryptoMaterialGenerator) {
	p.CryptoMaterialGenerators[driver] = generator
}

func (p *NetworkHandler) GenerateCryptoMaterial(cmGenerator generators.CryptoMaterialGenerator, tms *topology2.TMS, node *sfcnode.Node) {
	entry := p.GetEntry(tms)
	o := node.PlatformOpts()
	opts := topology2.ToOptions(o)

	wallet := &topology2.Wallets{
		Certifiers: []topology2.Identity{},
		Issuers:    []topology2.Identity{},
		Owners:     []topology2.Identity{},
		Auditors:   []topology2.Identity{},
	}
	entry.Wallets[node.Name] = wallet

	// Issuer identities
	issuers := opts.Issuers()
	if len(issuers) != 0 {
		var index int
		found := false
		for i, issuer := range issuers {
			if issuer == node.ID() || issuer == "_default_" {
				index = i
				found = true
				issuers[i] = node.ID()
				break
			}
		}
		if !found {
			issuers = append(issuers, node.ID())
			index = len(issuers) - 1
		}

		ids := cmGenerator.GenerateIssuerIdentities(tms, node, issuers...)
		if len(ids) > 0 {
			wallet.Issuers = append(wallet.Issuers, ids...)
			wallet.Issuers[index].Default = true
		}
	}

	// Owner identities
	owners := opts.Owners()
	if len(owners) != 0 {
		var index int
		found := false
		for i, owner := range owners {
			if owner == node.ID() || owner == "_default_" {
				index = i
				found = true
				owners[i] = node.ID()
				break
			}
		}
		if !found {
			owners = append(owners, node.ID())
			index = len(owners) - 1
		}
		ids := cmGenerator.GenerateOwnerIdentities(tms, node, owners...)
		if len(ids) > 0 {
			wallet.Owners = append(wallet.Owners, ids...)
			wallet.Owners[index].Default = true
		}
	}

	// Auditor identity
	if opts.Auditor() {
		ids := cmGenerator.GenerateAuditorIdentities(tms, node, node.Name)
		if len(ids) > 0 {
			wallet.Auditors = append(wallet.Auditors, ids...)
			wallet.Auditors[len(wallet.Auditors)-1].Default = true
		}
	}

	// Certifier identities
	if opts.Certifier() {
		ids := cmGenerator.GenerateCertifierIdentities(tms, node, node.Name)
		if len(ids) > 0 {
			wallet.Certifiers = append(wallet.Certifiers, ids...)
			wallet.Certifiers[len(wallet.Certifiers)-1].Default = true
		}
	}
}

func (p *NetworkHandler) GetEntry(tms *topology2.TMS) *Entry {
	entry, ok := p.Entries[tms.Network+tms.Channel+tms.Namespace]
	if !ok {
		entry = &Entry{
			TMS:     tms,
			Wallets: map[string]*topology2.Wallets{},
		}
		p.Entries[tms.Network+tms.Channel+tms.Namespace] = entry
	}
	return entry
}

type PPInitConfig struct {
	TMSID                   token.TMSID `yaml:"tmsID"`
	PPPath                  string      `yaml:"ppPath"`
	CustodianID             string      `yaml:"custodianId"`
	CustodianCertPath       string      `yaml:"custodianCertPath"`
	CustodianPrivateKeyPath string      `yaml:"custodianPrivateKeyPath"`
	CACertPath              string      `yaml:"caCertPath"`
	ServerID                string      `yaml:"serverId"`
	ServerUrl               string      `yaml:"serverUrl"`
}

func (p *PPInitConfig) createUserSession(bcdb bcdb.BCDB) (bcdb.DBSession, error) {
	return bcdb.Session(&config.SessionConfig{
		UserConfig: &config.UserConfig{
			UserID:         p.CustodianID,
			CertPath:       p.CustodianCertPath,
			PrivateKeyPath: p.CustodianPrivateKeyPath,
		},
		TxTimeout: time.Second * 5,
	})
}

func (p *PPInitConfig) createDBInstance() (bcdb.BCDB, error) {
	c := &logger2.Config{
		Level:         "info",
		OutputPath:    []string{"stdout"},
		ErrOutputPath: []string{"stderr"},
		Encoding:      "console",
		Name:          "bcdb-client",
	}
	clientLogger, err := logger2.New(c)
	if err != nil {
		return nil, err
	}

	return bcdb.Create(&config.ConnectionConfig{
		RootCAs: []string{
			p.CACertPath,
		},
		ReplicaSet: []*config.Replica{
			{
				ID:       p.ServerID,
				Endpoint: p.ServerUrl,
			},
		},
		Logger: clientLogger,
	})
}

func (p *PPInitConfig) Init() error {
	ppRaw, err := os.ReadFile(p.PPPath)
	if err != nil {
		return errors.WithMessagef(err, "failed to read pp file [%s]", p.PPPath)
	}
	return p.InitWithPPRaw(ppRaw)
}

func (p *PPInitConfig) InitWithPPRaw(ppRaw []byte) error {
	// Store the public parameters in orion
	db, err := p.createDBInstance()
	if err != nil {
		return err
	}

	session, err := p.createUserSession(db)
	if err != nil {
		return err
	}
	tx, err := session.DataTx()
	if err != nil {
		return err
	}

	rwset := &RWSWrapper{
		db: p.TMSID.Namespace,
		me: p.CustodianID,
		tx: tx,
	}
	w := translator.New("", translator.NewRWSetWrapper(rwset, "", ""), &translator.HashedKeyTranslator{KT: &keys.Translator{}})
	action := &tcc.SetupAction{
		SetupParameters: ppRaw,
	}
	if err := w.Write(action); err != nil {
		return err
	}
	if _, _, err = tx.Commit(true); err != nil {
		return err
	}
	return nil
}

type HelperConfig struct {
	PPInitConfigs []*PPInitConfig `yaml:"ppInits"`
}

func ReadHelperConfig(path string) (*HelperConfig, error) {
	data, err := os.ReadFile(path)
	if err != nil {
		return nil, err
	}
	var c HelperConfig

	if err := yaml.Unmarshal(data, &c); err != nil {
		return nil, err

	}
	return &c, nil
}

func (c *HelperConfig) GetByTMSID(tmsID token.TMSID) *PPInitConfig {
	for _, config := range c.PPInitConfigs {
		if config.TMSID.Equal(tmsID) {
			return config
		}
	}
	return nil
}
