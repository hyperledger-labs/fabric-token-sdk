/*
Copyright IBM Corp. All Rights Reserved.

SPDX-License-Identifier: Apache-2.0
*/

package fabric

import (
	"encoding/base64"
	"encoding/json"
	"strings"
	"time"

	"github.com/hyperledger-labs/fabric-smart-client/platform/fabric"
	weaver2 "github.com/hyperledger-labs/fabric-smart-client/platform/fabric/services/weaver"
	"github.com/hyperledger-labs/fabric-smart-client/platform/view/services/flogging"
	fabric2 "github.com/hyperledger-labs/fabric-token-sdk/token/core/state/fabric"
	"github.com/hyperledger-labs/fabric-token-sdk/token/services/interop/pledge"
	"github.com/hyperledger-labs/fabric-token-sdk/token/services/network/fabric/tcc"
	"github.com/hyperledger-labs/fabric-token-sdk/token/services/vault/keys"
	"github.com/hyperledger-labs/fabric-token-sdk/token/services/vault/translator"
	"github.com/hyperledger-labs/fabric-token-sdk/token/token"
	"github.com/pkg/errors"
)

var logger = flogging.MustGetLogger("token-sdk.driver.zkatdlog")

type RelayProvider interface {
	Relay(fns *fabric.NetworkService) *weaver2.Relay
}

type PledgeVault interface {
	PledgeByTokenID(tokenID *token.ID) ([]*pledge.Info, error)
}

type GetFabricNetworkServiceFunc = func(string) *fabric.NetworkService

type StateQueryExecutor struct {
	RelayProvider    RelayProvider
	TargetNetworkURL string
	RelaySelector    *fabric.NetworkService
}

func NewStateQueryExecutor(RelayProvider RelayProvider, targetNetworkURL string, relaySelector *fabric.NetworkService) (*StateQueryExecutor, error) {
	if err := fabric2.CheckFabricScheme(targetNetworkURL); err != nil {
		return nil, err
	}
	return &StateQueryExecutor{RelayProvider: RelayProvider, TargetNetworkURL: targetNetworkURL, RelaySelector: relaySelector}, nil
}

func (p *StateQueryExecutor) Exist(tokenID *token.ID) ([]byte, error) {
	raw, err := json.Marshal(tokenID)
	if err != nil {
		return nil, err
	}

	relay := p.RelayProvider.Relay(p.RelaySelector)
	logger.Debugf("Query [%s] for proof of existence of token [%s], input [%s]", p.TargetNetworkURL, tokenID.String(), base64.StdEncoding.EncodeToString(raw))

	query, err := relay.ToFabric().Query(p.TargetNetworkURL, tcc.ProofOfTokenExistenceQuery, base64.StdEncoding.EncodeToString(raw))
	if err != nil {
		return nil, err
	}
	res, err := query.Call()
	if err != nil {
		// todo: move this to the query executor
		errMsg := err.Error()
		switch {
		case strings.Contains(errMsg, "failed to confirm if token with ID"):
			return nil, errors.WithMessagef(pledge.TokenDoesNotExistError, "%s", err)
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
		return nil, err
	}
	relay := p.RelayProvider.Relay(p.RelaySelector)

	logger.Debugf("Query [%s] for proof of non-existence of token [%s], input [%s]", p.TargetNetworkURL, tokenID.String(), base64.StdEncoding.EncodeToString(raw))

	query, err := relay.ToFabric().Query(p.TargetNetworkURL, tcc.ProofOfTokenNonExistenceQuery, base64.StdEncoding.EncodeToString(raw))
	if err != nil {
		return nil, err
	}
	res, err := query.Call()
	if err != nil {
		errMsg := err.Error()
		switch {
		case strings.Contains(errMsg, "failed to confirm if token from network"):
			return nil, errors.WithMessagef(pledge.TokenExistsError, "%s", err)
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
		return nil, err
	}
	relay := p.RelayProvider.Relay(p.RelaySelector)

	logger.Debugf("Query [%s] for proof of existence of metadata with token [%s], input [%s]", p.TargetNetworkURL, tokenID.String(), base64.StdEncoding.EncodeToString(raw))

	query, err := relay.ToFabric().Query(p.TargetNetworkURL, tcc.ProofOfTokenMetadataExistenceQuery, base64.StdEncoding.EncodeToString(raw))
	if err != nil {
		return nil, err
	}
	res, err := query.Call()
	if err != nil {
		errMsg := err.Error()
		switch {
		case strings.Contains(errMsg, "failed to confirm if token from network"):
			return nil, errors.WithMessagef(pledge.TokenExistsError, "%s", err)
		default:
			return nil, err
		}
	}

	return res.Proof()
}

type StateVerifier struct {
	RelayProvider           RelayProvider
	NetworkURL              string
	RelaySelector           *fabric.NetworkService
	PledgeVault             PledgeVault
	GetFabricNetworkService GetFabricNetworkServiceFunc
}

func NewStateVerifier(relayProvider RelayProvider, PledgeVault PledgeVault, GetFabricNetworkService GetFabricNetworkServiceFunc, networkURL string, relaySelector *fabric.NetworkService) (*StateVerifier, error) {
	if err := fabric2.CheckFabricScheme(networkURL); err != nil {
		return nil, err
	}
	return &StateVerifier{
		RelayProvider:           relayProvider,
		NetworkURL:              networkURL,
		RelaySelector:           relaySelector,
		PledgeVault:             PledgeVault,
		GetFabricNetworkService: GetFabricNetworkService,
	}, nil
}

func (v *StateVerifier) VerifyProofExistence(proofRaw []byte, tokenID *token.ID, metadata []byte) error {
	relay := v.RelayProvider.Relay(v.RelaySelector)
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

	key, err := keys.CreateProofOfExistenceKey(tokenID)
	if err != nil {
		return err
	}
	tmsID, err := pledge.FabricURLToTMSID(v.NetworkURL)
	if err != nil {
		return err
	}
	raw, err := rwset.GetState(tmsID.Namespace, key)
	if err != nil {
		return errors.Wrapf(err, "failed to check proof of existence")
	}
	if len(raw) == 0 {
		return errors.Errorf("failed to check proof of existence, missing key-value pair")
	}

	// Validate against pledge
	logger.Debugf("verify proof of existence for token id [%s]", tokenID)
	pledges, err := v.PledgeVault.PledgeByTokenID(tokenID)
	if err != nil {
		logger.Errorf("failed retrieving pledge info for token id [%s]: [%s]", tokenID, err)
		return errors.WithMessagef(err, "failed getting pledge for [%s]", tokenID)
	}
	if len(pledges) != 1 {
		logger.Errorf("failed retrieving pledge info for token id [%s]: no info found", tokenID)
		return errors.Errorf("expected one pledge, got [%d]", len(pledges))
	}
	info := pledges[0]
	logger.Debugf("found pledge info for token id [%s]: [%s]", tokenID, info.Source)

	// TODO compare token type and quantity and script, as done in fabtoken driver

	return nil
}

func (v *StateVerifier) VerifyProofNonExistence(proofRaw []byte, tokenID *token.ID, origin string, deadline time.Time) error {
	// v.NetworkURL is the network from which the proof comes from
	tokenOriginNetworkTMSID, err := pledge.FabricURLToTMSID(origin)
	if err != nil {
		return errors.Wrapf(err, "failed to parse network url")
	}
	relay := v.RelayProvider.Relay(v.GetFabricNetworkService(tokenOriginNetworkTMSID.Network))
	proof, err := relay.ToFabric().ProofFromBytes(proofRaw)
	if err != nil {
		return errors.Wrapf(err, "failed to umarshal proof")
	}

	rwset, err := proof.RWSet()
	if err != nil {
		return errors.Wrapf(err, "failed to retrieve RWset")
	}

	key, err := keys.CreateProofOfNonExistenceKey(tokenID, origin)
	if err != nil {
		return errors.Wrapf(err, "failed creating key for proof of non-existence")
	}

	proofSourceNetworkTMSID, err := pledge.FabricURLToTMSID(v.NetworkURL)
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
	if p.Origin != pledge.FabricURL(tokenOriginNetworkTMSID) {
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
	tokenOriginNetworkTMSID, err := pledge.FabricURLToTMSID(origin)
	if err != nil {
		return errors.Wrapf(err, "failed to parse network url")
	}
	relay := v.RelayProvider.Relay(v.GetFabricNetworkService(tokenOriginNetworkTMSID.Network))
	proof, err := relay.ToFabric().ProofFromBytes(proofRaw)
	if err != nil {
		return errors.Wrapf(err, "failed to umarshal proof")
	}

	rwset, err := proof.RWSet()
	if err != nil {
		return errors.Wrapf(err, "failed to retrieve RWset")
	}

	key, err := keys.CreateProofOfMetadataExistenceKey(tokenID, origin)
	if err != nil {
		return errors.Wrapf(err, "failed creating key for proof of token existence")
	}

	proofSourceNetworkTMSID, err := pledge.FabricURLToTMSID(v.NetworkURL)
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
	if p.Origin != pledge.FabricURL(tokenOriginNetworkTMSID) {
		return errors.Errorf("origin in redeem request does not match origin in proof of token existence")
	}

	// todo check that address in proof is the destination network

	err = proof.Verify()
	if err != nil {
		return errors.Wrapf(err, "invalid proof of token existence")
	}

	return nil
}
