/*
Copyright IBM Corp. All Rights Reserved.

SPDX-License-Identifier: Apache-2.0
*/

package pledge

import (
	"time"

	"github.com/hyperledger-labs/fabric-smart-client/platform/view/view"
	"github.com/hyperledger-labs/fabric-token-sdk/token"
	"github.com/hyperledger-labs/fabric-token-sdk/token/services/network"
	token2 "github.com/hyperledger-labs/fabric-token-sdk/token/token"
	"github.com/pkg/errors"
)

type CollectProofOfExistenceView struct {
	tokenID *token2.ID
	source  string
}

func NewCollectProofOfExistenceView(tokenID *token2.ID, source string) *CollectProofOfExistenceView {
	return &CollectProofOfExistenceView{
		tokenID: tokenID,
		source:  source,
	}
}

func (c *CollectProofOfExistenceView) Call(context view.Context) (interface{}, error) {
	// get a query executor for the target network that should contain the pledge
	ssp, err := GetStateServiceProvider(context)
	if err != nil {
		return nil, errors.WithMessage(err, "failed getting state service provider")
	}
	p, err := ssp.QueryExecutor(c.source)
	if err != nil {
		return nil, errors.WithMessagef(err, "failed getting prover for [%s]", c.source)
	}
	return p.Exist(c.tokenID)
}

type CollectProofOfNonExistenceView struct {
	origin      string
	tokenID     *token2.ID
	deadline    time.Time
	destination string
}

func NewCollectProofOfNonExistenceView(tokenID *token2.ID, origin string, deadline time.Time, destination string) *CollectProofOfNonExistenceView {
	return &CollectProofOfNonExistenceView{
		origin:      origin,
		tokenID:     tokenID,
		deadline:    deadline,
		destination: destination,
	}
}

func (c *CollectProofOfNonExistenceView) Call(context view.Context) (interface{}, error) {
	ssp, err := GetStateServiceProvider(context)
	if err != nil {
		return nil, errors.WithMessage(err, "failed getting state service provider")
	}
	p, err := ssp.QueryExecutor(c.destination)
	if err != nil {
		return nil, errors.WithMessagef(err, "failed getting prover for [%s]", c.destination)
	}
	return p.DoesNotExist(c.tokenID, c.origin, c.deadline)
}

type CollectProofOfTokenWithMetadataExistenceView struct {
	origin      string
	tokenID     *token2.ID
	destination string
}

func NewCollectProofOfTokenWithMetadataExistenceView(tokenID *token2.ID, origin string, destination string) *CollectProofOfTokenWithMetadataExistenceView {
	return &CollectProofOfTokenWithMetadataExistenceView{
		origin:      origin,
		tokenID:     tokenID,
		destination: destination,
	}
}

func (c *CollectProofOfTokenWithMetadataExistenceView) Call(context view.Context) (interface{}, error) {
	ssp, err := GetStateServiceProvider(context)
	if err != nil {
		return nil, errors.WithMessage(err, "failed getting state service provider")
	}
	p, err := ssp.QueryExecutor(c.destination)
	if err != nil {
		return nil, errors.WithMessagef(err, "failed getting prover for [%s]", c.destination)
	}
	return p.ExistsWithMetadata(c.tokenID, c.origin)
}

// RequestProofOfExistence requests a proof of the existence of a pledge corresponding to the passed information
func RequestProofOfExistence(context view.Context, info *Info) ([]byte, error) {
	// collect proof
	boxed, err := context.RunView(NewCollectProofOfExistenceView(info.TokenID, info.Source))
	if err != nil {
		return nil, err
	}
	proof, ok := boxed.([]byte)
	if !ok {
		return nil, errors.Errorf("failed to collect proof of existence")
	}

	// verify proof before returning it
	// get a proof verifier for the network that generated the proof
	ssp, err := GetStateServiceProvider(context)
	if err != nil {
		return nil, errors.WithMessage(err, "failed getting state service provider")
	}
	v, err := ssp.Verifier(info.Source)
	if err != nil {
		return nil, errors.WithMessagef(err, "failed getting verifier for [%s]", info.Source)
	}
	if err := v.VerifyProofExistence(proof, info.TokenID, info.TokenMetadata); err != nil {
		return nil, errors.WithMessagef(err, "failed verifying proof of existence for [%s]", info.Source)
	}

	return proof, nil
}

// RequestProofOfNonExistence request a proof of non-existence of the given token, originally created in the given network,
// in the destination network identified by the given script.
// If no error is returned, the proof is valid with the respect to the given script.
func RequestProofOfNonExistence(context view.Context, tokenID *token2.ID, originTMSID token.TMSID, script *Script) ([]byte, error) {
	// collect proof
	net := network.GetInstance(context, originTMSID.Network, originTMSID.Channel)
	if net == nil {
		return nil, errors.Errorf("cannot find network for [%s]", originTMSID)
	}
	originNetwork := net.InteropURL(originTMSID.Namespace)

	boxed, err := context.RunView(NewCollectProofOfNonExistenceView(
		tokenID,
		originNetwork,
		script.Deadline,
		script.DestinationNetwork,
	))
	if err != nil {
		return nil, err
	}
	proof, ok := boxed.([]byte)
	if !ok {
		return nil, errors.Errorf("failed to collect proof of non-existence")
	}

	// verify proof before returning it
	ssp, err := GetStateServiceProvider(context)
	if err != nil {
		return nil, errors.WithMessage(err, "failed getting state service provider")
	}
	v, err := ssp.Verifier(script.DestinationNetwork)
	if err != nil {
		return nil, errors.WithMessagef(err, "failed getting verifier for [%s]", script.DestinationNetwork)
	}
	if err := v.VerifyProofNonExistence(proof, tokenID, originNetwork, script.Deadline); err != nil {
		return nil, errors.WithMessagef(err, "failed verifying proof of non-existence for [%s]", originNetwork)
	}

	return proof, nil
}

// RequestProofOfTokenWithMetadataExistence request a proof of a token existence with the given token ID and origin network,
// in the destination network identified by the given script.
// If no error is returned, the proof is valid with the respect to the given script.
func RequestProofOfTokenWithMetadataExistence(context view.Context, tokenID *token2.ID, originTMSID token.TMSID, script *Script) ([]byte, error) {
	// collect proof
	net := network.GetInstance(context, originTMSID.Network, originTMSID.Channel)
	if net == nil {
		return nil, errors.Errorf("cannot find network for [%s]", originTMSID)
	}
	originNetwork := net.InteropURL(originTMSID.Namespace)

	boxed, err := context.RunView(NewCollectProofOfTokenWithMetadataExistenceView(
		tokenID,
		originNetwork,
		script.DestinationNetwork,
	))
	if err != nil {
		return nil, err
	}
	proof, ok := boxed.([]byte)
	if !ok {
		return nil, errors.Errorf("failed to collect proof of token existence")
	}

	// verify proof before returning it
	ssp, err := GetStateServiceProvider(context)
	if err != nil {
		return nil, errors.WithMessage(err, "failed getting state service provider")
	}
	v, err := ssp.Verifier(script.DestinationNetwork)
	if err != nil {
		return nil, errors.WithMessagef(err, "failed getting verifier for [%s]", script.DestinationNetwork)
	}
	if err := v.VerifyProofTokenWithMetadataExistence(proof, tokenID, originNetwork); err != nil {
		return nil, errors.WithMessagef(err, "failed verifying proof of token existence for [%s]", originNetwork)
	}

	return proof, nil
}
