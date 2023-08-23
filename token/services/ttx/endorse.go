/*
Copyright IBM Corp. All Rights Reserved.

SPDX-License-Identifier: Apache-2.0
*/

package ttx

import (
	"encoding/base64"
	"time"

	view2 "github.com/hyperledger-labs/fabric-smart-client/platform/view"
	"github.com/hyperledger-labs/fabric-smart-client/platform/view/services/hash"
	"github.com/hyperledger-labs/fabric-smart-client/platform/view/services/kvs"
	"github.com/hyperledger-labs/fabric-smart-client/platform/view/view"
	"github.com/hyperledger-labs/fabric-token-sdk/token"
	"github.com/hyperledger-labs/fabric-token-sdk/token/services/network"
	"github.com/hyperledger-labs/fabric-token-sdk/token/services/vault/keys"
	"github.com/pkg/errors"
	"go.uber.org/zap/zapcore"
)

type ExternalWalletSigner interface {
	Sign(party view.Identity, message []byte) ([]byte, error)
	Done() error
}

type verifierGetterFunc func(identity view.Identity) (token.Verifier, error)

type SignatureRequest struct {
	TX      []byte
	Request []byte
	TxID    []byte
	Signer  view.Identity
}

func (sr *SignatureRequest) MessageToSign() []byte {
	return append(sr.Request, sr.TxID...)
}

type CollectEndorsementsView struct {
	tx       *Transaction
	Opts     *EndorsementsOpts
	sessions map[string]view.Session
}

// NewCollectEndorsementsView returns an instance of the CollectEndorsementsView struct.
// This view does the following:
// 1. It collects all the required signatures
// to authorize any issue and transfer operation contained in the token transaction.
// 2. It invokes the Token Chaincode to collect endorsements on the Token Request and prepare the relative transaction.
// 3. Before completing, all recipients receive the approved transaction.
// Depending on the token driver implementation, the recipient's signature might or might not be needed to make
// the token transaction valid.
func NewCollectEndorsementsView(tx *Transaction, opts ...EndorsementsOpt) *CollectEndorsementsView {
	options, err := CompileCollectEndorsementsOpts(opts...)
	if err != nil {
		panic(err)
	}
	return &CollectEndorsementsView{tx: tx, Opts: options, sessions: map[string]view.Session{}}
}

// Call executes the view.
// This view does the following:
// 1. It collects all the required signatures
// to authorize any issue and transfer operation contained in the token transaction.
// 2. It invokes the Token Chaincode to collect endorsements on the Token Request and prepare the relative transaction.
// 3. Before completing, all recipients receive the approved transaction.
// Depending on the token driver implementation, the recipient's signature might or might not be needed to make
// the token transaction valid.
func (c *CollectEndorsementsView) Call(context view.Context) (interface{}, error) {
	metrics := GetMetrics(context)

	// Store transient
	err := c.tx.storeTransient()
	if err != nil {
		return nil, errors.WithMessagef(err, "failed storing transient")
	}

	externalWallets := make(map[string]ExternalWalletSigner)
	// 1. First collect signatures on the token request
	issueSigmas, err := c.requestSignaturesOnIssues(context, externalWallets)
	if err != nil {
		return nil, errors.WithMessage(err, "failed requesting signatures on issues")
	}

	transferSigmas, err := c.requestSignaturesOnTransfers(context, externalWallets)
	if err != nil {
		return nil, errors.WithMessage(err, "failed requesting signatures on transfers")
	}

	// signal the external wallets that the process is completed
	for id, signer := range externalWallets {
		if err := signer.Done(); err != nil {
			logger.Errorf("failed to signal done external wallet [%s]", id)
		}
	}

	skippedSigmas := c.Opts.SkippedIds() //TODO: SkippedSigmas need to be here because of the next steps. To parallelize, we need to split the operations into two different views

	c.tx.TokenRequest.PutSignatures(mergeSigmas(issueSigmas, transferSigmas, skippedSigmas))

	// 2. Audit
	var auditors []view.Identity
	if !c.Opts.SkipAuditing {
		auditors, err = c.requestAudit(context)
		if err != nil {
			return nil, errors.WithMessage(err, "failed requesting auditing")
		}
	}

	// 3. Endorse and return the transaction envelope
	var env *network.Envelope
	if !c.Opts.SkipApproval {
		env, err = c.requestApproval(context)
		if err != nil {
			return nil, errors.WithMessage(err, "failed requesting approval")
		}
	}

	// Distribute Env to all parties
	distributionList := append(IssueDistributionList(c.tx.TokenRequest), TransferDistributionList(c.tx.TokenRequest)...)
	if err := c.distributeEnv(context, env, distributionList, auditors); err != nil {
		return nil, errors.WithMessage(err, "failed distributing envelope")
	}

	// Cleanup audit
	if err := c.cleanupAudit(context); err != nil {
		return nil, errors.WithMessage(err, "failed cleaning up audit")
	}

	if logger.IsEnabledFor(zapcore.DebugLevel) {
		logger.Debugf("CollectEndorsementsView done.")
	}

	labels := []string{
		"network", c.tx.Network(),
		"channel", c.tx.Channel(),
		"namespace", c.tx.Namespace(),
	}
	metrics.EndorsedTransactions.With(labels...).Add(1)
	return nil, nil
}

func (c *CollectEndorsementsView) requestSignaturesOnIssues(context view.Context, externalWallets map[string]ExternalWalletSigner) (map[string][]byte, error) {
	if logger.IsEnabledFor(zapcore.DebugLevel) {
		logger.Debugf("collecting signature on [%d] request issue", len(c.tx.TokenRequest.Metadata.Issues))
	}
	return c.requestSignatures(c.tx.TokenRequest.IssueSigners(), c.tx.TokenService().SigService().IssuerVerifier, context, externalWallets)
}

func (c *CollectEndorsementsView) requestSignaturesOnTransfers(context view.Context, externalWallets map[string]ExternalWalletSigner) (map[string][]byte, error) {
	if logger.IsEnabledFor(zapcore.DebugLevel) {
		logger.Debugf("collecting signature on [%d] request transfer", len(c.tx.TokenRequest.Metadata.Transfers))
	}

	return c.requestSignatures(c.tx.TokenRequest.TransferSigners(), c.tx.TokenService().SigService().OwnerVerifier, context, externalWallets)
}

func (c *CollectEndorsementsView) requestSignatures(signers []view.Identity, verifierGetter verifierGetterFunc, context view.Context, externalWallets map[string]ExternalWalletSigner) (map[string][]byte, error) {
	requestRaw, err := c.requestBytes()
	if err != nil {
		return nil, err
	}
	txRaw, err := c.tx.Bytes()
	if err != nil {
		return nil, err
	}

	skippedIds := c.Opts.SkippedIds()
	sigmas := make(map[string][]byte)
	for _, party := range signers {
		if _, ok := skippedIds[party.UniqueID()]; ok {
			continue
		}

		signatureRequest := &SignatureRequest{
			TX:      txRaw,
			Request: requestRaw,
			TxID:    []byte(c.tx.ID()),
			Signer:  party,
		}
		if logger.IsEnabledFor(zapcore.DebugLevel) {
			logger.Debugf("collecting signature on request from [%s]", party.UniqueID())
		}

		// 3 possibilities here:
		// 1. there is a signer locally bound to the party, use it to generate the signature
		// 2. there is a wallet bound to the party but the signer is not local, the signature is generated externally
		// 3. the signature must be generated by a remote party

		// Case 1:
		if signer, err := c.tx.TokenService().SigService().GetSigner(party); err == nil {
			if logger.IsEnabledFor(zapcore.DebugLevel) {
				logger.Debugf("found signer for party [%s], request local signature", party)
			}
			sigma, err := c.signLocal(party, signer, signatureRequest)
			if err != nil {
				return nil, err
			}
			sigmas[party.UniqueID()] = sigma
			continue
		}

		// Case 2:
		if w := c.tx.TokenService().WalletManager().OwnerWalletByIdentity(party); w != nil {
			if logger.IsEnabledFor(zapcore.DebugLevel) {
				logger.Debugf("found wallet for party [%s], request external signature", party)
			}
			ews := c.Opts.ExternalWalletSigner(w.ID())
			if ews == nil {
				return nil, errors.Errorf("no external wallet signer found for [%s]", w.ID())
			}
			externalWallets[w.ID()] = ews
			sigma, err := c.signExternal(party, ews, signatureRequest)
			if err != nil {
				return nil, err
			}
			sigmas[party.UniqueID()] = sigma
			continue
		}

		// Case 3:
		if logger.IsEnabledFor(zapcore.DebugLevel) {
			logger.Debugf("no signer or wallet found for party [%s], request remote signature", party)
		}
		sigma, err := c.signRemote(context, party, signatureRequest, verifierGetter)
		if err != nil {
			return nil, err
		}
		sigmas[party.UniqueID()] = sigma
	}

	return sigmas, nil
}

func (c *CollectEndorsementsView) signLocal(party view.Identity, signer token.Signer, signatureRequest *SignatureRequest) ([]byte, error) {
	if logger.IsEnabledFor(zapcore.DebugLevel) {
		logger.Debugf("signing [%s][%s]", hash.Hashable(signatureRequest.Request).String(), c.tx.ID())
		logger.Debugf("signing tx-id [%s,nonce=%s]", c.tx.ID(), base64.StdEncoding.EncodeToString(c.tx.TxID.Nonce))
	}
	sigma, err := signer.Sign(signatureRequest.MessageToSign())
	if err != nil {
		return nil, err
	}
	if logger.IsEnabledFor(zapcore.DebugLevel) {
		logger.Debugf("signature generated (me) [%s,%s,%s]",
			hash.Hashable(signatureRequest.MessageToSign()).String(),
			hash.Hashable(sigma).String(),
			party.UniqueID(),
		)
	}
	return sigma, nil
}

func (c *CollectEndorsementsView) signExternal(party view.Identity, signer ExternalWalletSigner, signatureRequest *SignatureRequest) ([]byte, error) {
	if logger.IsEnabledFor(zapcore.DebugLevel) {
		logger.Debugf("signing [%s][%s]", hash.Hashable(signatureRequest.Request).String(), c.tx.ID())
		logger.Debugf("signing tx-id [%s,nonce=%s]", c.tx.ID(), base64.StdEncoding.EncodeToString(c.tx.TxID.Nonce))
	}
	sigma, err := signer.Sign(party, signatureRequest.MessageToSign())
	if err != nil {
		return nil, err
	}
	if logger.IsEnabledFor(zapcore.DebugLevel) {
		logger.Debugf("signature generated (me) [%s,%s,%s]",
			hash.Hashable(signatureRequest.MessageToSign()).String(),
			hash.Hashable(sigma).String(),
			party.UniqueID(),
		)
	}
	return sigma, nil
}

func (c *CollectEndorsementsView) signRemote(context view.Context, party view.Identity, signatureRequest *SignatureRequest, verifierGetter verifierGetterFunc) ([]byte, error) {
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

	timeout := time.NewTimer(time.Minute)

	var msg *view.Message
	select {
	case msg = <-ch:
		if logger.IsEnabledFor(zapcore.DebugLevel) {
			logger.Debugf("collect signatures: reply received from [%s]", party)
		}
		timeout.Stop()
	case <-timeout.C:
		timeout.Stop()
		return nil, errors.Errorf("Timeout from party %s", party)
	}
	if msg.Status == view.ERROR {
		return nil, errors.New(string(msg.Payload))
	}

	sigma := msg.Payload

	verifier, err := verifierGetter(party)
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

	return sigma, nil
}

func (c *CollectEndorsementsView) requestApproval(context view.Context) (*network.Envelope, error) {
	requestRaw, err := c.tx.TokenRequest.RequestToBytes()
	if err != nil {
		return nil, errors.Wrapf(err, "failed marshalling request")
	}

	if logger.IsEnabledFor(zapcore.DebugLevel) {
		logger.Debugf("call chaincode for endorsement [nonce=%s]", base64.StdEncoding.EncodeToString(c.tx.TxID.Nonce))
	}

	env, err := network.GetInstance(context, c.tx.Network(), c.tx.Channel()).RequestApproval(
		context,
		c.tx.TokenRequest.TokenService,
		requestRaw,
		c.tx.Signer,
		c.tx.Payload.TxID,
	)
	if err != nil {
		return nil, err
	}

	err = c.tx.setEnvelope(env)
	if err != nil {
		return nil, err
	}

	return env, nil
}

func (c *CollectEndorsementsView) requestAudit(context view.Context) ([]view.Identity, error) {
	if !c.tx.Opts.Auditor.IsNone() {
		local := view2.GetSigService(context).IsMe(c.tx.Opts.Auditor)
		sessionBoxed, err := context.RunView(newAuditingViewInitiator(c.tx, local))
		if err != nil {
			return nil, errors.WithMessagef(err, "failed requesting auditing from [%s]", c.tx.Opts.Auditor.String())
		}
		c.sessions[c.tx.Opts.Auditor.String()] = sessionBoxed.(view.Session)
		return []view.Identity{c.tx.Opts.Auditor}, nil
	}
	return nil, nil
}

func (c *CollectEndorsementsView) cleanupAudit(context view.Context) error {
	if !c.tx.Opts.Auditor.IsNone() {
		session, err := c.getSession(context, c.tx.Opts.Auditor)
		if err != nil {
			return errors.Wrap(err, "failed getting auditor's session")
		}
		session.Close()
	}
	return nil
}

func (c *CollectEndorsementsView) distributeEnv(context view.Context, env *network.Envelope, distributionList []view.Identity, auditors []view.Identity) error {
	if !c.Opts.SkipApproval {
		// perform sanity checks
		if env == nil {
			return errors.New("transaction envelope is empty")
		}
		if logger.IsEnabledFor(zapcore.DebugLevel) {
			logger.Debugf("distribute env [%s]", env.String())
		}
	}

	// double check that the transaction is valid
	// if err := c.tx.IsValid(); err != nil {
	// 	return errors.Wrap(err, "failed verifying transaction content before distributing it")
	// }

	// Compress distributionList by removing duplicates
	type distributionListEntry struct {
		IsMe     bool
		LongTerm view.Identity
		ID       view.Identity
		EID      string
		Auditor  bool
	}
	skippedIds := c.Opts.SkippedIds()
	var distributionListCompressed []distributionListEntry
	for _, party := range distributionList {
		if _, ok := skippedIds[party.UniqueID()]; ok {
			continue
		}
		// For each party in the distribution list:
		// - check if it is me
		// - check if it is an auditor
		// - extract the corresponding long term identity
		// If the long term identity has not been added yet, add it to the list.
		// If the party is me or an auditor, no need to extract the enrollment ID.
		if party.IsNone() {
			// This is a redeem, nothing to do here.
			continue
		}
		if logger.IsEnabledFor(zapcore.DebugLevel) {
			logger.Debugf("distribute env to [%s]?", party.UniqueID())
		}
		isMe := c.tx.TokenService().SigService().IsMe(party)
		if !isMe {
			// check if there is a wallet that contains that identity
			isMe = c.tx.TokenService().WalletManager().OwnerWalletByIdentity(party) != nil
		}
		if logger.IsEnabledFor(zapcore.DebugLevel) {
			logger.Debugf("distribute env to [%s], it is me [%v].", party.UniqueID(), isMe)
		}
		var longTermIdentity view.Identity
		var err error
		// if it is me, no need to resolve, get directly the default identity
		if isMe {
			longTermIdentity = view2.GetIdentityProvider(context).DefaultIdentity()
		} else {
			longTermIdentity, _, _, err = view2.GetEndpointService(context).Resolve(party)
			if err != nil {
				return errors.Wrapf(err, "cannot resolve long term identity for [%s]", party.UniqueID())
			}
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
			eID := ""
			if !isMe {
				eID, err = c.tx.TokenService().WalletManager().GetEnrollmentID(party)
				if err != nil {
					return errors.Wrapf(err, "failed getting enrollment ID for [%s]", party.UniqueID())
				}
			}
			distributionListCompressed = append(distributionListCompressed, distributionListEntry{
				IsMe:     isMe,
				LongTerm: longTermIdentity,
				ID:       party,
				EID:      eID,
				Auditor:  false,
			})
		} else {
			if logger.IsEnabledFor(zapcore.DebugLevel) {
				logger.Debugf("skip adding [%s] to distribution list, already added", party)
			}
		}
	}

	// check the auditors
	for _, party := range auditors {
		isMe := c.tx.TokenService().SigService().IsMe(party)
		if logger.IsEnabledFor(zapcore.DebugLevel) {
			logger.Debugf("distribute env to auditor [%s], it is me [%v].", party.UniqueID(), isMe)
		}
		var longTermIdentity view.Identity
		var err error
		// if it is me, no need to resolve, get directly the default identity
		if isMe {
			longTermIdentity = view2.GetIdentityProvider(context).DefaultIdentity()
		} else {
			longTermIdentity, _, _, err = view2.GetEndpointService(context).Resolve(party)
			if err != nil {
				return errors.Wrapf(err, "cannot resolve long term auitor identity for [%s]", party.UniqueID())
			}
		}
		distributionListCompressed = append(distributionListCompressed, distributionListEntry{
			IsMe:     isMe,
			ID:       party,
			Auditor:  true,
			LongTerm: longTermIdentity,
		})

	}

	if logger.IsEnabledFor(zapcore.DebugLevel) {
		logger.Debugf("distributed tx to num parties [%d]", len(distributionListCompressed))
	}

	// Distribute the transaction to all parties in the distribution list.
	// Filter the metadata by Enrollment ID.
	// The auditor will receive the full set of metadata
	owner := NewOwner(context, c.tx.TokenService())
	for _, entry := range distributionListCompressed {
		if logger.IsEnabledFor(zapcore.DebugLevel) {
			logger.Debugf("distribute transaction envelope to [%s]", entry.ID.UniqueID())
		}

		// If it is me, no need to open a remote connection. Just store the envelope locally.
		if entry.IsMe && !entry.Auditor {
			if logger.IsEnabledFor(zapcore.DebugLevel) {
				logger.Debugf("This is me [%s], endorse locally", entry.ID.UniqueID())
			}

			// Store envelope
			if !c.Opts.SkipApproval {
				if err := StoreEnvelope(context, c.tx); err != nil {
					return errors.Wrapf(err, "failed storing envelope %s", c.tx.ID())
				}
			}

			// Store transaction in the token transaction database
			if err := StoreTransactionRecords(context, c.tx); err != nil {
				return errors.Wrapf(err, "failed adding transaction %s to the token transaction database", c.tx.ID())
			}

			continue
		} else {
			if logger.IsEnabledFor(zapcore.DebugLevel) {
				logger.Debugf("This is not me [%s:%s], ask endorse", entry.ID.UniqueID(), entry.EID)
			}
		}

		// The party is not me, open a connection to the party.
		// If the party is an auditor, then send the full set of metadata.
		// Otherwise, filter the metadata by Enrollment ID.
		var txRaw []byte
		var err error
		if entry.Auditor {
			if logger.IsEnabledFor(zapcore.DebugLevel) {
				logger.Debugf("This is an auditor [%s], send the full set of metadata", entry.ID.UniqueID())
			}
			txRaw, err = c.tx.Bytes()
			if err != nil {
				return errors.Wrap(err, "failed marshalling transaction content")
			}
		} else {
			if logger.IsEnabledFor(zapcore.DebugLevel) {
				logger.Debugf("This is not an auditor [%s], send the filtered metadata", entry.ID.UniqueID())
			}
			txRaw, err = c.tx.Bytes(entry.EID)
			if err != nil {
				return errors.Wrap(err, "failed marshalling transaction content")
			}
		}

		// Open a session to the party. and send the transaction.
		session, err := c.getSession(context, entry.ID)
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

		timeout := time.NewTimer(time.Minute * 4)

		var msg *view.Message
		select {
		case msg = <-ch:
			if logger.IsEnabledFor(zapcore.DebugLevel) {
				logger.Debugf("collect ack on distributed env: reply received from [%s]", entry.ID)
			}
			timeout.Stop()
		case <-timeout.C:
			timeout.Stop()
			return errors.Errorf("Timeout from party %s", entry.ID)
		}
		if msg.Status == view.ERROR {
			return errors.New(string(msg.Payload))
		}
		sigma := msg.Payload
		logger.Debugf("received ack from [%s] [%s], checking signature on [%s]", entry.LongTerm, hash.Hashable(sigma).String(),
			hash.Hashable(txRaw).String())

		verifier, err := view2.GetSigService(context).GetVerifier(entry.LongTerm)
		if err != nil {
			return errors.Wrapf(err, "failed getting verifier for [%s]", entry.ID)
		}
		if err := verifier.Verify(txRaw, sigma); err != nil {
			return errors.Wrapf(err, "failed verifying ack signature from [%s]", entry.ID)
		}

		if logger.IsEnabledFor(zapcore.DebugLevel) {
			logger.Debugf("CollectEndorsementsView: collected signature from %s", entry.ID)
		}

		if err := owner.appendTransactionEndorseAck(c.tx, entry.LongTerm, sigma); err != nil {
			return errors.Wrapf(err, "failed appending transaction endorsement ack to transaction %s", c.tx.ID())
		}
	}

	return nil
}

func (c *CollectEndorsementsView) requestBytes() ([]byte, error) {
	return c.tx.TokenRequest.MarshalToSign()
}

func (c *CollectEndorsementsView) getSession(context view.Context, p view.Identity) (view.Session, error) {
	s, ok := c.sessions[p.UniqueID()]
	if ok {
		logger.Debugf("getSession: found session for [%s]", p.UniqueID())
		return s, nil
	}
	return context.GetSession(context.Initiator(), p)
}

type ReceiveTransactionView struct {
	network string
}

func NewReceiveTransactionView(network string) *ReceiveTransactionView {
	return &ReceiveTransactionView{network: network}
}

func (f *ReceiveTransactionView) Call(context view.Context) (interface{}, error) {
	// Wait to receive a transaction back
	ch := context.Session().Receive()

	timeout := time.NewTimer(time.Minute * 4)
	defer timeout.Stop()

	select {
	case msg := <-ch:
		if msg.Status == view.ERROR {
			return nil, errors.New(string(msg.Payload))
		}
		if logger.IsEnabledFor(zapcore.DebugLevel) {
			logger.Debugf("ReceiveTransactionView: received transaction, len [%d][%s]", len(msg.Payload), hash.Hashable(msg.Payload))
		}
		tx, err := NewTransactionFromBytes(context, msg.Payload)
		if err != nil {
			// try to unmarshal pay
			tx, err = f.unmarshalAsSignatureRequest(context, msg.Payload)
			if err != nil {
				return nil, errors.Wrap(err, "failed to receive transaction")
			}
		}
		return tx, nil
	case <-timeout.C:
		return nil, errors.New("timeout reached")
	}
}

func (f *ReceiveTransactionView) unmarshalAsSignatureRequest(context view.Context, raw []byte) (*Transaction, error) {
	signatureRequest := &SignatureRequest{}
	err := Unmarshal(raw, signatureRequest)
	if err != nil {
		return nil, errors.Wrap(err, "failed unmarshalling signature request")
	}
	if len(signatureRequest.TX) == 0 {
		return nil, errors.Wrap(err, "no transaction received")
	}
	tx, err := NewTransactionFromBytes(context, signatureRequest.TX)
	if err != nil {
		return nil, errors.Wrap(err, "failed to receive transaction")
	}
	k, err := keys.CreateCompositeKey("signatureRequest", []string{tx.ID()})
	if err != nil {
		return nil, errors.Wrap(err, "failed to generate key to store signature request")
	}
	if err := kvs.GetService(context).Put(k, raw); err != nil {
		return nil, errors.Wrap(err, "failed to to store signature request")
	}
	return tx, nil
}

type EndorseView struct {
	tx *Transaction
}

// NewEndorseView returns an instance of the endorseView.
// The view does the following:
// 1. Wait for signature requests.
// 2. Upon receiving a signature request, it validates the request and send back the requested signature.
// 3. After, it waits to receive the Transaction. The Transaction is validated and stored locally
// to be processed at time of committing.
// 4. It sends back an ack.
func NewEndorseView(tx *Transaction) *EndorseView {
	return &EndorseView{tx: tx}
}

// Call executes the view.
// The view does the following:
// 1. Wait for signature requests.
// 2. Upon receiving a signature request, it validates the request and send back the requested signature.
// 3. After, it waits to receive the Transaction. The Transaction is validated and stored locally
// to be processed at time of committing.
// 4. It sends back an ack.
func (s *EndorseView) Call(context view.Context) (interface{}, error) {
	// Process signature requests
	requestsToBeSigned, err := requestsToBeSigned(s.tx.Request())
	if err != nil {
		return nil, errors.Wrapf(err, "failed collecting requests of signature")
	}

	session := context.Session()
	for range requestsToBeSigned {
		if logger.IsEnabledFor(zapcore.DebugLevel) {
			logger.Debugf("Receiving signature request...")
		}

		timeout := time.NewTimer(time.Minute)

		sessionChannel := session.Receive()
		var msg *view.Message
		select {
		case msg = <-sessionChannel:
			logger.Debug("message received from %s", session.Info().Caller)
			timeout.Stop()
		case <-timeout.C:
			timeout.Stop()
			return nil, errors.Errorf("Timeout from party %s", session.Info().Caller)
		}
		if msg.Status == view.ERROR {
			return nil, errors.New(string(msg.Payload))
		}

		// TODO: check what is signed...
		signatureRequest := &SignatureRequest{}
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

	// Store transient
	if err := s.tx.storeTransient(); err != nil {
		return nil, errors.Wrapf(err, "failed storing transient")
	}

	// Receive transaction with envelope
	_, rawRequest, err := s.receiveTransaction(context)
	if err != nil {
		return nil, errors.Wrapf(err, "failed receiving transaction")
	}

	// Store envelope
	if err := StoreEnvelope(context, s.tx); err != nil {
		return nil, errors.Wrapf(err, "failed storing envelope %s", s.tx.ID())
	}

	// Store transaction in the token transaction database
	if err := StoreTransactionRecords(context, s.tx); err != nil {
		return nil, errors.Wrapf(err, "failed storing transaction records %s", s.tx.ID())
	}

	// Send back an acknowledgement
	if logger.IsEnabledFor(zapcore.DebugLevel) {
		logger.Debugf("Send the ack")
		logger.Debugf("signing ack response: %s", hash.Hashable(rawRequest))
	}
	signer, err := view2.GetSigService(context).GetSigner(view2.GetIdentityProvider(context).DefaultIdentity())
	if err != nil {
		return nil, errors.WithMessagef(err, "failed to get signer for default identity")
	}
	sigma, err := signer.Sign(rawRequest)
	if err != nil {
		return nil, errors.WithMessage(err, "failed to sign ack response")
	}
	if logger.IsEnabledFor(zapcore.DebugLevel) {
		logger.Debugf("ack response: [%s] from [%s]", hash.Hashable(sigma), view2.GetIdentityProvider(context).DefaultIdentity())
	}
	if err := session.Send(sigma); err != nil {
		return nil, errors.WithMessage(err, "failed sending ack")
	}

	return s.tx, nil
}

func (s *EndorseView) receiveTransaction(context view.Context) (*Transaction, []byte, error) {
	if logger.IsEnabledFor(zapcore.DebugLevel) {
		logger.Debugf("Receive transaction with envelope...")
	}
	// TODO: this might also happen multiple times because of the pseudonym. Avoid this by identity resolution at the sender
	tx, err := ReceiveTransaction(context)
	if err != nil {
		return nil, nil, errors.Wrapf(err, "failed receiving transaction")
	}

	// TODO: compare with the existing transaction
	if logger.IsEnabledFor(zapcore.DebugLevel) {
		logger.Debugf("Processes Fabric Envelope with ID [%s]", tx.ID())
	}

	// Set the envelope
	s.tx = tx

	raw, err := tx.Bytes()
	if err != nil {
		return nil, nil, errors.Wrapf(err, "failed getting bytes for transaction %s", tx.ID())
	}
	return tx, raw, nil
}

func requestsToBeSigned(request *token.Request) ([]any, error) {
	var res []any
	transfers := request.Transfers()
	issues := request.Issues()
	sigService := request.TokenService.SigService()
	for _, issue := range issues {
		for _, sender := range issue.ExtraSigners {
			if _, err := sigService.GetSigner(sender); err == nil {
				res = append(res, issue)
			}
		}
	}
	for _, transfer := range transfers {
		for _, sender := range transfer.Senders {
			if _, err := sigService.GetSigner(sender); err == nil {
				res = append(res, transfer)
			}
		}
		for _, sender := range transfer.ExtraSigners {
			if _, err := sigService.GetSigner(sender); err == nil {
				res = append(res, transfer)
			}
		}
	}
	return res, nil
}

func mergeSigmas(maps ...map[string][]byte) map[string][]byte {
	merged := make(map[string][]byte)
	for _, m := range maps {
		for k, v := range m {
			merged[k] = v
		}
	}
	return merged
}

func IssueDistributionList(r *token.Request) []view.Identity {
	distributionList := make([]view.Identity, 0)
	for _, issue := range r.Issues() {
		distributionList = append(distributionList, issue.Issuer)
		distributionList = append(distributionList, issue.Receivers...)
	}
	return distributionList
}

func TransferDistributionList(r *token.Request) []view.Identity {
	distributionList := make([]view.Identity, 0)
	for _, transfer := range r.Transfers() {
		distributionList = append(distributionList, transfer.Senders...)
		distributionList = append(distributionList, transfer.Receivers...)
	}
	return distributionList
}
