/*
Copyright IBM Corp. All Rights Reserved.

SPDX-License-Identifier: Apache-2.0
*/

package htlc

import (
	"crypto"
	"crypto/rand"
	"encoding/base64"
	"encoding/json"
	"time"

	view2 "github.com/hyperledger-labs/fabric-smart-client/platform/view"
	"github.com/hyperledger-labs/fabric-smart-client/platform/view/view"
	"github.com/hyperledger-labs/fabric-token-sdk/token"
	"github.com/hyperledger-labs/fabric-token-sdk/token/core/identity"
	"github.com/hyperledger-labs/fabric-token-sdk/token/services/interop/encoding"
	"github.com/hyperledger-labs/fabric-token-sdk/token/services/ttx"
	token2 "github.com/hyperledger-labs/fabric-token-sdk/token/token"
	"github.com/pkg/errors"
)

const (
	ScriptType            = "htlc" // htlc script
	defaultDeadlineOffset = time.Hour
)

// WithHash sets a hash attribute to be used to customize the transfer command
func WithHash(hash []byte) token.TransferOption {
	return func(o *token.TransferOptions) error {
		if o.Attributes == nil {
			o.Attributes = map[interface{}]interface{}{}
		}
		o.Attributes["htlc.hash"] = hash
		return nil
	}
}

// WithHashFunc sets a hash function attribute to be used to customize the transfer command
func WithHashFunc(hashFunc crypto.Hash) token.TransferOption {
	return func(o *token.TransferOptions) error {
		if o.Attributes == nil {
			o.Attributes = map[interface{}]interface{}{}
		}
		o.Attributes["htlc.hashFunc"] = hashFunc
		return nil
	}
}

// WithHashEncoding sets a hash encoding attribute to be used to customize the transfer command
func WithHashEncoding(encoding encoding.Encoding) token.TransferOption {
	return func(o *token.TransferOptions) error {
		if o.Attributes == nil {
			o.Attributes = map[interface{}]interface{}{}
		}
		o.Attributes["htlc.hashEncoding"] = encoding
		return nil
	}
}

func compileTransferOptions(opts ...token.TransferOption) (*token.TransferOptions, error) {
	txOptions := &token.TransferOptions{}
	for _, opt := range opts {
		if err := opt(txOptions); err != nil {
			return nil, err
		}
	}
	return txOptions, nil
}

// Transaction holds a ttx transaction
type Transaction struct {
	*ttx.Transaction
}

// NewTransaction returns a new token transaction customized with the passed opts that will be signed by the passed signer
func NewTransaction(sp view.Context, signer view.Identity, opts ...ttx.TxOption) (*Transaction, error) {
	tx, err := ttx.NewTransaction(sp, signer, opts...)
	if err != nil {
		return nil, err
	}
	return &Transaction{
		Transaction: tx,
	}, nil
}

// NewAnonymousTransaction returns a new anonymous token transaction customized with the passed opts
func NewAnonymousTransaction(sp view.Context, opts ...ttx.TxOption) (*Transaction, error) {
	tx, err := ttx.NewAnonymousTransaction(sp, opts...)
	if err != nil {
		return nil, err
	}
	return &Transaction{
		Transaction: tx,
	}, nil
}

// NewTransactionFromBytes returns a new transaction from the passed bytes
func NewTransactionFromBytes(ctx view.Context, network, channel string, raw []byte) (*Transaction, error) {
	tx, err := ttx.NewTransactionFromBytes(ctx, raw)
	if err != nil {
		return nil, err
	}
	return &Transaction{
		Transaction: tx,
	}, nil
}

// Outputs returns a new OutputStream of the transaction's outputs
func (t *Transaction) Outputs() (*OutputStream, error) {
	outs, err := t.TokenRequest.Outputs()
	if err != nil {
		return nil, err
	}
	return NewOutputStream(outs), nil
}

// Lock appends a lock action to the token request of the transaction
func (t *Transaction) Lock(wallet *token.OwnerWallet, sender view.Identity, typ string, value uint64, recipient view.Identity, deadline time.Duration, opts ...token.TransferOption) ([]byte, error) {
	options, err := compileTransferOptions(opts...)
	if err != nil {
		return nil, err
	}
	if deadline == 0 {
		deadline = defaultDeadlineOffset
	}
	if recipient.IsNone() {
		return nil, errors.Errorf("must specify a recipient")
	}

	if sender == nil {
		sender, err = wallet.GetRecipientIdentity()
		if err != nil {
			return nil, errors.WithMessagef(err, "failed getting sender identity")
		}
	}

	var hash []byte
	hashFunc := crypto.SHA256 // default hash function
	var hashEncoding encoding.Encoding
	if options.Attributes != nil {
		boxed, ok := options.Attributes["htlc.hash"]
		if ok {
			hash, ok = boxed.([]byte)
			if !ok {
				return nil, errors.Errorf("expected htlc.hash attribute to be []byte, got [%T]", boxed)
			}
		}
		boxed, ok = options.Attributes["htlc.hashFunc"]
		if ok {
			hashFunc, ok = boxed.(crypto.Hash)
			if !ok {
				return nil, errors.Errorf("expected htlc.hashFunc attribute to be crypto.Hash, got [%T]", boxed)
			}
			if hashFunc == 0 {
				hashFunc = crypto.SHA256 // default hash function
			}
		}
		boxed, ok = options.Attributes["htlc.hashEncoding"]
		if ok {
			hashEncoding, ok = boxed.(encoding.Encoding)
			if !ok {
				return nil, errors.Errorf("expected htlc.hashEncoding attribute to be Encoding, got [%T]", boxed)
			}
		}
	}
	scriptID, preImage, script, err := t.recipientAsScript(sender, recipient, deadline, hash, hashFunc, hashEncoding)
	if err != nil {
		return nil, err
	}
	_, err = t.TokenRequest.Transfer(
		wallet,
		typ,
		[]uint64{value},
		[]view.Identity{scriptID},
		append(opts, token.WithTransferMetadata(LockKey(script.HashInfo.Hash), LockValue(script.HashInfo.Hash)))...,
	)
	if err != nil {
		return nil, err
	}

	return preImage, nil
}

// Reclaim appends a reclaim (transfer) action to the token request of the transaction
func (t *Transaction) Reclaim(wallet *token.OwnerWallet, tok *token2.UnspentToken) error {
	// TODO: handle this properly
	q, err := token2.ToQuantity(tok.Quantity, t.TokenRequest.TokenService.PublicParametersManager().Precision())
	if err != nil {
		return errors.Wrapf(err, "failed to convert quantity [%s]", tok.Quantity)
	}
	owner, err := identity.UnmarshallRawOwner(tok.Owner.Raw)
	if err != nil {
		return err
	}
	if owner.Type != ScriptType {
		return errors.Errorf("invalid owner type, expected htlc script")
	}
	script := &Script{}
	err = json.Unmarshal(owner.Identity, script)
	if err != nil {
		return errors.Errorf("failed to unmarshal RawOwner as an htlc script")
	}

	// Register the signer for the reclaim
	sigService := t.TokenService().SigService()
	signer, err := sigService.GetSigner(script.Sender)
	if err != nil {
		return err
	}
	verifier, err := sigService.OwnerVerifier(script.Sender)
	if err != nil {
		return err
	}
	logger.Debugf("registering signer for reclaim...")
	if err := sigService.RegisterSigner(
		tok.Owner.Raw,
		signer,
		verifier,
	); err != nil {
		return err
	}

	if err := view2.GetEndpointService(t.SP).Bind(script.Sender, tok.Owner.Raw); err != nil {
		return err
	}

	return t.Transfer(wallet, tok.Type, []uint64{q.ToBigInt().Uint64()}, []view.Identity{script.Sender}, token.WithTokenIDs(tok.Id))
}

// Claim appends a claim (transfer) action to the token request of the transaction
func (t *Transaction) Claim(wallet *token.OwnerWallet, tok *token2.UnspentToken, preImage []byte) error {
	// TODO: handle this properly
	q, err := token2.ToQuantity(tok.Quantity, t.TokenRequest.TokenService.PublicParametersManager().Precision())
	if err != nil {
		return errors.Wrapf(err, "failed to convert quantity [%s]", tok.Quantity)
	}

	owner, err := identity.UnmarshallRawOwner(tok.Owner.Raw)
	if err != nil {
		return err
	}
	script := &Script{}
	if owner.Type != ScriptType {
		return errors.New("invalid owner type, expected htlc script")
	}
	if err := json.Unmarshal(owner.Identity, script); err != nil {
		return errors.New("failed to unmarshal RawOwner as an htlc script")
	}

	if len(preImage) == 0 {
		return errors.New("preImage is nil")
	}

	// TODO: does the pre-image match?

	// Register the signer for the claim
	logger.Debugf("registering signer for claim...")
	sigService := t.TokenService().SigService()
	recipientSigner, err := sigService.GetSigner(script.Recipient)
	if err != nil {
		return err
	}
	recipientVerifier, err := sigService.OwnerVerifier(script.Recipient)
	if err != nil {
		return err
	}
	if err := sigService.RegisterSigner(
		tok.Owner.Raw,
		&ClaimSigner{
			Recipient: recipientSigner,
			Preimage:  preImage,
		},
		&ClaimVerifier{
			Recipient: recipientVerifier,
			HashInfo: HashInfo{
				Hash:         script.HashInfo.Hash,
				HashFunc:     script.HashInfo.HashFunc,
				HashEncoding: script.HashInfo.HashEncoding,
			},
		},
	); err != nil {
		return err
	}

	if err := view2.GetEndpointService(t.SP).Bind(script.Recipient, tok.Owner.Raw); err != nil {
		return err
	}

	image, err := script.HashInfo.Image(preImage)
	if err != nil {
		return errors.WithMessagef(err, "failed to compute image of [%x]", preImage)
	}

	return t.Transfer(
		wallet,
		tok.Type,
		[]uint64{q.ToBigInt().Uint64()},
		[]view.Identity{script.Recipient},
		token.WithTokenIDs(tok.Id),
		token.WithTransferMetadata(ClaimKey(image), preImage),
	)
}

func (t *Transaction) recipientAsScript(sender, recipient view.Identity, deadline time.Duration, h []byte, hashFunc crypto.Hash, hashEncoding encoding.Encoding) (view.Identity, []byte, *Script, error) {
	// sample pre-image and its hash
	var preImage []byte
	var err error
	if len(h) == 0 {
		preImage, err = CreateNonce()
		if err != nil {
			return nil, nil, nil, err
		}
		hash := hashFunc.New()
		if _, err := hash.Write(preImage); err != nil {
			return nil, nil, nil, err
		}
		h = hash.Sum(nil)
		// no need to encode if encoding is none (=0)
		if hashEncoding != 0 {
			he := hashEncoding.New()
			if he == nil {
				return nil, nil, nil, errors.New("hashEncoding.New() returned nil")
			}
			h = []byte(he.EncodeToString(h))
		}
	}

	logger.Debugf("pair (pre-image, hash) = (%s,%s)", base64.StdEncoding.EncodeToString(preImage), base64.StdEncoding.EncodeToString(h))

	script := &Script{
		HashInfo: HashInfo{
			Hash:         h,
			HashFunc:     hashFunc,
			HashEncoding: hashEncoding,
		},
		Deadline:  time.Now().Add(deadline),
		Recipient: recipient,
		Sender:    sender,
	}
	rawScript, err := json.Marshal(script)
	if err != nil {
		return nil, nil, nil, err
	}
	ro := &identity.RawOwner{
		Type:     ScriptType,
		Identity: rawScript,
	}
	raw, err := identity.MarshallRawOwner(ro)
	if err != nil {
		return nil, nil, nil, err
	}
	return raw, preImage, script, nil
}

// CreateNonce generates a nonce using the common/crypto package
func CreateNonce() ([]byte, error) {
	nonce, err := getRandomNonce()
	return nonce, errors.WithMessage(err, "error generating random nonce")
}

func getRandomNonce() ([]byte, error) {
	key := make([]byte, 24)
	_, err := rand.Read(key)
	if err != nil {
		return nil, errors.Wrap(err, "error getting random bytes")
	}
	return key, nil
}
