/*
Copyright IBM Corp. All Rights Reserved.

SPDX-License-Identifier: Apache-2.0
*/

package pledge

import (
	"encoding/json"
	"time"

	view2 "github.com/hyperledger-labs/fabric-smart-client/platform/view"
	"github.com/hyperledger-labs/fabric-smart-client/platform/view/services/session"
	"github.com/hyperledger-labs/fabric-smart-client/platform/view/view"
	"github.com/hyperledger-labs/fabric-token-sdk/token"
	"github.com/hyperledger-labs/fabric-token-sdk/token/services/network"
	"github.com/hyperledger-labs/fabric-token-sdk/token/services/ttx"
	token2 "github.com/hyperledger-labs/fabric-token-sdk/token/token"
	"github.com/pkg/errors"
)

func (t *Transaction) Claim(issuerWallet *token.IssuerWallet, typ string, value uint64, recipient view.Identity, originTokenID *token2.ID, originNetwork string, proof []byte) error {
	if typ == "" {
		return errors.Errorf("must specify a type")
	}
	if value == 0 {
		return errors.Errorf("must specify a value")
	}
	if recipient.IsNone() {
		return errors.Errorf("must specify a recipient")
	}
	if originTokenID == nil {
		return errors.Errorf("must specify the origin token ID")
	}
	if originNetwork == "" {
		return errors.Errorf("must specify the origin network")
	}
	if proof == nil {
		return errors.Errorf("must provide a proof")
	}

	_, err := t.TokenRequest.Issue(issuerWallet, recipient, typ, value, WithMetadata(originTokenID, originNetwork, proof))
	return err
}

func WithMetadata(tokenID *token2.ID, network string, proof []byte) token.IssueOption {
	return func(options *token.IssueOptions) error {
		if options.Attributes == nil {
			options.Attributes = make(map[interface{}]interface{})
		}
		options.Attributes["github.com/hyperledger-labs/fabric-token-sdk/token/services/interop/pledge/tokenID"] = tokenID
		options.Attributes["github.com/hyperledger-labs/fabric-token-sdk/token/services/interop/pledge/network"] = network
		options.Attributes["github.com/hyperledger-labs/fabric-token-sdk/token/services/interop/pledge/proof"] = proof
		return nil
	}
}

type ClaimRequest struct {
	TokenType          string
	Quantity           uint64
	Recipient          view.Identity
	RecipientAuditInfo []byte
	ClaimDeadline      time.Time
	OriginTokenID      *token2.ID
	OriginNetwork      string
	PledgeProof        []byte
	RequestorSignature []byte
}

func (cr *ClaimRequest) Bytes() ([]byte, error) {
	return json.Marshal(cr)
}

type claimInitiatorView struct {
	issuer      view.Identity
	recipient   view.Identity
	pledgeInfo  *Info
	pledgeProof []byte
}

func RequestClaim(context view.Context, issuer view.Identity, pledgeInfo *Info, recipient view.Identity, pledgeProof []byte) (view.Session, error) {
	boxed, err := context.RunView(&claimInitiatorView{
		issuer:      issuer,
		pledgeInfo:  pledgeInfo,
		recipient:   recipient,
		pledgeProof: pledgeProof,
	})
	if err != nil {
		return nil, err
	}
	return boxed.(view.Session), err
}

func (v *claimInitiatorView) Call(context view.Context) (interface{}, error) {
	session, err := context.GetSession(context.Initiator(), v.issuer)
	if err != nil {
		return nil, err
	}

	w := token.GetManagementService(context).WalletManager().OwnerWalletByIdentity(v.recipient)
	if w == nil {
		return nil, errors.Wrapf(err, "cannot find owner wallet for recipient [%s]", v.recipient)
	}
	auditInfo, err := w.GetAuditInfo(v.recipient)
	if err != nil {
		return nil, errors.Wrapf(err, "failed getting audit info for recipient [%s]", v.recipient)
	}

	req := &ClaimRequest{
		TokenType:          v.pledgeInfo.TokenType,
		Quantity:           v.pledgeInfo.Amount,
		Recipient:          v.recipient,
		RecipientAuditInfo: auditInfo,
		ClaimDeadline:      v.pledgeInfo.Script.Deadline,
		OriginTokenID:      v.pledgeInfo.TokenID,
		OriginNetwork:      v.pledgeInfo.Source,
		PledgeProof:        v.pledgeProof,
	}
	reqRaw, err := req.Bytes()
	if err != nil {
		return nil, errors.Wrapf(err, "failed marshalling claim request")
	}
	signer, err := token.GetManagementService(context).SigService().GetSigner(v.recipient)
	if err != nil {
		return nil, err
	}
	req.RequestorSignature, err = signer.Sign(append(reqRaw, context.Me().Bytes()...))
	if err != nil {
		return nil, err
	}
	reqRaw, err = req.Bytes()
	if err != nil {
		return nil, errors.Wrapf(err, "failed marshalling claim request")
	}
	err = session.Send(reqRaw)
	if err != nil {
		return nil, err
	}

	return session, nil
}

type receiveClaimRequestView struct{}

func ReceiveClaimRequest(context view.Context) (*ClaimRequest, error) {
	req, err := context.RunView(&receiveClaimRequestView{})
	if err != nil {
		return nil, err
	}
	return req.(*ClaimRequest), nil
}

func (v *receiveClaimRequestView) Call(context view.Context) (interface{}, error) {
	s := session.JSON(context)
	req := &ClaimRequest{}
	if err := s.Receive(&req); err != nil {
		return nil, errors.Wrapf(err, "failed unmarshalling claim request")
	}
	tms := token.GetManagementService(context)
	verifier, err := tms.SigService().OwnerVerifier(req.Recipient)
	if err != nil {
		return nil, errors.Wrapf(err, "failed to retrieve a verifier to check sender signature [%s]", req.Recipient)
	}
	request := &ClaimRequest{
		TokenType:          req.TokenType,
		Quantity:           req.Quantity,
		Recipient:          req.Recipient,
		RecipientAuditInfo: req.RecipientAuditInfo,
		ClaimDeadline:      req.ClaimDeadline,
		OriginTokenID:      req.OriginTokenID,
		OriginNetwork:      req.OriginNetwork,
		PledgeProof:        req.PledgeProof,
	}
	toBeVerified, err := json.Marshal(request)
	if err != nil {
		return nil, err
	}
	err = verifier.Verify(append(toBeVerified, s.Session().Info().Caller.Bytes()...), req.RequestorSignature)
	if err != nil {
		return nil, errors.Wrapf(err, "failed to verify claim request signature")
	}

	if err := view2.GetEndpointService(context).Bind(
		s.Session().Info().Caller,
		req.Recipient,
	); err != nil {
		return nil, errors.Wrapf(err, "failed binding caller's identity to request's recipient")
	}

	if err := tms.WalletManager().RegisterRecipientIdentity(&token.RecipientData{
		Identity:  req.Recipient,
		AuditInfo: req.RecipientAuditInfo}); err != nil {
		return nil, errors.Wrapf(err, "failed registering request recipient info")
	}

	return req, nil
}

func ValidateClaimRequest(context view.Context, req *ClaimRequest, opts ...ttx.TxOption) error {
	txOpts, err := ttx.CompileTxOption(opts...)
	if err != nil {
		return errors.WithMessage(err, "failed compiling tx options")
	}
	tms := token.GetManagementService(context, token.WithTMSID(txOpts.TMSID()))
	if tms == nil {
		return errors.Errorf("cannot find tms for [%s]", txOpts.TMSID())
	}

	tmsID := tms.ID()
	net := network.GetInstance(context, tmsID.Network, tmsID.Channel)
	if net == nil {
		return errors.Errorf("cannot find network for [%s]", tmsID)
	}
	destination := net.InteropURL(tmsID.Namespace)

	info := &Info{
		Amount:        req.Quantity,
		TokenID:       req.OriginTokenID,
		TokenMetadata: nil,
		TokenType:     req.TokenType,
		Source:        req.OriginNetwork,
		Script: &Script{
			Deadline:           req.ClaimDeadline,
			Recipient:          req.Recipient,
			DestinationNetwork: destination,
		},
	}

	if err := Vault(context).Store(info); err != nil {
		return errors.WithMessagef(err, "failed storing temporary pledge info for [%s]", info.Source)
	}
	ssp, err := GetStateServiceProvider(context)
	if err != nil {
		return errors.WithMessage(err, "failed getting state service provider")
	}
	v, err := ssp.Verifier(info.Source)
	if err != nil {
		return errors.WithMessagef(err, "failed getting verifier for [%s]", info.Source)
	}
	// todo check that address in proof matches the source network
	// todo check that destination network matches issuer's network
	err = v.VerifyProofExistence(req.PledgeProof, req.OriginTokenID, info.TokenMetadata)
	if err != nil {
		logger.Errorf("proof of existence in claim request is not valid valid [%s]", err)
		return errors.WithMessagef(err, "failed verifying proof of existence for [%s]", info.Source)
	}
	logger.Debugf("proof of existence in claim request is valid [%s]", err)

	if !req.ClaimDeadline.After(time.Now()) {
		return errors.Errorf("deadline for claim has elapsed")
	}

	return nil
}
