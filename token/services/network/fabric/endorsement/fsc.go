/*
Copyright IBM Corp. All Rights Reserved.

SPDX-License-Identifier: Apache-2.0
*/

package endorsement

import (
	"math/rand"

	"github.com/hyperledger-labs/fabric-smart-client/platform/fabric"
	"github.com/hyperledger-labs/fabric-smart-client/platform/view/view"
	"github.com/hyperledger-labs/fabric-token-sdk/token"
	tdriver "github.com/hyperledger-labs/fabric-token-sdk/token/driver"
	"github.com/hyperledger-labs/fabric-token-sdk/token/services/network/common/rws/translator"
	"github.com/hyperledger-labs/fabric-token-sdk/token/services/network/driver"
	"github.com/pkg/errors"
)

const (
	AmIAnEndorserKey = "services.network.fabric.fsc_endorsement.endorser"
	EndorsersKey     = "services.network.fabric.fsc_endorsement.endorsers"
	PolicyType       = "services.network.fabric.fsc_endorsement.policy.type"

	OneOutNPolicy = "1outn"
	AllPolicy     = "all"
)

type FSCService struct {
	TmsID       token.TMSID
	Endorsers   []view.Identity
	ViewManager ViewManager
	PolicyType  string
}

func NewFSCService(
	fnsp *fabric.NetworkServiceProvider,
	tmsID token.TMSID,
	configuration tdriver.Configuration,
	viewRegistry ViewRegistry,
	viewManager ViewManager,
	identityProvider IdentityProvider,
	keyTranslator translator.KeyTranslator,
	getTranslator TranslatorProviderFunc,
) (*FSCService, error) {
	nw, err := fnsp.FabricNetworkService(tmsID.Network)
	if err != nil {
		return nil, errors.WithMessagef(err, "failed getting fabric network service for [%s]", tmsID.Network)
	}
	ch, err := nw.Channel(tmsID.Channel)
	if err != nil {
		return nil, errors.Wrapf(err, "failed getting channel [%s]", tmsID.Channel)
	}
	committer := ch.Committer()
	if configuration.GetBool(AmIAnEndorserKey) {
		logger.Debug("this node is an endorser, prepare it...")
		// if I'm an endorser, I need to process all token transactions
		if err := committer.ProcessNamespace(tmsID.Namespace); err != nil {
			return nil, errors.WithMessagef(err, "failed to add namespace to committer [%s]", tmsID.Namespace)
		}
		if err := viewRegistry.RegisterResponder(
			NewRequestApprovalResponderView(keyTranslator, getTranslator),
			&RequestApprovalView{},
		); err != nil {
			return nil, errors.WithMessagef(err, "failed to register approval view for [%s]", tmsID)
		}
	} else {
		logger.Debugf("this node is an not endorser, is key set? [%v].", configuration.IsSet(AmIAnEndorserKey))
	}

	policyType := configuration.GetString(PolicyType)
	if len(policyType) == 0 {
		policyType = AllPolicy
	}

	var endorserIDs []string
	if err := configuration.UnmarshalKey(EndorsersKey, &endorserIDs); err != nil {
		return nil, errors.WithMessage(err, "failed to load endorsers")
	}
	logger.Debugf("defined [%s] as endorsers for [%s]", endorserIDs, tmsID)
	if len(endorserIDs) == 0 {
		return nil, errors.Errorf("no endorsers found for [%s]", tmsID)
	}
	endorsers := make([]view.Identity, 0, len(endorserIDs))
	for _, id := range endorserIDs {
		if endorserID := identityProvider.Identity(id); endorserID.IsNone() {
			return nil, errors.Errorf("cannot find identity for endorser [%s]", id)
		} else {
			endorsers = append(endorsers, endorserID)
		}
	}

	return &FSCService{
		Endorsers:   endorsers,
		TmsID:       tmsID,
		ViewManager: viewManager,
		PolicyType:  policyType,
	}, nil
}

func (e *FSCService) Endorse(context view.Context, requestRaw []byte, signer view.Identity, txID driver.TxID) (driver.Envelope, error) {
	var endorsers []view.Identity
	switch e.PolicyType {
	case OneOutNPolicy:
		endorsers = []view.Identity{e.Endorsers[rand.Intn(len(e.Endorsers))]}
	case AllPolicy:
		endorsers = e.Endorsers
	default:
		endorsers = e.Endorsers
	}
	logger.Debugf("request approval via fts endrosers with policy [%s]: [%d]...", e.PolicyType, len(endorsers))

	envBoxed, err := e.ViewManager.InitiateView(&RequestApprovalView{
		TMSID:      e.TmsID,
		RequestRaw: requestRaw,
		TxID:       txID,
		Endorsers:  endorsers,
	}, context.Context())
	if err != nil {
		return nil, errors.WithMessagef(err, "failed to request approval")
	}
	env, ok := envBoxed.(driver.Envelope)
	if !ok {
		return nil, errors.Errorf("expected driver.Envelope, got [%T]", envBoxed)
	}
	return env, nil
}
