/*
Copyright IBM Corp. All Rights Reserved.

SPDX-License-Identifier: Apache-2.0
*/

package ttxcc

import (
	"encoding/base64"
	"strconv"
	"time"

	"github.com/hyperledger-labs/fabric-smart-client/platform/view/services/tracker/metrics"
	"github.com/pkg/errors"
	"go.uber.org/zap/zapcore"

	view2 "github.com/hyperledger-labs/fabric-smart-client/platform/view"
	"github.com/hyperledger-labs/fabric-smart-client/platform/view/services/hash"
	"github.com/hyperledger-labs/fabric-smart-client/platform/view/view"

	"github.com/hyperledger-labs/fabric-token-sdk/token"
	"github.com/hyperledger-labs/fabric-token-sdk/token/services/network"
)

type signatureRequest struct {
	Request []byte
	TxID    []byte
	Signer  view.Identity
}

func (sr *signatureRequest) MessageToSign() []byte {
	return append(sr.Request, sr.TxID...)
}

type collectEndorsementsView struct {
	tx *Transaction
}

// NewCollectEndorsementsView returns an instance of the collectEndorsementsView struct.
// This view does the following:
// 1. It collects all the required signatures
// to authorize any issue and transfer operation contained in the token transaction.
// 2. It invokes the Token Chaincode to collect endorsements on the Token Request and prepare the relative Fabric transaction.
// 3. Before completing, all recipients receive the approved Fabric transaction.
// Depending on the token driver implementation, the recipient's signature might or might not be needed to make
// the token transaction valid.
func NewCollectEndorsementsView(tx *Transaction) *collectEndorsementsView {
	return &collectEndorsementsView{tx: tx}
}

// Call executes the view.
// This view does the following:
// 1. It collects all the required signatures
// to authorize any issue and transfer operation contained in the token transaction.
// 2. It invokes the Token Chaincode to collect endorsements on the Token Request and prepare the relative Fabric transaction.
// 3. Before completing, all recipients receive the approved Fabric transaction.
// Depending on the token driver implementation, the recipient's signature might or might not be needed to make
// the token transaction valid.
func (c *collectEndorsementsView) Call(context view.Context) (interface{}, error) {
	agent := metrics.Get(context)
	agent.EmitKey(0, "ttxcc", "start", "collectEndorsements", c.tx.ID())
	defer agent.EmitKey(0, "ttxcc", "end", "collectEndorsements", c.tx.ID())

	// Store transient
	err := c.tx.storeTransient()
	if err != nil {
		return nil, errors.Wrapf(err, "failed storing transient")
	}

	// 1. First collect signatures on the token request
	var distributionList []view.Identity

	parties, err := c.requestSignaturesOnIssues(context)
	if err != nil {
		return nil, err
	}
	distributionList = append(distributionList, parties...)

	parties, err = c.requestSignaturesOnTransfers(context)
	if err != nil {
		return nil, err
	}
	distributionList = append(distributionList, parties...)

	// 2. Audit
	if !c.tx.Opts.Auditor.IsNone() {
		_, err := context.RunView(newAuditingViewInitiator(c.tx))
		if err != nil {
			return nil, errors.WithMessagef(err, "failed requesting auditing from [%s]", c.tx.Opts.Auditor.String())
		}
		distributionList = append(distributionList, c.tx.Opts.Auditor)
	}

	// 3. Endorse and return the Fabric transaction envelope
	env, err := c.callChaincode(context)
	if err != nil {
		return nil, err
	}

	// Distribute Env to all parties
	if err := c.distributeEnv(context, env, distributionList); err != nil {
		return nil, err
	}

	// Cleanup
	if !c.tx.Opts.Auditor.IsNone() {
		session, err := context.GetSession(nil, c.tx.Opts.Auditor)
		if err != nil {
			return nil, errors.Wrap(err, "failed getting auditor's session")
		}
		session.Close()
	}

	if logger.IsEnabledFor(zapcore.DebugLevel) {
		logger.Debugf("collectEndorsementsView done.")
	}
	return nil, nil
}

func (c *collectEndorsementsView) requestSignaturesOnIssues(context view.Context) ([]view.Identity, error) {

	issues := c.tx.TokenRequest.Issues()
	if logger.IsEnabledFor(zapcore.DebugLevel) {
		logger.Debugf("collecting signature on [%d] request issue", len(issues))
	}

	if len(issues) == 0 {
		return nil, nil
	}

	agent := metrics.Get(context)
	agent.EmitKey(0, "ttxcc", "start", "requestSignaturesOnIssues", c.tx.ID())
	defer agent.EmitKey(0, "ttxcc", "end", "requestSignaturesOnIssues", c.tx.ID())

	requestRaw, err := c.requestBytes()
	if err != nil {
		return nil, err
	}

	var distributionList []view.Identity
	for _, issue := range issues {
		distributionList = append(distributionList, issue.Issuer)
		distributionList = append(distributionList, issue.Receivers...)

		// contact issuer and ask for the signature unless it is me
		party := issue.Issuer
		if logger.IsEnabledFor(zapcore.DebugLevel) {
			logger.Debugf("collecting signature on request (issue) from [%s]", party.UniqueID())
		}
		if signer, err := c.tx.TokenService().SigService().GetSigner(party); err == nil {
			if logger.IsEnabledFor(zapcore.DebugLevel) {
				logger.Debugf("signing [%s][%s]", hash.Hashable(requestRaw).String(), c.tx.ID())
			}
			if logger.IsEnabledFor(zapcore.DebugLevel) {
				logger.Debugf("signing tx-id [%s,nonce=%s]", c.tx.ID(), base64.StdEncoding.EncodeToString(c.tx.TxID.Nonce))
			}
			sigma, err := signer.Sign(append(requestRaw, []byte(c.tx.ID())...))
			if err != nil {
				return nil, err
			}
			c.tx.TokenRequest.AppendSignature(sigma)
			continue
		}

		session, err := context.GetSession(context.Initiator(), party)
		if err != nil {
			return nil, errors.Wrap(err, "failed getting session")
		}
		// Wait to receive a content back
		ch := session.Receive()

		signatureRequest := &signatureRequest{
			Request: requestRaw,
			TxID:    []byte(c.tx.ID()),
			Signer:  party,
		}
		signatureRequestRaw, err := Marshal(signatureRequest)
		if err != nil {
			return nil, err
		}
		err = session.Send(signatureRequestRaw)
		if err != nil {
			return nil, errors.Wrap(err, "failed sending transaction content")
		}

		var msg *view.Message
		select {
		case msg = <-ch:
			if logger.IsEnabledFor(zapcore.DebugLevel) {
				logger.Debugf("collect signatures on issue: reply received from [%s]", party)
			}
		case <-time.After(60 * time.Second):
			return nil, errors.Errorf("Timeout from party %s", party)
		}
		if msg.Status == view.ERROR {
			return nil, errors.New(string(msg.Payload))
		}

		sigma := msg.Payload

		verifier, err := c.tx.TokenService().SigService().IssuerVerifier(party)
		if err != nil {
			return nil, errors.Wrapf(err, "failed getting verifier for [%s]", party)
		}
		err = verifier.Verify(signatureRequest.MessageToSign(), sigma)
		if err != nil {
			return nil, errors.Wrapf(err, "failed verifying signature from [%s]", party)
		}

		c.tx.TokenRequest.AppendSignature(sigma)
	}

	return distributionList, nil
}

func (c *collectEndorsementsView) requestSignaturesOnTransfers(context view.Context) ([]view.Identity, error) {
	transfers := c.tx.TokenRequest.Transfers()
	if logger.IsEnabledFor(zapcore.DebugLevel) {
		logger.Debugf("collecting signature on [%d] request transfer", len(transfers))
	}

	if len(transfers) == 0 {
		return nil, nil
	}

	agent := metrics.Get(context)
	agent.EmitKey(0, "ttxcc", "start", "requestSignaturesOnTransfers", c.tx.ID())
	defer agent.EmitKey(0, "ttxcc", "end", "requestSignaturesOnTransfers", c.tx.ID())

	requestRaw, err := c.requestBytes()
	if err != nil {
		return nil, err
	}

	var distributionList []view.Identity
	for i, transfer := range transfers {
		distributionList = append(distributionList, transfer.Senders...)
		distributionList = append(distributionList, transfer.Receivers...)

		if logger.IsEnabledFor(zapcore.DebugLevel) {
			logger.Debugf("collecting signature on [%d]-th request transfer, signers [%d]", i, len(transfer.Senders))
		}

		// contact transfer and ask for the signature unless it is me
		for _, party := range transfer.Senders {
			signatureRequest := &signatureRequest{
				Request: requestRaw,
				TxID:    []byte(c.tx.ID()),
				Signer:  party,
			}

			if logger.IsEnabledFor(zapcore.DebugLevel) {
				logger.Debugf("collecting signature on request (transfer) from [%s]", party.UniqueID())
			}

			if signer, err := c.tx.TokenService().SigService().GetSigner(party); err == nil {
				if logger.IsEnabledFor(zapcore.DebugLevel) {
					logger.Debugf("collecting signature on request (transfer) from [%s], it is me!", party.UniqueID())
					logger.Debugf("signing tx-id [%s,nonce=%s]", c.tx.ID(), base64.StdEncoding.EncodeToString(c.tx.TxID.Nonce))
				}
				sigma, err := signer.Sign(signatureRequest.MessageToSign())
				if err != nil {
					return nil, err
				}
				if logger.IsEnabledFor(zapcore.DebugLevel) {
					logger.Debugf("signature verified (me) [%s,%s,%s]",
						hash.Hashable(signatureRequest.MessageToSign()).String(),
						hash.Hashable(sigma).String(),
						party.UniqueID(),
					)
				}

				c.tx.TokenRequest.AppendSignature(sigma)
				continue
			}
			if logger.IsEnabledFor(zapcore.DebugLevel) {
				logger.Debugf("collecting signature on request (transfer) from [%s], it is not me, connect to party!", party.UniqueID())
			}

			session, err := context.GetSession(context.Initiator(), party)
			if err != nil {
				return nil, errors.Wrap(err, "failed getting session")
			}
			// Wait to receive a content back
			ch := session.Receive()

			signatureRequestRaw, err := Marshal(signatureRequest)
			if err != nil {
				return nil, err
			}
			err = session.Send(signatureRequestRaw)
			if err != nil {
				return nil, errors.Wrap(err, "failed sending transaction content")
			}

			var msg *view.Message
			select {
			case msg = <-ch:
				if logger.IsEnabledFor(zapcore.DebugLevel) {
					logger.Debugf("collect signatures on transfer: reply received from [%s]", party)
				}
			case <-time.After(60 * time.Second):
				return nil, errors.Errorf("Timeout from party %s", party)
			}
			if msg.Status == view.ERROR {
				return nil, errors.New(string(msg.Payload))
			}

			sigma := msg.Payload

			verifier, err := c.tx.TokenService().SigService().OwnerVerifier(party)
			if err != nil {
				return nil, errors.Wrapf(err, "failed getting verifier for [%s]", party)
			}
			err = verifier.Verify(signatureRequest.MessageToSign(), sigma)
			if err != nil {
				return nil, errors.Wrapf(err, "failed verifying signature from [%s]", party)
			}

			if logger.IsEnabledFor(zapcore.DebugLevel) {
				logger.Debugf("signature verified [%s,%s,%s]",
					hash.Hashable(signatureRequest.MessageToSign()).String(),
					hash.Hashable(sigma).String(),
					party.UniqueID(),
				)
			}

			c.tx.TokenRequest.AppendSignature(sigma)
		}
	}

	return distributionList, nil
}

func (c *collectEndorsementsView) callChaincode(context view.Context) (*network.Envelope, error) {
	agent := metrics.Get(context)
	agent.EmitKey(0, "ttxcc", "start", "callChaincode", c.tx.ID())
	defer agent.EmitKey(0, "ttxcc", "end", "callChaincode", c.tx.ID())

	requestRaw, err := c.tx.TokenRequest.RequestToBytes()
	if err != nil {
		return nil, errors.Wrapf(err, "failed marshalling request")
	}
	agent.EmitKey(0, "ttxcc", "size", "callChaincodeSize", c.tx.ID(), strconv.Itoa(len(requestRaw)))

	if logger.IsEnabledFor(zapcore.DebugLevel) {
		logger.Debugf("call chaincode for endorsement [nonce=%s]", base64.StdEncoding.EncodeToString(c.tx.TxID.Nonce))
	}

	agent.EmitKey(0, "ttxcc", "start", "callChaincodeRequest", c.tx.ID())
	env, err := network.GetInstance(context, c.tx.Network(), c.tx.Channel()).RequestApproval(
		context,
		c.tx.Namespace(),
		requestRaw,
		c.tx.Signer,
		c.tx.Payload.TxID,
	)
	agent.EmitKey(0, "ttxcc", "end", "callChaincodeRequest", c.tx.ID())
	if err != nil {
		return nil, err
	}

	err = c.tx.setEnvelope(env)
	if err != nil {
		return nil, err
	}

	return env, nil
}

func (c *collectEndorsementsView) distributeEnv(context view.Context, env *network.Envelope, distributionList []view.Identity) error {
	agent := metrics.Get(context)
	agent.EmitKey(0, "ttxcc", "start", "distributeEnv", c.tx.ID())
	defer agent.EmitKey(0, "ttxcc", "end", "distributeEnv", c.tx.ID())

	if env == nil {
		return errors.New("fabric transaction envelope is empty")
	}

	// double check that the transaction is valid
	// if err := c.tx.Verify(); err != nil {
	// 	return errors.Wrap(err, "failed verifying transaction content before distributing it")
	// }

	txRaw, err := c.tx.Bytes()
	if err != nil {
		return errors.Wrap(err, "failed marshalling transaction content")
	}
	agent.EmitKey(0, "ttxcc", "size", "distributeEnvSize", c.tx.ID(), strconv.Itoa(len(txRaw)))

	// Compress distributionList
	type distributionListEntry struct {
		IsMe     bool
		LongTerm view.Identity
		ID       view.Identity
	}
	var distributionListCompressed []distributionListEntry
	for _, party := range distributionList {
		if party.IsNone() {
			// In the case of a redeem
			continue
		}
		if logger.IsEnabledFor(zapcore.DebugLevel) {
			logger.Debugf("distribute env to [%s]?", party.UniqueID())
		}
		isMe := c.tx.TokenService().SigService().IsMe(party)
		if logger.IsEnabledFor(zapcore.DebugLevel) {
			logger.Debugf("distribute env to [%s], it is me [%v].", party.UniqueID(), isMe)
		}
		longTermIdentity, _, _, err := view2.GetEndpointService(context).Resolve(party)
		if err != nil {
			return errors.Wrapf(err, "cannot resolve long term identity for [%s]", party.UniqueID())
		}
		if logger.IsEnabledFor(zapcore.DebugLevel) {
			logger.Debugf("searching for long term identity [%s]", longTermIdentity)
		}
		found := false
		for _, entry := range distributionListCompressed {
			if longTermIdentity.Equal(entry.LongTerm) {
				found = true
				break
			}
		}
		if !found {
			if logger.IsEnabledFor(zapcore.DebugLevel) {
				logger.Debugf("adding [%s] to distribution list", party)
			}
			distributionListCompressed = append(distributionListCompressed, distributionListEntry{
				IsMe:     isMe,
				LongTerm: longTermIdentity,
				ID:       party,
			})
		} else {
			if logger.IsEnabledFor(zapcore.DebugLevel) {
				logger.Debugf("skip adding [%s] to distribution list, already added", party)
			}
		}
	}

	if logger.IsEnabledFor(zapcore.DebugLevel) {
		logger.Debugf("distributed env of size [%d] to num parties", len(txRaw), len(distributionListCompressed))
	}

	for _, entry := range distributionListCompressed {
		if logger.IsEnabledFor(zapcore.DebugLevel) {
			logger.Debugf("distribute fabric transaction enveloper to [%s]", entry.ID.UniqueID())
		}

		if entry.IsMe {
			if logger.IsEnabledFor(zapcore.DebugLevel) {
				logger.Debugf("This is me [%s], endorse locally", entry.ID.UniqueID())
			}

			// Inform the vault about the transaction
			ch := network.GetInstance(context, c.tx.Network(), c.tx.Channel())
			rws, err := ch.GetRWSet(c.tx.ID(), env.Results())
			if err != nil {
				return errors.WithMessagef(err, "failed getting rwset for tx [%s]", c.tx.ID())
			}
			rws.Done()

			rawEnv, err := env.Bytes()
			if err != nil {
				return errors.WithMessagef(err, "failed marshalling tx env [%s]", c.tx.ID())
			}
			if err := ch.StoreEnvelope(env.TxID(), rawEnv); err != nil {
				return errors.WithMessagef(err, "failed storing tx env [%s]", c.tx.ID())
			}

			continue
		} else {
			if logger.IsEnabledFor(zapcore.DebugLevel) {
				logger.Debugf("This is not me [%s], ask endorse", entry.ID.UniqueID())
			}
		}

		session, err := context.GetSession(context.Initiator(), entry.ID)
		if err != nil {
			return errors.Wrap(err, "failed getting session")
		}
		// Wait to receive a content back
		ch := session.Receive()

		// Send the content
		err = session.Send(txRaw)
		if err != nil {
			return errors.Wrap(err, "failed sending transaction content")
		}
		agent.EmitKey(0, "ttxcc", "sent", "tx", c.tx.ID())

		var msg *view.Message
		select {
		case msg = <-ch:
			if logger.IsEnabledFor(zapcore.DebugLevel) {
				logger.Debugf("collect ack on distributed env: reply received from [%s]", entry.ID)
			}
		case <-time.After(240 * time.Second):
			return errors.Errorf("Timeout from party %s", entry.ID)
		}
		if msg.Status == view.ERROR {
			return errors.New(string(msg.Payload))
		}
		// TODO: Check ack
		agent.EmitKey(0, "ttxcc", "received", "txAck", c.tx.ID())

		if logger.IsEnabledFor(zapcore.DebugLevel) {
			logger.Debugf("collectEndorsementsView: collected signature from %s", entry.ID)
		}
	}

	return nil
}

func (c *collectEndorsementsView) requestBytes() ([]byte, error) {
	return c.tx.TokenRequest.MarshallToSign()
}

type receiveTransactionView struct {
	network string
	channel string
}

func NewReceiveTransactionView(network string) *receiveTransactionView {
	return &receiveTransactionView{network: network}
}

func (f *receiveTransactionView) Call(context view.Context) (interface{}, error) {
	// Wait to receive a transaction back
	ch := context.Session().Receive()

	select {
	case msg := <-ch:
		if msg.Status == view.ERROR {
			return nil, errors.New(string(msg.Payload))
		}
		tx, err := NewTransactionFromBytes(context, f.network, f.channel, msg.Payload)
		if err != nil {
			return nil, err
		}
		return tx, nil
	case <-time.After(240 * time.Second):
		return nil, errors.New("timeout reached")
	}
}

type endorseView struct {
	tx *Transaction
}

// NewEndorseView returns an instance of the endorseView.
// The view does the following:
// 1. Wait for signature requests.
// 2. Upon receiving a signature request, it validates the request and send back the requested signature.
// 3. After, it waits to receive the Fabric Transaction. The Fabric Transaction is validated and stored locally
// to be processed at time of committing.
// 4. It sends back an ack.
func NewEndorseView(tx *Transaction) *endorseView {
	return &endorseView{tx: tx}
}

// Call executes the view.
// The view does the following:
// 1. Wait for signature requests.
// 2. Upon receiving a signature request, it validates the request and send back the requested signature.
// 3. After, it waits to receive the Fabric Transaction. The Fabric Transaction is validated and stored locally
// to be processed at time of committing.
// 4. It sends back an ack.
func (s *endorseView) Call(context view.Context) (interface{}, error) {
	// Process signature requests
	requestsToBeSigned, err := s.requestsToBeSigned()
	if err != nil {
		return nil, errors.Wrapf(err, "failed collecting requests of signature")
	}

	session := context.Session()
	for range requestsToBeSigned {
		if logger.IsEnabledFor(zapcore.DebugLevel) {
			logger.Debugf("Receiving signature request...")
		}
		sessionChannel := session.Receive()
		var msg *view.Message
		select {
		case msg = <-sessionChannel:
			logger.Debug("message received from %s", session.Info().Caller)
		case <-time.After(60 * time.Second):
			return nil, errors.Errorf("Timeout from party %s", session.Info().Caller)
		}
		if msg.Status == view.ERROR {
			return nil, errors.New(string(msg.Payload))
		}

		// TODO: check what is signed...
		signatureRequest := &signatureRequest{}
		err := Unmarshal(msg.Payload, signatureRequest)
		if err != nil {
			return nil, errors.Wrap(err, "failed unmarshalling signature request")
		}

		tms := token.GetManagementService(context, token.WithTMS(s.tx.Network(), s.tx.Channel(), s.tx.Namespace()))
		if tms == nil {
			return nil, errors.Errorf("failed getting TMS for [%s:%s:%s]", s.tx.Network(), s.tx.Channel(), s.tx.Namespace())
		}
		if !tms.WalletManager().IsMe(signatureRequest.Signer) {
			return nil, errors.Errorf("identity [%s] is not me", signatureRequest.Signer.UniqueID())
		}
		signer, err := s.tx.TokenService().SigService().GetSigner(signatureRequest.Signer)
		if err != nil {
			return nil, errors.Wrapf(err, "cannot find signer for [%s]", signatureRequest.Signer.UniqueID())
		}
		sigma, err := signer.Sign(signatureRequest.MessageToSign())
		if err != nil {
			return nil, errors.Wrapf(err, "failed signing request")
		}
		if logger.IsEnabledFor(zapcore.DebugLevel) {
			logger.Debugf("Send back signature...")
		}
		err = session.Send(sigma)
		if err != nil {
			return nil, errors.Wrapf(err, "failed sending signature back")
		}
	}

	// Receive transaction with envelope
	if logger.IsEnabledFor(zapcore.DebugLevel) {
		logger.Debugf("Receive transaction with envelope...")
	}
	// TODO: this might also happen multiple times because of the pseudonym. Avoid this by identity resolution at the sender
	tx, err := ReceiveTransaction(context)
	if err != nil {
		return nil, errors.Wrapf(err, "failed receiving transaction")
	}
	agent := metrics.Get(context)
	agent.EmitKey(0, "ttxcc", "received", "env", tx.ID())

	// Process Fabric Envelope
	if logger.IsEnabledFor(zapcore.DebugLevel) {
		logger.Debugf("Processes Fabric Envelope with ID [%s]", tx.ID())
	}
	env := tx.Payload.Envelope
	if env == nil {
		return nil, errors.Errorf("expected fabric envelope")
	}

	err = tx.storeTransient()
	if err != nil {
		return nil, errors.Wrapf(err, "failed storing transient")
	}

	ch := network.GetInstance(context, tx.Network(), tx.Channel())
	rws, err := ch.GetRWSet(tx.ID(), env.Results())
	if err != nil {
		return nil, errors.WithMessagef(err, "failed getting rwset for tx [%s]", tx.ID())
	}
	rws.Done()

	// TODO: remove this
	rawEnv, err := env.Bytes()
	if err != nil {
		return nil, errors.WithMessagef(err, "failed marshalling tx env [%s]", tx.ID())
	}
	if err := ch.StoreEnvelope(env.TxID(), rawEnv); err != nil {
		return nil, errors.WithMessagef(err, "failed storing tx env [%s]", tx.ID())
	}

	// Send the proposal response back
	if logger.IsEnabledFor(zapcore.DebugLevel) {
		logger.Debugf("Send the ack")
	}
	err = session.Send([]byte("ack"))
	if err != nil {
		return nil, err
	}
	agent.EmitKey(0, "ttxcc", "sent", "txAck", tx.ID())

	return tx, nil
}

func (s *endorseView) requestsToBeSigned() ([]*token.Transfer, error) {
	var res []*token.Transfer
	for _, transfer := range s.tx.TokenRequest.Transfers() {
		for _, sender := range transfer.Senders {
			if _, err := s.tx.TokenService().SigService().GetSigner(sender); err == nil {
				res = append(res, transfer)
			}
		}
	}
	return res, nil
}
