/*
Copyright IBM Corp. All Rights Reserved.

SPDX-License-Identifier: Apache-2.0
*/

package pledge

import (
	"encoding/json"
	"fmt"
	"time"

	"github.com/hyperledger-labs/fabric-smart-client/platform/view/services/hash"
	"github.com/hyperledger-labs/fabric-smart-client/platform/view/services/session"
	"github.com/hyperledger-labs/fabric-smart-client/platform/view/view"
	tokn "github.com/hyperledger-labs/fabric-token-sdk/token"
	"github.com/hyperledger-labs/fabric-token-sdk/token/services/network"
	"github.com/hyperledger-labs/fabric-token-sdk/token/token"
	"github.com/pkg/errors"
)

type IssuerApprovalRequest struct {
	OriginTMSID        tokn.TMSID
	TokenID            *token.ID
	Proof              []byte
	Destination        string
	RequestorSignature []byte
}

func (r *IssuerApprovalRequest) Bytes() ([]byte, error) {
	return json.Marshal(r)
}

func (r *IssuerApprovalRequest) FromBytes(raw []byte) error {
	return json.Unmarshal(raw, r)
}

type IssuerApprovalResponse struct {
	Signature []byte
}

func (r *IssuerApprovalResponse) Bytes() ([]byte, error) {
	return json.Marshal(r)
}

func (r *IssuerApprovalResponse) FromBytes(raw []byte) error {
	return json.Unmarshal(raw, r)
}

type RequestIssuerSignatureView struct {
	originTMSID  tokn.TMSID
	sender       view.Identity
	issuer       view.Identity
	tokenID      *token.ID
	reclaimProof []byte
	network      string
	pledgeID     string
}

func RequestIssuerSignature(context view.Context, tokenID *token.ID, originTMSID tokn.TMSID, script *Script, proof []byte) ([]byte, error) {
	boxed, err := context.RunView(&RequestIssuerSignatureView{
		originTMSID:  originTMSID,
		sender:       script.Sender,
		issuer:       script.Issuer,
		tokenID:      tokenID,
		reclaimProof: proof,
		network:      script.DestinationNetwork,
		pledgeID:     script.ID,
	})
	if err != nil {
		return nil, err
	}
	return boxed.([]byte), nil
}

func (v *RequestIssuerSignatureView) Call(context view.Context) (interface{}, error) {
	logger.Debugf("RequestIssuerSignatureView:caller [%s]", context.Initiator())

	session, err := context.GetSession(context.Initiator(), v.issuer)
	if err != nil {
		return nil, err
	}

	// Ask for issuer's signature
	req := &IssuerApprovalRequest{
		OriginTMSID: v.originTMSID,
		TokenID:     v.tokenID,
		Destination: v.network,
		Proof:       v.reclaimProof,
	}

	reqRaw, err := req.Bytes()
	if err != nil {
		return nil, errors.Wrapf(err, "failed marshalling issuer signature request")
	}
	// sign request
	logger.Debugf("sign request [%s]", v.sender)
	signer, err := tokn.GetManagementService(context, tokn.WithTMSID(v.originTMSID)).SigService().GetSigner(v.sender)
	if err != nil {
		return nil, err
	}
	msg := append(reqRaw, v.sender.Bytes()...)
	req.RequestorSignature, err = signer.Sign(msg)
	if err != nil {
		return nil, err
	}
	verifier, err := tokn.GetManagementService(context, tokn.WithTMSID(req.OriginTMSID)).SigService().OwnerVerifier(v.sender)
	if err != nil {
		return nil, errors.Wrapf(err, "failed to retrieve a verifier to check sender signature")
	}
	if err := verifier.Verify(msg, req.RequestorSignature); err != nil {
		return nil, errors.Wrapf(err, "failed to double-verify sender signature")
	}

	reqRaw, err = req.Bytes()
	if err != nil {
		return nil, errors.Wrapf(err, "failed marshalling issuer signature request")
	}
	err = session.Send(reqRaw)
	if err != nil {
		return nil, err
	}

	// Wait to receive a signature
	ch := session.Receive()
	var payload []byte
	select {
	case msg := <-ch:
		payload = msg.Payload
		if msg.Status == view.ERROR {
			return nil, errors.Errorf("failed requesting approval [%s]", string(payload))
		}
	case <-time.After(60 * time.Second):
		return nil, errors.New("time out reached")
	}
	logger.Debugf("received approval response [%v]", payload)

	res := &IssuerApprovalResponse{}
	err = res.FromBytes(payload)
	if err != nil {
		return nil, errors.Wrapf(err, "failed to unmarshal approval response [%s]", string(payload))
	}
	// check if signature is valid
	// TODO: The issuer here is identified with it is owner identity. Shall we have the issuer identity?
	verifier, err = tokn.GetManagementService(context, tokn.WithTMSID(req.OriginTMSID)).SigService().OwnerVerifier(v.issuer)
	if err != nil {
		return nil, errors.Wrapf(err, "failed to retrieve a verifier to check issuer signature")
	}

	err = verifier.Verify([]byte(v.pledgeID), res.Signature)
	if err != nil {
		return nil, errors.Wrapf(err, "invalid issuer signature")
	}
	return res.Signature, nil
}

type RequestIssuerSignatureResponderView struct {
	walletID string
}

func RespondRequestIssuerSignature(context view.Context, walletID string) ([]byte, error) {
	sig, err := context.RunView(&RequestIssuerSignatureResponderView{walletID: walletID})
	if err != nil {
		return nil, err
	}
	return sig.([]byte), nil
}

func (v *RequestIssuerSignatureResponderView) Call(context view.Context) (interface{}, error) {
	s, payload, err := session.ReadFirstMessage(context)
	if err != nil {
		return nil, err
	}

	req := &IssuerApprovalRequest{}
	if err := req.FromBytes(payload); err != nil {
		return nil, errors.Wrapf(err, "failed unmarshalling signature request")
	}
	w, err := GetIssuerWallet(context, v.walletID)
	if err != nil {
		return nil, err
	}

	wallet := NewIssuerWallet(context, w)
	_, script, err := wallet.GetPledgedToken(req.TokenID)
	if err != nil {
		return nil, err
	}
	// check validity of reclaim
	if time.Now().Before(script.Deadline) {
		return nil, errors.Errorf("cannot reclaim token yet; deadline has not elapsed yet")
	}
	if req.Destination != script.DestinationNetwork {
		return nil, errors.Errorf("destination network in reclaim request does not match destination network in pledged token")
	}
	// access control check
	logger.Debugf("verify request [%s]", script.Sender)
	verifier, err := tokn.GetManagementService(context, tokn.WithTMSID(req.OriginTMSID)).SigService().OwnerVerifier(script.Sender)
	if err != nil {
		return nil, errors.Wrapf(err, "failed to retrieve a verifier to check sender signature")
	}
	request := &IssuerApprovalRequest{
		OriginTMSID: req.OriginTMSID,
		TokenID:     req.TokenID,
		Proof:       req.Proof,
		Destination: req.Destination,
	}
	toBeVerified, err := json.Marshal(request)
	if err != nil {
		return nil, err
	}
	err = verifier.Verify(append(toBeVerified, script.Sender...), req.RequestorSignature)
	if err != nil {
		return nil, errors.Wrapf(err, "failed to verify reclaim request signature")
	}

	// verify proof before returning it
	net := network.GetInstance(context, req.OriginTMSID.Network, req.OriginTMSID.Channel)
	if net == nil {
		return nil, errors.Errorf("cannot find network for [%s]", req.OriginTMSID)
	}
	origin := net.InteropURL(req.OriginTMSID.Namespace)

	ssp, err := GetStateServiceProvider(context)
	if err != nil {
		return nil, errors.WithMessage(err, "failed getting state service provider")
	}
	stateProofVerifier, err := ssp.Verifier(script.DestinationNetwork)
	if err != nil {
		return nil, errors.WithMessagef(err, "failed getting verifier for [%s]", origin)
	}
	if err := stateProofVerifier.VerifyProofNonExistence(req.Proof, req.TokenID, origin, script.Deadline); err != nil {
		return nil, errors.WithMessagef(err, "failed verifying proof of existence for [%s]", origin)
	}

	// sign
	signer, err := tokn.GetManagementService(context, tokn.WithTMSID(req.OriginTMSID)).SigService().GetSigner(script.Issuer)
	if err != nil {
		return nil, err
	}

	sigma, err := signer.Sign([]byte(script.ID))
	if err != nil {
		return nil, err
	}

	ver, err := tokn.GetManagementService(context, tokn.WithTMSID(req.OriginTMSID)).SigService().OwnerVerifier(script.Issuer)
	if err != nil {
		return nil, err
	}
	if err := ver.Verify([]byte(script.ID), sigma); err != nil {
		return nil, errors.Wrapf(err, "failed to verify issuer signature [%s]", script.Issuer)
	}

	logger.Debugf("produced signature by (me) [%s,%s,%s]",
		hash.Hashable(req.TokenID.String()).String(),
		hash.Hashable(sigma).String(),
		script.Issuer.UniqueID(),
	)
	res := &IssuerApprovalResponse{Signature: sigma}
	resRaw, err := res.Bytes()
	if err != nil {
		return nil, err
	}
	fmt.Printf("sent approval response [%v]\n", resRaw)
	err = s.Send(resRaw)
	if err != nil {
		return nil, err
	}
	fmt.Printf("sent approval response [%v]\n", resRaw)

	return resRaw, nil
}
