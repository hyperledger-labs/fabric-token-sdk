/*
Copyright IBM Corp. All Rights Reserved.

SPDX-License-Identifier: Apache-2.0
*/

package ttx

import (
	"encoding/base64"
	errors2 "errors"
	"fmt"
	"reflect"
	"runtime/debug"
	"time"

	"github.com/hyperledger-labs/fabric-smart-client/platform/common/utils/collections"
	view2 "github.com/hyperledger-labs/fabric-smart-client/platform/view"
	"github.com/hyperledger-labs/fabric-smart-client/platform/view/services/hash"
	"github.com/hyperledger-labs/fabric-smart-client/platform/view/services/kvs"
	"github.com/hyperledger-labs/fabric-smart-client/platform/view/view"
	"github.com/hyperledger-labs/fabric-token-sdk/token"
	"github.com/hyperledger-labs/fabric-token-sdk/token/services/identity/multisig"
	"github.com/hyperledger-labs/fabric-token-sdk/token/services/network"
	"github.com/hyperledger-labs/fabric-token-sdk/token/services/tokens"
	"github.com/pkg/errors"
	"go.opentelemetry.io/otel/trace"
	"go.uber.org/zap/zapcore"
)

type distributionListEntry struct {
	IsMe     bool
	LongTerm view.Identity
	ID       view.Identity
	EID      string
	Auditor  bool
}

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
	return sr.Request
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
	span := trace.SpanFromContext(context.Context())
	metrics := GetMetrics(context)

	externalWallets := make(map[string]ExternalWalletSigner)
	// 1. First collect signatures on the token request
	span.AddEvent("Request signatures on issues")
	issueSigmas, err := c.requestSignaturesOnIssues(context, externalWallets)
	if err != nil {
		return nil, errors.WithMessage(err, "failed requesting signatures on issues")
	}

	span.AddEvent("Request signatures on transfers")
	transferSigmas, err := c.requestSignaturesOnTransfers(context, externalWallets)
	if err != nil {
		return nil, errors.WithMessage(err, "failed requesting signatures on transfers")
	}

	// signal the external wallets that the process is completed
	span.AddEvent("Inform external wallets that endorsement is complete")
	for id, signer := range externalWallets {
		if err := signer.Done(); err != nil {
			span.RecordError(err)
			logger.Errorf("failed to signal done external wallet [%s]", id)
		}
	}

	// Add the signatures to the token request
	span.AddEvent("Add the signatures to the token request")
	if !c.tx.TokenRequest.SetSignatures(mergeSigmas(issueSigmas, transferSigmas)) {
		return nil, errors.New("failed setting signatures on token request, some signatures are missing")
	}

	// 2. Audit
	var auditors []view.Identity
	if !c.Opts.SkipAuditing {
		span.AddEvent("Request audit")
		auditors, err = c.requestAudit(context)
		if err != nil {
			return nil, errors.WithMessage(err, "failed requesting auditing")
		}
	}
	// 3. Endorse and return the transaction envelope
	var env *network.Envelope
	if !c.Opts.SkipApproval {
		span.AddEvent("Request approval from endorser")
		env, err = c.requestApproval(context)
		if err != nil {
			return nil, errors.WithMessage(err, "failed requesting approval")
		}
	}
	// Distribute Env to all parties
	distributionList := append(IssueDistributionList(c.tx.TokenRequest), TransferDistributionList(c.tx.TokenRequest)...)
	span.AddEvent(fmt.Sprintf("Distribute envelope to %d involved parties", len(distributionList)))
	if err := c.distributeEnvToParties(context, env, distributionList, auditors); err != nil {
		span.RecordError(err)
		return nil, errors.WithMessage(err, "failed distributing envelope")
	}

	// Cleanup audit
	span.AddEvent("Cleanup audit")
	if err := c.cleanupAudit(context); err != nil {
		span.RecordError(err)
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
	return c.requestSignatures(
		c.tx.TokenRequest.IssueSigners(),
		c.tx.TokenService().SigService().IssuerVerifier,
		context,
		externalWallets,
	)
}

func (c *CollectEndorsementsView) requestSignaturesOnTransfers(context view.Context, externalWallets map[string]ExternalWalletSigner) (map[string][]byte, error) {
	if logger.IsEnabledFor(zapcore.DebugLevel) {
		logger.Debugf("collecting signature on [%d] request transfer", len(c.tx.TokenRequest.Metadata.Transfers))
	}

	return c.requestSignatures(
		c.tx.TokenRequest.TransferSigners(),
		c.tx.TokenService().SigService().OwnerVerifier,
		context,
		externalWallets,
	)
}

func (c *CollectEndorsementsView) requestSignatures(signers []view.Identity, verifierGetter verifierGetterFunc, context view.Context, externalWallets map[string]ExternalWalletSigner) (map[string][]byte, error) {
	span := trace.SpanFromContext(context.Context())
	span.AddEvent(fmt.Sprintf("Request %d signatures", len(signers)))
	requestRaw, err := c.requestBytes()
	if err != nil {
		return nil, err
	}
	txRaw, err := c.tx.Bytes()
	if err != nil {
		return nil, err
	}
	txIDRaw := []byte(c.tx.ID())

	sigmas := make(map[string][]byte)
	for i, signerIdentity := range signers {
		span.AddEvent(fmt.Sprintf("Request signature no %d", i))
		// we have the following possibilities:
		// - there is a signer locally bound to the party, use it to generate the signature
		// - there is a wallet bound to the party but the signer is not local, the signature is generated externally
		// - the identity is a multi-sig identity
		// - the signature must be generated by a remote party

		signatureRequest := &SignatureRequest{
			TX:      txRaw,
			Request: requestRaw,
			TxID:    txIDRaw,
			Signer:  signerIdentity,
		}
		logger.Debugf("collecting signature on request from [%s]", signerIdentity)

		// Case: the identity is a multi-sig identity
		ok, multiSigners, err := multisig.Unwrap(signerIdentity)
		if err != nil {
			return nil, errors.Wrapf(err, "failed unwrapping multi-sig identity [%s]", signerIdentity)
		}
		if ok {
			span.AddEvent(fmt.Sprintf("%d. Multi-sig signer", i))
			logger.Debugf("found multi-sig identity [%s], request multi-sig signature to [%d] parties", signerIdentity, len(multiSigners))
			// collect the signatures from multiSigners
			multiSignersSigmas, err := c.requestSignatures(multiSigners, verifierGetter, context, externalWallets)
			if err != nil {
				return nil, errors.WithMessage(err, "failed requesting signatures")
			}
			logger.Debugf("collected [%d] signatures for multi-sig identity [%s]", len(multiSignersSigmas), signerIdentity)
			sigma, err := multisig.JoinSignatures(multiSigners, multiSignersSigmas)
			if err != nil {
				return nil, errors.WithMessage(err, "failed joining multi-sig signatures")
			}
			sigmas[signerIdentity.UniqueID()] = sigma
			span.AddEvent("Done requesting multi-sig")
			continue
		}

		// Case: there is a signer locally bound to the party, use it to generate the signature
		if signer, err := c.tx.TokenService().SigService().GetSigner(signerIdentity); err == nil {
			span.AddEvent(fmt.Sprintf("%d. Local signer", i))
			if logger.IsEnabledFor(zapcore.DebugLevel) {
				logger.Debugf("found signer for party [%s], request local signature", signerIdentity)
			}
			sigma, err := c.signLocal(signerIdentity, signer, signatureRequest)
			if err != nil {
				return nil, errors.WithMessagef(err, "failed signing local for party [%s]", signerIdentity)
			}
			sigmas[signerIdentity.UniqueID()] = sigma
			span.AddEvent("Done local signing")
			continue
		} else {
			if logger.IsEnabledFor(zapcore.DebugLevel) {
				logger.Debugf("failed to find a signer for party [%s]: [%s]", signerIdentity, err)
			}
		}

		// Case: there is a wallet bound to the party but the signer is not local, the signature is generated externally
		if w := c.tx.TokenService().WalletManager().OwnerWallet(signerIdentity); w != nil {
			span.AddEvent(fmt.Sprintf("%d. External signer", i))
			if logger.IsEnabledFor(zapcore.DebugLevel) {
				logger.Debugf("found wallet for party [%s], request external signature", signerIdentity)
			}
			ews := c.Opts.ExternalWalletSigner(w.ID())
			if ews == nil {
				return nil, errors.Errorf("no external wallet signer found for [%s][%s]", w.ID(), signerIdentity)
			}
			externalWallets[w.ID()] = ews
			sigma, err := c.signExternal(signerIdentity, ews, signatureRequest)
			if err != nil {
				return nil, errors.WithMessagef(err, "failed signing external for party [%s]", signerIdentity)
			}
			sigmas[signerIdentity.UniqueID()] = sigma
			span.AddEvent("Done external signing")
			continue
		}

		// Case: the signature must be generated by a remote party
		if logger.IsEnabledFor(zapcore.DebugLevel) {
			logger.Debugf("no signer or wallet found for party [%s], request remote signature", signerIdentity)
		}
		span.AddEvent(fmt.Sprintf("%d. Remote signing", i))
		sigma, err := c.signRemote(context, signerIdentity, signatureRequest, verifierGetter)
		if err != nil {
			return nil, errors.WithMessagef(err, "failed signing remote for party [%s]", signerIdentity)
		}
		sigmas[signerIdentity.UniqueID()] = sigma
		span.AddEvent("Done remote signing")
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
		logger.Debugf("signature generated (local, me) [%s,%s,%s,%v]",
			hash.Hashable(signatureRequest.MessageToSign()).String(),
			hash.Hashable(sigma).String(),
			party.UniqueID(),
			getIdentifier(signer),
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
		logger.Debugf("signature generated (external, me) [%s,%s,%s]",
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
	signatureRequestRaw, err := Marshal(signatureRequest)
	if err != nil {
		return nil, err
	}
	err = session.SendWithContext(context.Context(), signatureRequestRaw)
	if err != nil {
		return nil, errors.Wrap(err, "failed sending transaction content")
	}

	sigma, err := ReadMessage(session, time.Minute)
	if err != nil {
		return nil, errors.Wrap(err, "failed reading message")
	}

	verifier, err := verifierGetter(party)
	if err != nil {
		return nil, errors.Wrapf(err, "failed getting verifier for [%s]", party)
	}
	if logger.IsEnabledFor(zapcore.DebugLevel) {
		logger.Debugf("verify signature [%s][%s][%s] for txid [%s]",
			hash.Hashable(signatureRequest.MessageToSign()).String(),
			hash.Hashable(sigma).String(),
			party,
			c.tx.ID(),
		)
	}

	err = verifier.Verify(signatureRequest.MessageToSign(), sigma)
	if err != nil {
		return nil, errors.Wrapf(err, "failed verifying signature [%s] from [%s]", sigma, party)
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
	c.tx.Envelope = env
	return env, nil
}

func (c *CollectEndorsementsView) requestAudit(context view.Context) ([]view.Identity, error) {
	auditors := c.tx.TokenService().PublicParametersManager().PublicParameters().Auditors()
	logger.Debugf("# auditors in public parameters [%d]", len(auditors))
	if len(c.tx.TokenService().PublicParametersManager().PublicParameters().Auditors()) == 0 {
		return nil, nil
	}

	if !c.tx.Opts.Auditor.IsNone() {
		if logger.IsEnabledFor(zapcore.DebugLevel) {
			logger.Debugf("ask auditing to [%s]", c.tx.Opts.Auditor)
		}
		local := view2.GetSigService(context).IsMe(c.tx.Opts.Auditor)
		sessionBoxed, err := context.RunView(newAuditingViewInitiator(c.tx, local))
		if err != nil {
			return nil, errors.WithMessagef(err, "failed requesting auditing from [%s]", c.tx.Opts.Auditor.String())
		}
		c.sessions[c.tx.Opts.Auditor.String()] = sessionBoxed.(view.Session)
		return []view.Identity{c.tx.Opts.Auditor}, nil
	} else {
		logger.Warnf("no auditor specified, skip auditing, but # auditors in public parameters is [%d]", len(auditors))
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

func (c *CollectEndorsementsView) distributeEnvToParties(context view.Context, env *network.Envelope, distributionList []view.Identity, auditors []view.Identity) error {
	span := trace.SpanFromContext(context.Context())
	span.AddEvent("Start distribute to parties")
	if c.Opts.SkipDistributeEnv {
		span.AddEvent("Skip distribute envelopes")
		return nil
	}
	span.AddEvent("Start distributing envelopes")

	if !c.Opts.SkipApproval {
		// perform sanity checks
		if env == nil {
			return errors.New("transaction envelope is empty")
		}
		if logger.IsEnabledFor(zapcore.DebugLevel) {
			logger.Debugf("distribute env [%s]", env.String())
		}
	}

	// TODO: double check that the transaction is valid
	// if err := c.tx.IsValid(); err != nil {
	// 	return errors.Wrap(err, "failed verifying transaction content before distributing it")
	// }

	// Distribute the transaction to all parties in the distribution list.
	// Filter the metadata by Enrollment ID.
	// The auditor will receive the full set of metadata
	span.AddEvent("Prepare final distribution list")
	finalDistributionList, err := c.prepareDistributionList(context, auditors, distributionList)
	if err != nil {
		return errors.Wrap(err, "failed preparing distribution list")
	}

	owner := NewOwner(context, c.tx.TokenService())

	// Store transaction in the token transaction database
	span.AddEvent("Store transaction records")
	if err := StoreTransactionRecords(context, c.tx); err != nil {
		return errors.Wrapf(err, "failed adding transaction %s to the token transaction database", c.tx.ID())
	}

	span.AddEvent(fmt.Sprintf("Start distributing to %d parties", len(finalDistributionList)))
	for i, entry := range finalDistributionList {
		span.AddEvent(fmt.Sprintf("%d. Distribute to %s", i, entry.EID))
		if logger.IsEnabledFor(zapcore.DebugLevel) {
			logger.Debugf("distribute transaction envelope to [%s]", entry.ID.UniqueID())
		}

		// If it is me, no need to open a remote connection. Just store the envelope locally.
		if entry.IsMe && !entry.Auditor {
			if logger.IsEnabledFor(zapcore.DebugLevel) {
				logger.Debugf("This is me [%s], endorse locally", entry.ID.UniqueID())
			}
			span.AddEvent("Skip myself")
			continue
		} else {
			if logger.IsEnabledFor(zapcore.DebugLevel) {
				logger.Debugf("This is not me [%s:%s], ask endorse", entry.ID.UniqueID(), entry.EID)
			}
		}

		// The party is not mex, open a connection to the party.
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

		// TODO:
		// This operation might be retried, but this requires a change of protocol to make sure the recipient can always receive.
		// It could be done by using a new context.
		span.AddEvent(fmt.Sprintf("Distribute to %s", entry.EID))
		if err := c.distributeEvnToParty(context, &entry, txRaw, owner); err != nil {
			return errors.Wrapf(err, "failed distribute evn of tx [%s] to party [%s:%s]", c.tx.ID(), entry.EID, entry.ID)
		}
		span.AddEvent(fmt.Sprintf("Done distributing to %s", entry.EID))
	}

	return nil
}

func (c *CollectEndorsementsView) distributeEvnToParty(context view.Context, entry *distributionListEntry, txRaw []byte, owner *TxOwner) error {
	span := trace.SpanFromContext(context.Context())
	// Open a session to the party. and send the transaction.
	session, err := c.getSession(context, entry.ID)
	if err != nil {
		return errors.Wrap(err, "failed getting session")
	}
	// Send the content
	span.AddEvent("Send transaction content")
	err = session.SendWithContext(context.Context(), txRaw)
	if err != nil {
		return errors.Wrap(err, "failed sending transaction content")
	}

	span.AddEvent("Wait for ack")
	sigma, err := ReadMessage(session, 1*time.Minute)
	if err != nil {
		return errors.Wrapf(err, "failed reading message on session [%s]", session.Info().ID)
	}
	logger.Debugf("received ack from [%s] [%s], checking signature on [%s]",
		entry.LongTerm, hash.Hashable(sigma).String(),
		hash.Hashable(txRaw).String())

	span.AddEvent("Verify signature")
	verifier, err := view2.GetSigService(context).GetVerifier(entry.LongTerm)
	if err != nil {
		return errors.Wrapf(err, "failed getting verifier for identity [%s]", entry.ID)
	}
	if err := verifier.Verify(txRaw, sigma); err != nil {
		return errors.Wrapf(err, "failed verifying ack signature from [%s]", entry.ID)
	}

	if logger.IsEnabledFor(zapcore.DebugLevel) {
		logger.Debugf("CollectEndorsementsView: collected signature from %s", entry.ID)
	}

	span.AddEvent("Append ack to Endorsement table")
	if err := owner.appendTransactionEndorseAck(c.tx, entry.LongTerm, sigma); err != nil {
		return errors.Wrapf(err, "failed appending transaction endorsement ack to transaction %s", c.tx.ID())
	}

	return nil
}

func (c *CollectEndorsementsView) prepareDistributionList(context view.Context, auditors []view.Identity, distributionList []view.Identity) ([]distributionListEntry, error) {
	// Compress distributionList by removing duplicates

	// check if there are multisig identities, if yes, unwrap them
	allIds := make([]view.Identity, 0, len(distributionList)+len(auditors))
	for _, id := range distributionList {
		if id.IsNone() {
			// This is a redeem, nothing to do here.
			continue
		}
		ok, multiSigners, err := multisig.Unwrap(id)
		if err != nil {
			return nil, errors.Wrapf(err, "failed unwrapping multi-sig identity [%s]", id)
		}
		if ok {
			allIds = append(allIds, multiSigners...)
		} else {
			allIds = append(allIds, id)
		}
	}
	distributionList = allIds
	allIds = append(allIds, auditors...)

	mine := collections.NewSet(view2.GetSigService(context).AreMe(allIds...)...)
	remainingIds := make([]view.Identity, 0, len(allIds)-mine.Length())
	for _, id := range allIds {
		if !mine.Contains(id.UniqueID()) {
			remainingIds = append(remainingIds, id)
		}
	}
	mine.Add(c.tx.TokenService().SigService().AreMe(remainingIds...)...)
	logger.Debugf("%d/%d ids were mine", mine.Length(), len(allIds))

	var distributionListCompressed []distributionListEntry
	for _, party := range distributionList {
		// For each party in the distribution list:
		// - check if it is me
		// - check if it is an auditor
		// - extract the corresponding long term identity
		// If the long term identity has not been added yet, add it to the list.
		// If the party is me or an auditor, no need to extract the enrollment ID.
		if logger.IsEnabledFor(zapcore.DebugLevel) {
			logger.Debugf("distribute env to [%s]?", party.UniqueID())
		}

		isMe := mine.Contains(party.UniqueID())
		if !isMe {
			// check if there is a wallet that contains that identity
			isMe = c.tx.TokenService().WalletManager().OwnerWallet(party) != nil
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
				return nil, errors.Wrapf(err, "cannot resolve long term identity for [%s]", party.UniqueID())
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
					return nil, errors.Wrapf(err, "failed getting enrollment ID for [%s]", party.UniqueID())
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
		isMe := mine.Contains(party.UniqueID())
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
				return nil, errors.Wrapf(err, "cannot resolve long term auitor identity for [%s]", party.UniqueID())
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
	return distributionListCompressed, nil
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

type ReceiveTransactionView struct{}

func NewReceiveTransactionView() *ReceiveTransactionView {
	return &ReceiveTransactionView{}
}

func (f *ReceiveTransactionView) Call(context view.Context) (interface{}, error) {
	span := trace.SpanFromContext(context.Context())
	span.AddEvent("start_receive_transaction_view")
	defer span.AddEvent("end_receive_transaction_view")

	msg, err := ReadMessage(context.Session(), time.Minute*4)
	if err != nil {
		span.RecordError(err)
	}
	span.AddEvent("receive_tx")

	if logger.IsEnabledFor(zapcore.DebugLevel) {
		logger.Debugf("ReceiveTransactionView: received transaction, len [%d][%s]", len(msg), hash.Hashable(msg))
	}
	if len(msg) == 0 {
		info := context.Session().Info()
		logger.Errorf("received empty message, session closed [%s:%v]: [%s]", info.ID, info.Closed, string(debug.Stack()))
		return nil, errors.Errorf("received empty message, session closed [%s:%v]", info.ID, info.Closed)
	}
	tx, err := NewTransactionFromBytes(context, msg)
	if err != nil {
		logger.Warnf("failed creating transaction from bytes: [%v], try to unmarshal as signature request...", err)
		// try to unmarshal as SignatureRequest
		var err2 error
		tx, err2 = f.unmarshalAsSignatureRequest(context, msg)
		if err2 != nil {
			return nil, errors.Wrap(errors2.Join(err, err2), "failed to receive transaction")
		}
	}
	return tx, nil
}

func (f *ReceiveTransactionView) unmarshalAsSignatureRequest(context view.Context, raw []byte) (*Transaction, error) {
	signatureRequest := &SignatureRequest{}
	err := Unmarshal(raw, signatureRequest)
	if err != nil {
		return nil, errors.Wrapf(err, "failed unmarshalling signature request, got [%s]", string(raw))
	}
	if len(signatureRequest.TX) == 0 {
		return nil, errors.Wrap(err, "no transaction received")
	}
	tx, err := NewTransactionFromBytes(context, signatureRequest.TX)
	if err != nil {
		return nil, errors.Wrap(err, "failed to receive transaction")
	}
	k, err := kvs.CreateCompositeKey("signatureRequest", []string{tx.ID()})
	if err != nil {
		return nil, errors.Wrap(err, "failed to generate key to store signature request")
	}
	k = base64.StdEncoding.EncodeToString([]byte(k))
	if kvss, err := context.GetService(&kvs.KVS{}); err != nil {
		return nil, errors.Wrap(err, "failed to get KVS from context")
	} else if err := kvss.(*kvs.KVS).Put(k, base64.StdEncoding.EncodeToString(raw)); err != nil {
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
	logger.Debugf("check expected number of requests to sign for txid [%s]", s.tx.ID())
	requestsToBeSigned, err := requestsToBeSigned(s.tx.Request())
	if err != nil {
		return nil, errors.Wrapf(err, "failed collecting requests of signature")
	}

	logger.Debugf("expect [%d] requests to sign for txid [%s]", len(requestsToBeSigned), s.tx.ID())

	session := context.Session()
	k, err := kvs.CreateCompositeKey("signatureRequest", []string{s.tx.ID()})
	if err != nil {
		return nil, errors.Wrap(err, "failed to generate key to store signature request")
	}
	k = base64.StdEncoding.EncodeToString([]byte(k))
	kvss, err := context.GetService(&kvs.KVS{})
	if err != nil {
		return nil, errors.Wrap(err, "failed to get KVS from context")
	}
	storage := kvss.(*kvs.KVS)
	for i := range requestsToBeSigned {
		var srRaw []byte
		signatureRequest := &SignatureRequest{}

		if i == 0 && storage.Exists(k) {
			if err := kvss.(*kvs.KVS).Get(k, &srRaw); err != nil {
				return nil, errors.Wrap(err, "failed to to store signature request")
			}
		} else {
			if logger.IsEnabledFor(zapcore.DebugLevel) {
				logger.Debugf("Receiving signature request...")
			}
			srRaw, err = ReadMessage(session, time.Minute)
			if err != nil {
				return nil, errors.Wrap(err, "failed reading signature request")
			}
		}
		// TODO: check what is signed...
		err = Unmarshal(srRaw, signatureRequest)
		if err != nil {
			return nil, errors.Wrap(err, "failed unmarshalling signature request")
		}

		sigService := s.tx.TokenService().SigService()
		if !sigService.IsMe(signatureRequest.Signer) {
			return nil, errors.Errorf("identity [%s] is not me", signatureRequest.Signer.UniqueID())
		}
		signer, err := sigService.GetSigner(signatureRequest.Signer)
		if err != nil {
			return nil, errors.Wrapf(err, "cannot find signer for [%s]", signatureRequest.Signer.UniqueID())
		}
		sigma, err := signer.Sign(signatureRequest.MessageToSign())
		if err != nil {
			return nil, errors.Wrapf(err, "failed signing request")
		}
		if logger.IsEnabledFor(zapcore.DebugLevel) {
			logger.Debugf("Send back signature [%s][%s]", signatureRequest.Signer, hash.Hashable(sigma))
		}
		err = session.SendWithContext(context.Context(), sigma)
		if err != nil {
			return nil, errors.Wrapf(err, "failed sending signature back")
		}
	}

	// Receive transaction with envelope
	receivedTx, err := s.receiveTransaction(context)
	if err != nil {
		return nil, errors.Wrapf(err, "failed receiving transaction")
	}

	// Store transaction in the token transaction database
	if err := StoreTransactionRecords(context, s.tx); err != nil {
		return nil, errors.Wrapf(err, "failed storing transaction records %s", s.tx.ID())
	}

	// Send back an acknowledgement
	if logger.IsEnabledFor(zapcore.DebugLevel) {
		logger.Debugf("signing ack response [%s] with identity [%s]", hash.Hashable(receivedTx.FromRaw), view2.GetIdentityProvider(context).DefaultIdentity())
	}
	signer, err := view2.GetSigService(context).GetSigner(view2.GetIdentityProvider(context).DefaultIdentity())
	if err != nil {
		return nil, errors.WithMessagef(err, "failed to get signer for default identity")
	}
	sigma, err := signer.Sign(receivedTx.FromRaw)
	if err != nil {
		return nil, errors.WithMessage(err, "failed to sign ack response")
	}
	if logger.IsEnabledFor(zapcore.DebugLevel) {
		logger.Debugf("ack response: [%s] from [%s]", hash.Hashable(sigma), view2.GetIdentityProvider(context).DefaultIdentity())
	}
	if err := session.SendWithContext(context.Context(), sigma); err != nil {
		return nil, errors.WithMessage(err, "failed sending ack")
	}

	// cache the token request into the tokens db
	t, err := tokens.GetService(context, s.tx.TMSID())
	if err != nil {
		return nil, errors.Wrapf(err, "failed to get tokens db for [%s]", s.tx.TMSID())
	}
	if err := t.CacheRequest(s.tx.TMSID(), s.tx.TokenRequest); err != nil {
		logger.Warnf("failed to cache token request [%s], this might cause delay, investigate when possible: [%s]", s.tx.TokenRequest.Anchor, err)
	}

	return s.tx, nil
}

func (s *EndorseView) receiveTransaction(context view.Context) (*Transaction, error) {
	if logger.IsEnabledFor(zapcore.DebugLevel) {
		logger.Debugf("Receive transaction with envelope...")
	}
	// TODO: this might also happen multiple times because of the pseudonym. Avoid this by identity resolution at the sender
	tx, err := ReceiveTransaction(context)
	if err != nil {
		return nil, errors.Wrapf(err, "failed receiving transaction")
	}

	// TODO: compare with the existing transaction
	if logger.IsEnabledFor(zapcore.DebugLevel) {
		logger.Debugf("Processes Fabric Envelope with ID [%s]", tx.ID())
	}

	// Set the envelope
	request := s.tx.TokenRequest
	s.tx = tx
	s.tx.TokenRequest = request
	return tx, nil
}

func requestsToBeSigned(request *token.Request) ([]any, error) {
	var res []any
	transfers := request.Transfers()
	issues := request.Issues()
	sigService := request.TokenService.SigService()
	for _, issue := range issues {
		for _, sender := range issue.ExtraSigners {
			if sigService.IsMe(sender) {
				res = append(res, issue)
			}
		}
	}
	for _, transfer := range transfers {
		for _, sender := range transfer.Senders {
			if sigService.IsMe(sender) {
				res = append(res, transfer)
			}
		}
		for _, sender := range transfer.ExtraSigners {
			if sigService.IsMe(sender) {
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

func getIdentifier(f any) string {
	if f == nil {
		return "<nil view>"
	}
	t := reflect.TypeOf(f)
	for t.Kind() == reflect.Ptr {
		t = t.Elem()
	}
	return t.PkgPath() + "/" + t.Name()
}
