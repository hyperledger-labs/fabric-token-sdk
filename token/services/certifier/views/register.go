/*
Copyright IBM Corp. All Rights Reserved.

SPDX-License-Identifier: Apache-2.0
*/

package certifier

import (
	"time"

	"github.com/hyperledger-labs/fabric-smart-client/platform/view/view"
	"github.com/hyperledger-labs/fabric-token-sdk/token"
	"github.com/hyperledger-labs/fabric-token-sdk/token/services/certifier"
	"github.com/hyperledger-labs/fabric-token-sdk/token/services/logging"
	"github.com/pkg/errors"
)

var logger = logging.MustGetLogger("token-sdk.certifier")

type RegisterView struct {
	Network   string
	Channel   string
	Namespace string
	Wallet    string
}

func NewRegisterView(network string, channel string, namespace string, wallet string) *RegisterView {
	return &RegisterView{Network: network, Channel: channel, Namespace: namespace, Wallet: wallet}
}

func (r *RegisterView) Call(context view.Context) (interface{}, error) {
	// If the tms does not support graph hiding, skip
	tms := token.GetManagementService(
		context,
		token.WithNetwork(r.Network),
		token.WithChannel(r.Channel),
		token.WithNamespace(r.Namespace),
	)
	if tms == nil {
		return nil, errors.Errorf("tms not found [%s:%s:%s]", r.Namespace, r.Channel, r.Namespace)
	}
	pp := tms.PublicParametersManager().PublicParameters()
	if pp == nil {
		logger.Debugf("public parameters not yet available, start a background task...")
		go func() {
			for {
				pp := tms.PublicParametersManager().PublicParameters()
				if pp == nil {
					logger.Debugf("public parameters not yet available, wait...")
					time.Sleep(500 * time.Millisecond)
					continue
				}
				logger.Debugf("public parameters available, set certification service...")
				if err := r.startCertificationService(context, tms, pp); err != nil {
					logger.Errorf("failed to start certification service [%s]", err)
				}
				break
			}
		}()
	} else {
		logger.Debugf("public parameters available, set certification service...")
		if err := r.startCertificationService(context, tms, pp); err != nil {
			return nil, err
		}
	}

	return nil, nil
}

func (r *RegisterView) startCertificationService(context view.Context, tms *token.ManagementService, pp *token.PublicParameters) error {
	if !pp.GraphHiding() {
		logger.Warnf("the token management system for [%s:%s] does not support graph hiding, skipping certifier registration", r.Channel, r.Namespace)
		return nil
	}

	// Start Certifier
	certificationDriver := pp.CertificationDriver()
	logger.Debugf("start certification service with driver [%s]...", certificationDriver)
	c, err := certifier.NewCertificationService(tms, r.Wallet)
	if err != nil {
		return errors.WithMessagef(err, "failed instantiating certifier [%s]", tms)
	}
	if err := c.Start(); err != nil {
		return errors.WithMessagef(err, "failed starting certifier [%s]", tms)
	}
	return nil
}
