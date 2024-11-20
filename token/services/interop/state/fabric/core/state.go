/*
Copyright IBM Corp. All Rights Reserved.

SPDX-License-Identifier: Apache-2.0
*/

package core

import (
	"encoding/base64"
	"encoding/json"
	"strings"
	"time"

	"github.com/hyperledger-labs/fabric-smart-client/platform/fabric"
	"github.com/hyperledger-labs/fabric-token-sdk/token/services/interop/pledge"
	"github.com/hyperledger-labs/fabric-token-sdk/token/services/interop/state"
	"github.com/hyperledger-labs/fabric-token-sdk/token/services/interop/state/driver"
	fabric3 "github.com/hyperledger-labs/fabric-token-sdk/token/services/interop/state/fabric"
	"github.com/hyperledger-labs/fabric-token-sdk/token/services/logging"
	"github.com/hyperledger-labs/fabric-token-sdk/token/services/network/common/rws/translator"
	"github.com/hyperledger-labs/fabric-token-sdk/token/services/network/fabric/tcc"
	"github.com/hyperledger-labs/fabric-token-sdk/token/token"
	"github.com/pkg/errors"
)

// KeyTranslator is used to translate tokens' concepts into backend's keys.
type KeyTranslator interface {
	CreateProofOfExistenceKey(tokenId *token.ID) (string, error)
	CreateProofOfNonExistenceKey(tokenID *token.ID, origin string) (string, error)
	CreateProofOfMetadataExistenceKey(tokenID *token.ID, origin string) (string, error)
}

type Validator interface {
	Validate(tok []byte, info *pledge.Info) error
}

type PledgeVault interface {
	PledgeByTokenID(tokenID *token.ID) (*pledge.Info, error)
}

type GetFabricNetworkServiceFunc = func(string) (*fabric.NetworkService, error)

type StateQueryExecutor struct {
	Logger           logging.Logger
	RelayProvider    fabric3.RelayProvider
	TargetNetworkURL string
	RelaySelector    *fabric.NetworkService
}

func NewStateQueryExecutor(
	Logger logging.Logger,
	RelayProvider fabric3.RelayProvider,
	targetNetworkURL string,
	relaySelector *fabric.NetworkService,
) (*StateQueryExecutor, error) {
	if err := fabric3.CheckFabricScheme(targetNetworkURL); err != nil {
		return nil, err
	}
	return &StateQueryExecutor{
		Logger:           Logger,
		RelayProvider:    RelayProvider,
		TargetNetworkURL: targetNetworkURL,
		RelaySelector:    relaySelector,
	}, nil
}

func (p *StateQueryExecutor) Exist(tokenID *token.ID) ([]byte, error) {
	raw, err := json.Marshal(tokenID)
	if err != nil {
		return nil, errors.Wrapf(err, "failed marshalling tokenID")
	}

	// get local relay
	relay := p.RelayProvider.Relay(p.RelaySelector)

	// Query
	p.Logger.Debugf("Query [%s] for proof of existence of token [%s], input [%s]", p.TargetNetworkURL, tokenID.String(), base64.StdEncoding.EncodeToString(raw))
	query, err := relay.ToFabric().Query(
		p.TargetNetworkURL,
		tcc.ProofOfTokenExistenceQuery,
		base64.StdEncoding.EncodeToString(raw),
	)
	if err != nil {
		return nil, errors.Wrapf(err, "failed querying token")
	}
	res, err := query.Call()
	if err != nil {
		// todo: move this to the query executor
		errMsg := err.Error()
		switch {
		case strings.Contains(errMsg, "failed to confirm if token with ID"):
			return nil, errors.WithMessagef(state.TokenDoesNotExistError, "%s", err)
		default:
			return nil, err
		}
	}

	return res.Proof()
}

func (p *StateQueryExecutor) DoesNotExist(tokenID *token.ID, origin string, deadline time.Time) ([]byte, error) {
	req := &tcc.ProofOfTokenNonExistenceRequest{
		Deadline:      deadline,
		OriginNetwork: origin,
		TokenID:       tokenID,
	}
	raw, err := json.Marshal(req)
	if err != nil {
		return nil, errors.Wrapf(err, "failed marshalling tokenID")
	}

	// get local relay
	relay := p.RelayProvider.Relay(p.RelaySelector)

	// Query
	p.Logger.Debugf("Query [%s] for proof of non-existence of token [%s], input [%s]", p.TargetNetworkURL, tokenID.String(), base64.StdEncoding.EncodeToString(raw))
	query, err := relay.ToFabric().Query(
		p.TargetNetworkURL,
		tcc.ProofOfTokenNonExistenceQuery,
		base64.StdEncoding.EncodeToString(raw),
	)
	if err != nil {
		return nil, errors.Wrapf(err, "failed querying token")
	}
	res, err := query.Call()
	if err != nil {
		errMsg := err.Error()
		switch {
		case strings.Contains(errMsg, "failed to confirm if token from network"):
			return nil, errors.WithMessagef(state.TokenExistsError, "%s", err)
		default:
			return nil, err
		}
	}

	return res.Proof()
}

// ExistsWithMetadata returns a proof that a token with metadata including the passed token ID and origin network exists
// in the network this query executor targets
func (p *StateQueryExecutor) ExistsWithMetadata(tokenID *token.ID, origin string) ([]byte, error) {
	req := &tcc.ProofOfTokenMetadataExistenceRequest{
		OriginNetwork: origin,
		TokenID:       tokenID,
	}
	raw, err := json.Marshal(req)
	if err != nil {
		return nil, errors.Wrapf(err, "failed to marshal proof of token metadata [%s]", req)
	}

	// Get local relay
	relay := p.RelayProvider.Relay(p.RelaySelector)

	// Query
	p.Logger.Debugf("Query [%s] for proof of existence of metadata with token [%s], input [%s]", p.TargetNetworkURL, tokenID.String(), base64.StdEncoding.EncodeToString(raw))
	query, err := relay.ToFabric().Query(
		p.TargetNetworkURL,
		tcc.ProofOfTokenMetadataExistenceQuery,
		base64.StdEncoding.EncodeToString(raw),
	)
	if err != nil {
		return nil, errors.Wrapf(err, "failed to query proof of metadata [%s]", req)
	}
	res, err := query.Call()
	if err != nil {
		errMsg := err.Error()
		switch {
		case strings.Contains(errMsg, "failed to confirm if token from network"):
			return nil, errors.WithMessagef(state.TokenExistsError, "%s", err)
		default:
			return nil, err
		}
	}

	return res.Proof()
}

type StateVerifier struct {
	Logger                  logging.Logger
	RelayProvider           fabric3.RelayProvider
	NetworkURL              string
	RelaySelector           *fabric.NetworkService
	PledgeVault             PledgeVault
	GetFabricNetworkService GetFabricNetworkServiceFunc
	validator               Validator
	keyTranslator           KeyTranslator
}

func NewStateVerifier(
	Logger logging.Logger,
	relayProvider fabric3.RelayProvider,
	PledgeVault PledgeVault,
	GetFabricNetworkService GetFabricNetworkServiceFunc,
	networkURL string,
	relaySelector *fabric.NetworkService,
	validator Validator,
	keyTranslator KeyTranslator,
) (*StateVerifier, error) {
	if err := fabric3.CheckFabricScheme(networkURL); err != nil {
		return nil, err
	}
	return &StateVerifier{
		Logger:                  Logger,
		RelayProvider:           relayProvider,
		NetworkURL:              networkURL,
		RelaySelector:           relaySelector,
		PledgeVault:             PledgeVault,
		GetFabricNetworkService: GetFabricNetworkService,
		validator:               validator,
		keyTranslator:           keyTranslator,
	}, nil
}

func (v *StateVerifier) VerifyProofExistence(proofRaw []byte, tokenID *token.ID, metadata []byte) error {
	// Get local relay
	relay := v.RelayProvider.Relay(v.RelaySelector)

	// Parse proof
	proof, err := relay.ToFabric().ProofFromBytes(proofRaw)
	if err != nil {
		return errors.Wrapf(err, "failed to unmarshal claim proof")
	}
	if err := proof.Verify(); err != nil {
		return errors.Wrapf(err, "failed to verify pledge proof")
	}

	// todo check that address in proof matches source network

	rwset, err := proof.RWSet()
	if err != nil {
		return errors.Wrapf(err, "failed to unmarshal claim proof")
	}

	key, err := v.keyTranslator.CreateProofOfExistenceKey(tokenID)
	if err != nil {
		return errors.Wrapf(err, "failed to create proof of existence key from token [%s]", tokenID)
	}
	tmsID, err := fabric3.FabricURLToTMSID(v.NetworkURL)
	if err != nil {
		return errors.Wrapf(err, "failed to extract tms id from [%s]", v.NetworkURL)
	}
	raw, err := rwset.GetState(tmsID.Namespace, key)
	if err != nil {
		return errors.Wrapf(err, "failed to get state for token [%s:%s]", tmsID.Namespace, key)
	}
	if len(raw) == 0 {
		return errors.Errorf("token [%s:%s] does not contain proof", tmsID.Namespace, key)
	}
	// Validate against pledge
	v.Logger.Debugf("verify proof of existence for token id [%s]", tokenID)
	pledge, err := v.PledgeVault.PledgeByTokenID(tokenID)
	if err != nil {
		v.Logger.Errorf("failed retrieving pledge info for token id [%s]: [%s]", tokenID, err)
		return errors.WithMessagef(err, "failed getting pledge for [%s]", tokenID)
	}
	if pledge == nil {
		v.Logger.Errorf("failed retrieving pledge info for token id [%s]: no info found", tokenID)
		return errors.Errorf("expected one pledge, got nil")
	}
	v.Logger.Debugf("found pledge info for token id [%s]: [%s]", tokenID, pledge.Source)

	// validate
	if err := v.validator.Validate(raw, pledge); err != nil {
		return errors.Wrapf(err, "failed to check token")
	}

	return nil
}

func (v *StateVerifier) VerifyProofNonExistence(proofRaw []byte, tokenID *token.ID, origin string, deadline time.Time) error {
	// v.NetworkURL is the network from which the proof comes from
	tokenOriginNetworkTMSID, err := fabric3.FabricURLToTMSID(origin)
	if err != nil {
		return errors.Wrapf(err, "failed to parse network url")
	}
	// get local relay
	fns, err := v.GetFabricNetworkService(tokenOriginNetworkTMSID.Network)
	if err != nil {
		return errors.Wrapf(err, "failed to get fabric network service for network [%s]", tokenOriginNetworkTMSID.Network)
	}
	relay := v.RelayProvider.Relay(fns)

	// parse proof
	proof, err := relay.ToFabric().ProofFromBytes(proofRaw)
	if err != nil {
		return errors.Wrapf(err, "failed to umarshal proof")
	}

	rwset, err := proof.RWSet()
	if err != nil {
		return errors.Wrapf(err, "failed to retrieve RWset")
	}

	key, err := v.keyTranslator.CreateProofOfNonExistenceKey(tokenID, origin)
	if err != nil {
		return errors.Wrapf(err, "failed creating key for proof of non-existence")
	}

	proofSourceNetworkTMSID, err := fabric3.FabricURLToTMSID(v.NetworkURL)
	if err != nil {
		return err
	}
	raw, err := rwset.GetState(proofSourceNetworkTMSID.Namespace, key)
	if err != nil {
		return errors.Wrapf(err, "failed to check proof of non-existence")
	}
	p := &translator.ProofOfTokenMetadataNonExistence{}
	if raw == nil {
		return errors.Errorf("could not find proof of non-existence")
	}
	err = json.Unmarshal(raw, p)
	if err != nil {
		return errors.Wrapf(err, "failed to unmarshal proof of non-existence")
	}
	if p.Deadline != deadline {
		return errors.Errorf("deadline in reclaim request does not match deadline in proof of non-existence")
	}
	if p.TokenID.String() != tokenID.String() {
		return errors.Errorf("token ID in reclaim request does not match token ID in proof of non-existence")
	}
	if p.Origin != fabric3.FabricURL(tokenOriginNetworkTMSID) {
		return errors.Errorf("origin in reclaim request does not match origin in proof of non-existence")
	}

	// todo check that address in proof is the destination network

	err = proof.Verify()
	if err != nil {
		return errors.Wrapf(err, "invalid proof of non-existence")
	}

	return nil
}

// VerifyProofTokenWithMetadataExistence verifies that a proof of existence of a token
// with metadata including the given token ID and origin network, in the target network is valid
func (v *StateVerifier) VerifyProofTokenWithMetadataExistence(proofRaw []byte, tokenID *token.ID, origin string) error {
	// v.NetworkURL is the network from which the proof comes from
	tokenOriginNetworkTMSID, err := fabric3.FabricURLToTMSID(origin)
	if err != nil {
		return errors.Wrapf(err, "failed to parse network url")
	}

	// get local relay
	fns, err := v.GetFabricNetworkService(tokenOriginNetworkTMSID.Network)
	if err != nil {
		return errors.Wrapf(err, "failed to get fabric network service for network [%s]", tokenOriginNetworkTMSID.Network)
	}
	relay := v.RelayProvider.Relay(fns)

	// parse proof
	proof, err := relay.ToFabric().ProofFromBytes(proofRaw)
	if err != nil {
		return errors.Wrapf(err, "failed to umarshal proof")
	}

	rwset, err := proof.RWSet()
	if err != nil {
		return errors.Wrapf(err, "failed to retrieve RWset")
	}

	key, err := v.keyTranslator.CreateProofOfMetadataExistenceKey(tokenID, origin)
	if err != nil {
		return errors.Wrapf(err, "failed creating key for proof of token existence")
	}

	proofSourceNetworkTMSID, err := fabric3.FabricURLToTMSID(v.NetworkURL)
	if err != nil {
		return err
	}
	raw, err := rwset.GetState(proofSourceNetworkTMSID.Namespace, key)
	if err != nil {
		return errors.Wrapf(err, "failed to check proof of token existence")
	}
	p := &translator.ProofOfTokenMetadataExistence{}
	if raw == nil {
		return errors.Errorf("could not find proof of token existence")
	}
	err = json.Unmarshal(raw, p)
	if err != nil {
		return errors.Wrapf(err, "failed to unmarshal proof of token existence")
	}
	if p.TokenID.String() != tokenID.String() {
		return errors.Errorf("token ID in redeem request does not match token ID in proof of token existence")
	}
	if p.Origin != fabric3.FabricURL(tokenOriginNetworkTMSID) {
		return errors.Errorf("origin in redeem request does not match origin in proof of token existence")
	}

	// todo check that address in proof is the destination network

	err = proof.Verify()
	if err != nil {
		return errors.Wrapf(err, "invalid proof of token existence")
	}

	return nil
}

type StateDriver struct {
	Logger        logging.Logger
	FNSProvider   *fabric.NetworkServiceProvider
	RelayProvider fabric3.RelayProvider
	VaultStore    *pledge.VaultStore
	Validator     Validator
	KeyTranslator KeyTranslator
}

func NewStateDriver(
	logger logging.Logger,
	FNSProvider *fabric.NetworkServiceProvider,
	relayProvider fabric3.RelayProvider,
	vaultStore *pledge.VaultStore,
	validator Validator,
	keyTranslator KeyTranslator,
) *StateDriver {
	return &StateDriver{
		Logger:        logger,
		FNSProvider:   FNSProvider,
		RelayProvider: relayProvider,
		VaultStore:    vaultStore,
		Validator:     validator,
		KeyTranslator: keyTranslator,
	}
}

func (d *StateDriver) NewStateQueryExecutor(url string) (driver.StateQueryExecutor, error) {
	fns, err := d.FNSProvider.FabricNetworkService("")
	if err != nil {
		return nil, errors.Wrapf(err, "failed to get default FNS")
	}

	return NewStateQueryExecutor(d.Logger, d.RelayProvider, url, fns)
}

func (d *StateDriver) NewStateVerifier(url string) (driver.StateVerifier, error) {
	fns, err := d.FNSProvider.FabricNetworkService("")
	if err != nil {
		return nil, errors.Wrapf(err, "failed to get default FNS")
	}
	return NewStateVerifier(
		d.Logger,
		d.RelayProvider,
		d.VaultStore,
		func(id string) (*fabric.NetworkService, error) {
			return d.FNSProvider.FabricNetworkService(id)
		},
		url,
		fns,
		d.Validator,
		d.KeyTranslator,
	)
}
