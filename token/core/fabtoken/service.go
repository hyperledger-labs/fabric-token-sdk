/*
Copyright IBM Corp. All Rights Reserved.

SPDX-License-Identifier: Apache-2.0
*/
package fabtoken

import (
	"encoding/base64"
	"encoding/json"
	"github.com/golang/protobuf/proto"
	"sync"

	token2 "github.com/hyperledger-labs/fabric-token-sdk/token/token"

	"github.com/pkg/errors"

	"github.com/hyperledger-labs/fabric-smart-client/platform/fabric"
	view2 "github.com/hyperledger-labs/fabric-smart-client/platform/view"
	"github.com/hyperledger-labs/fabric-smart-client/platform/view/services/hash"
	"github.com/hyperledger-labs/fabric-smart-client/platform/view/view"
	"github.com/hyperledger-labs/fabric-token-sdk/token/driver"
	"github.com/hyperledger-labs/fabric-token-sdk/token/services/vault/keys"
)

type Channel interface {
	Name() string
	Vault() *fabric.Vault
}

type PublicParamsLoader interface {
	Load() (*PublicParams, error)
}

type QueryEngine interface {
	IsMine(id *token2.Id) (bool, error)
	ListUnspentTokens() (*token2.UnspentTokens, error)
	ListAuditTokens(ids ...*token2.Id) ([]*token2.Token, error)
	ListHistoryIssuedTokens() (*token2.IssuedTokens, error)
	PublicParams() ([]byte, error)
}

type service struct {
	sp                  view2.ServiceProvider
	channel             Channel
	namespace           string
	pp                  *PublicParams
	publicParamsFetcher driver.PublicParamsFetcher
	publicParamsLoader  PublicParamsLoader
	qe                  QueryEngine

	identityProvider driver.IdentityProvider
	ownerWallets     []*ownerWallet
	issuerWallets    []*issuerWallet
	auditorWallets   []*auditorWallet
	walletsLock      sync.Mutex
}

func NewService(
	sp view2.ServiceProvider,
	channel Channel,
	namespace string,
	publicParamsFetcher driver.PublicParamsFetcher,
	publicParamsLoader PublicParamsLoader,
	qe QueryEngine,
	identityProvider driver.IdentityProvider,
) *service {
	return &service{
		sp:                  sp,
		channel:             channel,
		namespace:           namespace,
		publicParamsFetcher: publicParamsFetcher,
		publicParamsLoader:  publicParamsLoader,
		qe:                  qe,
		identityProvider:    identityProvider,
	}
}

func (s *service) PublicParams() interface{} {
	if s.pp == nil {
		var err error
		s.pp, err = s.publicParamsLoader.Load()
		if err != nil {
			panic(err)
		}
	}
	return s.pp
}

func (s *service) FetchPublicParams() error {
	raw, err := s.publicParamsFetcher.Fetch()
	if err != nil {
		return errors.WithMessagef(err, "failed fetching public params from fabric")
	}

	pp := &PublicParams{}
	err = pp.Deserialize(raw)
	if err != nil {
		return errors.Wrapf(err, "failed deserializing public params")
	}

	s.pp = pp
	return nil
}

func (s *service) RegisterRecipientIdentity(id view.Identity, auditInfo []byte, metadata []byte) error {
	logger.Debugf("register recipient identity [%s] with audit info [%s]", id.String(), hash.Hashable(auditInfo).String())

	// recognize identity and register it
	_, err := view2.GetSigService(s.sp).GetVerifier(id)
	if err != nil {
		return err
	}

	if err := view2.GetSigService(s.sp).RegisterAuditInfo(id, auditInfo); err != nil {
		return err
	}

	return nil
}

func (s *service) GenerateIssuerKeyPair(tokenType string) (driver.Key, driver.Key, error) {
	panic("implement me")
}

func (s *service) RegisterAuditInfo(id view.Identity, auditInfo []byte) error {
	if err := view2.GetSigService(s.sp).RegisterAuditInfo(id, auditInfo); err != nil {
		return err
	}
	return nil
}

func (s *service) RegisterIssuer(label string, sk driver.Key, pk driver.Key) error {
	panic("implement me")
}

func (s *service) IssuerIdentity(label string) (view.Identity, error) {
	panic("implement me")
}

func (s *service) GetAuditInfo(id view.Identity) ([]byte, error) {
	return view2.GetSigService(s.sp).GetAuditInfo(id)
}

func (s *service) GetEnrollmentID(auditInfo []byte) (string, error) {
	return string(auditInfo), nil
}

func (s *service) Issue(issuerIdentity view.Identity, typ string, values []uint64, owners [][]byte) (driver.IssueAction, [][]byte, view.Identity, error) {
	for _, owner := range owners {
		if len(owner) == 0 {
			return nil, nil, nil, errors.Errorf("all recipients should be defined")
		}
	}

	var outs []*TransferOutput
	var infos [][]byte
	for i, v := range values {
		ro := &RawOwner{Type: SerializedIdentityType, Identity: owners[i]}
		rawOwner, err := proto.Marshal(ro)
		if err != nil {
			return nil, nil, nil, err
		}
		outs = append(outs, &TransferOutput{
			Output: &token2.Token{
				Owner: &token2.Owner{
					Raw: rawOwner,
				},
				Type:     typ,
				Quantity: token2.NewQuantityFromUInt64(v).Hex(),
			},
		})

		ti := &TokenInformation{
			Issuer: issuerIdentity,
		}
		tiRaw, err := ti.Serialize()
		if err != nil {
			return nil, nil, nil, errors.Wrapf(err, "failed serializing token information")
		}
		infos = append(infos, tiRaw)
	}

	return &IssueAction{Issuer: issuerIdentity, Outputs: outs},
		infos,
		issuerIdentity,
		nil
}

func (s *service) VerifyIssue(tr driver.IssueAction, tokenInfos [][]byte) error {
	// TODO:
	return nil
}

func (s *service) DeserializeIssueAction(raw []byte) (driver.IssueAction, error) {
	issue := &IssueAction{}
	if err := issue.Deserialize(raw); err != nil {
		return nil, errors.Wrap(err, "failed deserializing issue action")
	}
	return issue, nil
}

func (s *service) Transfer(txID string, wallet driver.OwnerWallet, ids []*token2.Id, Outputs ...*token2.Token) (driver.TransferAction, *driver.TransferMetadata, error) {
	id, err := wallet.GetRecipientIdentity()
	if err != nil {
		return nil, nil, errors.WithMessagef(err, "failed getting sender identity")
	}

	var tokens []*token2.Token
	var inputIDs []string
	var signerIds []view.Identity

	qe, err := s.channel.Vault().NewQueryExecutor()
	if err != nil {
		return nil, nil, err
	}
	defer qe.Done()

	for _, id := range ids {
		// Token and InputID
		outputID, err := keys.CreateTokenKey(id.TxId, int(id.Index))
		if err != nil {
			return nil, nil, errors.Wrapf(err, "error creating output ID: %v", id)
		}
		val, err := qe.GetState(s.namespace, outputID)
		if err != nil {
			return nil, nil, errors.Wrapf(err, "failed getting state [%s]", outputID)
		}

		logger.Debugf("loaded transfer input [%s]", hash.Hashable(val).String())
		tok := &token2.Token{}
		err = json.Unmarshal(val, tok)
		if err != nil {
			return nil, nil, errors.Wrapf(err, "failed unmarshalling token for id [%v]", id)
		}

		ro := &RawOwner{}
		if err := proto.Unmarshal(tok.Owner.Raw, ro); err != nil {
			return nil, nil, errors.Errorf( "failed deserializing owner [%d][%s][%s]", id.Index, id.TxId, base64.StdEncoding.EncodeToString(tok.Owner.Raw))
		}

		if ro.Type != SerializedIdentityType {
			return nil, nil, errors.Errorf("unknown Owner type (%s), expected '%s'", ro.Type, SerializedIdentityType)
		}

		logger.Debugf("Selected output [%s,%s,%s]", tok.Type, tok.Quantity, view.Identity(ro.Identity))

		// Signer
		si, err := view2.GetSigService(s.sp).GetSigningIdentity(ro.Identity)
		if err != nil {
			return nil, nil, errors.Wrapf(err, "failed getting signing identity for id [%v]", id)
		}
		ser, err := si.Serialize()
		if err != nil {
			return nil, nil, errors.Wrapf(err, "failed serializing signing identity for id [%v]", id)
		}

		inputIDs = append(inputIDs, outputID)
		tokens = append(tokens, tok)
		signerIds = append(signerIds, ser)
	}

	var outs []*TransferOutput
	var infos [][]byte
	for _, output := range Outputs {
		outs = append(outs, &TransferOutput{
			Output: output,
		})
		ti := &TokenInformation{}
		tiRaw, err := ti.Serialize()
		if err != nil {
			return nil, nil, errors.Wrapf(err, "failed serializing token information")
		}
		infos = append(infos, tiRaw)
	}

	transfer := &TransferAction{
		Sender:  id,
		Inputs:  inputIDs,
		Outputs: outs,
	}

	var ownerIdentities []view.Identity
	for _, output := range outs {
		// add owner identity if not present already
		found := false
		for _, identity := range ownerIdentities {
			if identity.Equal(output.Output.Owner.Raw) {
				found = true
				break
			}
		}
		if !found {
			ownerIdentities = append(ownerIdentities, output.Output.Owner.Raw)
		}
	}
	var senderAuditInfos [][]byte
	for _, t := range tokens {
		auditInfo, err := view2.GetSigService(s.sp).GetAuditInfo(t.Owner.Raw)
		if err != nil {
			return nil, nil, errors.Wrapf(err, "failed getting audit info for sender identity [%s]", view.Identity(t.Owner.Raw).String())
		}
		senderAuditInfos = append(senderAuditInfos, auditInfo)
	}
	var receiverAuditInfos [][]byte
	for _, output := range outs {
		auditInfo, err := view2.GetSigService(s.sp).GetAuditInfo(output.Output.Owner.Raw)
		if err != nil {
			return nil, nil, errors.Wrapf(err, "failed getting audit info for recipient identity [%s]", view.Identity(output.Output.Owner.Raw).String())
		}
		receiverAuditInfos = append(receiverAuditInfos, auditInfo)
	}
	outputs, err := transfer.GetSerializedOutputs()
	if err != nil {
		return nil, nil, errors.Wrapf(err, "failed getting serialized outputs")
	}

	receiverIsSender := make([]bool, len(ownerIdentities))
	for i, receiver := range ownerIdentities {
		receiverIsSender[i] = s.ownerWallet(receiver) != nil
	}

	metadata := &driver.TransferMetadata{
		Outputs:            outputs,
		Senders:            signerIds,
		SenderAuditInfos:   senderAuditInfos,
		TokenIDs:           ids,
		TokenInfo:          infos,
		Receivers:          ownerIdentities,
		ReceiverIsSender:   receiverIsSender,
		ReceiverAuditInfos: receiverAuditInfos,
	}

	return transfer, metadata, nil
}

func (s *service) VerifyTransfer(tr driver.TransferAction, tokenInfos [][]byte) error {
	// TODO:
	return nil
}

func (s *service) DeserializeTransferAction(raw []byte) (driver.TransferAction, error) {
	t := &TransferAction{}
	if err := t.Deserialize(raw); err != nil {
		return nil, errors.Wrap(err, "failed deserializing transfer action")
	}
	return t, nil
}

func (s *service) ListTokens() (*token2.UnspentTokens, error) {
	return s.qe.ListUnspentTokens()
}

func (s *service) HistoryIssuedTokens() (*token2.IssuedTokens, error) {
	return s.qe.ListHistoryIssuedTokens()
}

func (s *service) DeserializeToken(outputRaw []byte, tokenInfoRaw []byte) (*token2.Token, view.Identity, error) {
	tok := &token2.Token{}
	if err := json.Unmarshal(outputRaw, tok); err != nil {
		return nil, nil, errors.Wrap(err, "failed unmarshalling token")
	}

	tokInfo := &TokenInformation{}
	if err := tokInfo.Deserialize(tokenInfoRaw); err != nil {
		return nil, nil, errors.Wrap(err, "failed unmarshalling token information")
	}

	return tok, tokInfo.Issuer, nil
}

func (s *service) AuditorCheck(tokenRequest *driver.TokenRequest, tokenRequestMetadata *driver.TokenRequestMetadata, txID string) error {
	// TODO:
	return nil
}

func (s *service) IdentityProvider() driver.IdentityProvider {
	return s.identityProvider
}

func (s *service) Validator() driver.Validator {
	return NewValidator(s.publicParams())
}

func (s *service) PublicParamsManager() driver.PublicParamsManager {
	return NewPublicParamsManager(s.publicParams())
}

func (s *service) NewCertificationRequest(ids []*token2.Id) ([]byte, error) {
	return nil, nil
}

func (s *service) Certify(wallet driver.CertifierWallet, ids []*token2.Id, tokens [][]byte, request []byte) ([][]byte, error) {
	return nil, nil
}

func (s *service) VerifyCertifications(ids []*token2.Id, certifications [][]byte) error {
	return nil
}

func (s *service) publicParams() *PublicParams {
	s.PublicParams()
	return s.pp
}
