/*
Copyright IBM Corp. All Rights Reserved.

SPDX-License-Identifier: Apache-2.0
*/
package interactive

import (
	"fmt"
	"sync"
	"time"

	"github.com/pkg/errors"

	view2 "github.com/hyperledger-labs/fabric-smart-client/platform/view"
	"github.com/hyperledger-labs/fabric-smart-client/platform/view/services/session"
	"github.com/hyperledger-labs/fabric-smart-client/platform/view/view"

	token2 "github.com/hyperledger-labs/fabric-token-sdk/token"
	"github.com/hyperledger-labs/fabric-token-sdk/token/services/tcc"
	"github.com/hyperledger-labs/fabric-token-sdk/token/token"
)

type CertificationService struct {
	sp      view2.ServiceProvider
	wallets map[string]string
}

func NewCertificationService(sp view2.ServiceProvider) *CertificationService {
	return &CertificationService{
		sp:      sp,
		wallets: map[string]string{},
	}
}

func (c *CertificationService) Start() error {
	(&sync.Once{}).Do(func() {
		view2.GetRegistry(c.sp).RegisterResponder(c, &CertificationRequestView{})
	})
	return nil
}

func (c *CertificationService) SetWallet(network string, channel string, namespace string, wallet string) {
	c.wallets[network+":"+channel+":"+namespace] = wallet
}

func (c *CertificationService) Call(context view.Context) (interface{}, error) {
	// 1. receive request
	logger.Debugf("receive certification request [%s]", context.ID())
	s := session.JSON(context)
	var cr *CertificationRequest
	if err := s.Receive(&cr); err != nil {
		return nil, errors.WithMessage(err, "failed receiving certification request")
	}
	logger.Debugf("received certification request [%v]", cr)

	// TODO: 2. validate request, if needed

	// 3. invoke chaincode to get token commitment
	logger.Debugf("invoke chaincode to get commitments for [%v]", cr.IDs)
	// TODO: if the certifier fetches all token transactions, it might have the tokens in its on vault.
	tokensBoxed, err := context.RunView(tcc.NewGetTokensView(cr.Channel, cr.Namespace, cr.IDs...))
	if err != nil {
		return nil, errors.WithMessagef(err, "failed getting tokens [%s:%s][%v]", cr.Channel, cr.Namespace, cr.IDs)
	}

	tokens, ok := tokensBoxed.([][]byte)
	if !ok {
		return nil, errors.Errorf("expected [][]byte, got [%T]", tokens)
	}

	// 4. certify token commitment
	logger.Debugf("certify commitments for [%v]...", cr.IDs)
	tms := token2.GetManagementService(
		context,
		token2.WithNetwork(cr.Network),
		token2.WithChannel(cr.Channel),
		token2.WithNamespace(cr.Namespace),
	)
	walletKey := tms.Network() + ":" + tms.Channel() + ":" + tms.Namespace()
	logger.Debugf("lookup wallet ID with key [%s]", walletKey)
	walletID, ok := c.wallets[walletKey]
	if !ok {
		logger.Errorf("failed getting certifier wallet, namespace not registered [%s]: [%s]", cr, err)
		return nil, errors.WithMessagef(err, "failed getting certifier wallet, namespace not registered [%s]", cr)
	}
	logger.Debugf("certify with wallet [%s]", walletID)
	w := tms.WalletManager().CertifierWallet(walletID)
	if w == nil {
		return nil, errors.WithMessagef(err, "failed getting certifier wallet, wallet [%s] not found [%s:%s][%v]", walletID, cr.Channel, cr.Namespace, cr.IDs)
	}
	logger.Debugf("certify request [%v]", cr)
	certifications, err := tms.CertificationManager().Certify(w, cr.IDs, tokens, cr.Request)
	if err != nil {
		return nil, errors.WithMessagef(err, "failed certifying tokens [%s:%s][%v]", cr.Channel, cr.Namespace, cr.IDs)
	}

	// 5. respond
	logger.Debugf("send back certifications for [%v]", cr.IDs)
	if err := s.Send(certifications); err != nil {
		return nil, errors.WithMessagef(err, "failed sending certifications")
	}

	return nil, nil
}

type CertificationRequest struct {
	Network, Channel, Namespace string
	IDs                         []*token.ID
	Request                     []byte
}

func (cr *CertificationRequest) String() string {
	return fmt.Sprintf("CertificationRequest[%s,%s,%s][%v]", cr.Request, cr.Channel, cr.Namespace, cr.IDs)
}

type CertificationRequestView struct {
	network, channel, ns string
	ids                  []*token.ID
	certifier            view.Identity
}

func NewCertificationRequestView(channel, ns string, certifier view.Identity, ids ...*token.ID) *CertificationRequestView {
	return &CertificationRequestView{
		channel:   channel,
		certifier: certifier,
		ns:        ns,
		ids:       ids,
	}
}

func (i *CertificationRequestView) Call(context view.Context) (interface{}, error) {
	// 1. prepare request
	logger.Debugf("prepare certification request for [%v]", i.ids)
	cm := token2.GetManagementService(
		context,
		token2.WithNetwork(i.network),
		token2.WithChannel(i.channel),
		token2.WithNamespace(i.ns),
	).CertificationManager()
	cr, err := cm.NewCertificationRequest(i.ids)
	if err != nil {
		return nil, errors.WithMessagef(err, "failed creating certification request fo [%v]", i.ids)
	}

	// 2. send request
	logger.Debugf("send certification request for [%v]", i.ids)
	if i.certifier.IsNone() {
		return nil, errors.Errorf("no certifiers defined")
	}

	s, err := session.NewJSON(context, i, i.certifier)
	if err != nil {
		return nil, errors.WithMessagef(err, "failed opening session to [%s]", i.certifier)
	}
	if err := s.Send(&CertificationRequest{
		Channel:   i.channel,
		Namespace: i.ns,
		IDs:       i.ids,
		Request:   cr,
	}); err != nil {
		return nil, errors.WithMessagef(err, "failed sending certification request [%v] to [%s]", i.ids, i.certifier)
	}

	// 3. wait response
	logger.Debugf("wait certification request response for [%v]", i.ids)
	var certifications [][]byte
	if err := s.ReceiveWithTimeout(&certifications, 60*time.Second); err != nil {
		return nil, errors.WithMessagef(err, "failed receiving certifications [%v] from [%s]", i.ids, i.certifier)
	}

	// 4. Validate response
	logger.Debugf("validate certification request response for [%v]", i.ids)
	if err := cm.VerifyCertifications(i.ids, certifications); err != nil {
		logger.Errorf("failed verifying certifications of [%v] from [%s] with err [%s]", i.ids, i.certifier, err)
		return nil, errors.WithMessagef(err, "failed verifying certifications of [%v] from [%s]", i.ids, i.certifier)
	}

	logger.Debugf("certifications of [%v] from [%s] are valid", i.ids, i.certifier)

	// 5. return token certifications in the form of a map
	result := map[*token.ID][]byte{}
	for index, id := range i.ids {
		result[id] = certifications[index]
	}
	return result, nil
}
