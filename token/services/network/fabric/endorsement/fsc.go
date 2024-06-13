/*
Copyright IBM Corp. All Rights Reserved.

SPDX-License-Identifier: Apache-2.0
*/

package endorsement

import (
	"github.com/hyperledger-labs/fabric-smart-client/platform/fabric"
	"github.com/hyperledger-labs/fabric-smart-client/platform/view/view"
	"github.com/hyperledger-labs/fabric-token-sdk/token"
	tdriver "github.com/hyperledger-labs/fabric-token-sdk/token/driver"
	"github.com/hyperledger-labs/fabric-token-sdk/token/services/network/driver"
	"github.com/hyperledger-labs/fabric-token-sdk/token/services/network/fabric/fts"
	"github.com/pkg/errors"
)

func newFSCService(nw *fabric.NetworkService, tms *token.ManagementService, configuration tdriver.Configuration, viewRegistry ViewRegistry, viewManager ViewManager, identityProvider IdentityProvider) (*fscService, error) {
	tmsID := tms.ID()
	ch, err := nw.Channel(tmsID.Channel)
	if err != nil {
		return nil, errors.Wrapf(err, "failed getting channel [%s]", tmsID.Channel)
	}
	committer := ch.Committer()
	if configuration.GetBool("services.network.fabric.endorsement.endorser") {
		logger.Info("this node is an endorser, prepare it...")
		if err := committer.ProcessNamespace(tmsID.Namespace); err != nil {
			return nil, errors.WithMessagef(err, "failed to add namespace to committer [%s]", tmsID.Namespace)
		}
		if err := viewRegistry.RegisterResponder(&fts.RequestApprovalResponderView{}, &fts.RequestApprovalView{}); err != nil {
			return nil, errors.WithMessagef(err, "failed to register approval view for [%s]", tmsID)
		}
	}

	var endorserIDs []string
	if err := configuration.UnmarshalKey("services.network.fabric.endorsement.endorsers", &endorserIDs); err != nil {
		return nil, errors.WithMessage(err, "failed to load endorsers")
	}
	logger.Infof("defined [%s] as endorsers for [%s]", endorserIDs, tmsID)
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

	return &fscService{endorsers: endorsers, tms: tms, viewManager: viewManager}, nil
}

type fscService struct {
	tms         *token.ManagementService
	endorsers   []view.Identity
	viewManager ViewManager
}

func (e *fscService) Endorse(context view.Context, requestRaw []byte, signer view.Identity, txID driver.TxID) (driver.Envelope, error) {
	logger.Debugf("request approval via fts endrosers...")
	envBoxed, err := e.viewManager.InitiateView(&fts.RequestApprovalView{
		TMS:        e.tms,
		RequestRaw: requestRaw,
		TxID:       txID,
		Endorsers:  e.endorsers,
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
