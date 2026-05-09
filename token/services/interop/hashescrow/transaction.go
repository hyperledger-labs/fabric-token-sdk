/*
Copyright IBM Corp. All Rights Reserved.

SPDX-License-Identifier: Apache-2.0
*/

package hashescrow

import (
	"context"
	"crypto"
	"crypto/rand"
	"encoding/base64"
	"encoding/json"

	"github.com/hyperledger-labs/fabric-smart-client/pkg/utils/errors"
	"github.com/hyperledger-labs/fabric-smart-client/platform/view/services/endpoint"
	"github.com/hyperledger-labs/fabric-smart-client/platform/view/view"
	"github.com/hyperledger-labs/fabric-token-sdk/token"
	"github.com/hyperledger-labs/fabric-token-sdk/token/driver"
	"github.com/hyperledger-labs/fabric-token-sdk/token/services/identity"
	"github.com/hyperledger-labs/fabric-token-sdk/token/services/interop/encoding"
	"github.com/hyperledger-labs/fabric-token-sdk/token/services/ttx"
	token2 "github.com/hyperledger-labs/fabric-token-sdk/token/token"
)

const (
	ScriptType       = driver.HashEscrowIdentityType
	ScriptTypeString = driver.HashEscrowIdentityTypeString
)

// WithHash sets a hash attribute to customize the lock command.
// This is kept for backward compatibility and maps to recipient hash.
func WithHash(hash []byte) token.TransferOption {
	return WithRecipientHash(hash)
}

// WithRecipientHash sets recipient-claim hash attribute.
func WithRecipientHash(hash []byte) token.TransferOption {
	return func(o *token.TransferOptions) error {
		if o.Attributes == nil {
			o.Attributes = map[interface{}]interface{}{}
		}
		o.Attributes["hashescrow.recipientHash"] = hash

		return nil
	}
}

// WithSenderHash sets sender-return hash attribute.
func WithSenderHash(hash []byte) token.TransferOption {
	return func(o *token.TransferOptions) error {
		if o.Attributes == nil {
			o.Attributes = map[interface{}]interface{}{}
		}
		o.Attributes["hashescrow.senderHash"] = hash

		return nil
	}
}

// WithHashFunc sets a hash function attribute to customize the lock command.
func WithHashFunc(hashFunc crypto.Hash) token.TransferOption {
	return func(o *token.TransferOptions) error {
		if o.Attributes == nil {
			o.Attributes = map[interface{}]interface{}{}
		}
		o.Attributes["hashescrow.hashFunc"] = hashFunc

		return nil
	}
}

// WithHashEncoding sets a hash encoding attribute to customize the lock command.
func WithHashEncoding(enc encoding.Encoding) token.TransferOption {
	return func(o *token.TransferOptions) error {
		if o.Attributes == nil {
			o.Attributes = map[interface{}]interface{}{}
		}
		o.Attributes["hashescrow.hashEncoding"] = enc

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

type Binder interface {
	Bind(ctx context.Context, longTerm view.Identity, ephemeral ...view.Identity) error
}

// Transaction holds a ttx transaction.
type Transaction struct {
	*ttx.Transaction
	Binder Binder
}

// PreImages carries the generated preimages for recipient and sender paths.
// If a hash is externally provided for a path, the corresponding preimage is nil.
type PreImages struct {
	Recipient []byte
	Sender    []byte
}

func NewTransaction(sp view.Context, signer view.Identity, opts ...ttx.TxOption) (*Transaction, error) {
	tx, err := ttx.NewTransaction(sp, signer, opts...)
	if err != nil {
		return nil, err
	}

	return &Transaction{Transaction: tx, Binder: endpoint.GetService(sp)}, nil
}

func NewAnonymousTransaction(sp view.Context, opts ...ttx.TxOption) (*Transaction, error) {
	tx, err := ttx.NewAnonymousTransaction(sp, opts...)
	if err != nil {
		return nil, err
	}

	return &Transaction{Transaction: tx, Binder: endpoint.GetService(sp)}, nil
}

func NewTransactionFromBytes(ctx view.Context, network, channel string, raw []byte) (*Transaction, error) {
	tx, err := ttx.NewTransactionFromBytes(ctx, raw)
	if err != nil {
		return nil, err
	}

	return &Transaction{Transaction: tx, Binder: endpoint.GetService(ctx)}, nil
}

// Lock appends a hash-based escrow lock action to the token request.
// This lock has no timeout semantics.
func (t *Transaction) Lock(ctx context.Context, wallet *token.OwnerWallet, sender view.Identity, typ token2.Type, value uint64, recipient view.Identity, opts ...token.TransferOption) ([]byte, error) {
	preImages, err := t.LockWithPreImages(ctx, wallet, sender, typ, value, recipient, opts...)
	if err != nil {
		return nil, err
	}

	return preImages.Recipient, nil
}

// LockWithPreImages appends a hash-based escrow lock action and returns both preimages.
func (t *Transaction) LockWithPreImages(ctx context.Context, wallet *token.OwnerWallet, sender view.Identity, typ token2.Type, value uint64, recipient view.Identity, opts ...token.TransferOption) (*PreImages, error) {
	options, err := compileTransferOptions(opts...)
	if err != nil {
		return nil, err
	}
	if recipient.IsNone() {
		return nil, errors.Errorf("must specify a recipient")
	}
	if sender == nil {
		sender, err = wallet.GetRecipientIdentity(ctx)
		if err != nil {
			return nil, errors.WithMessagef(err, "failed getting sender identity")
		}
	}

	var recipientHash []byte
	var senderHash []byte
	hashFunc := crypto.SHA256
	var hashEncoding encoding.Encoding
	if options.Attributes != nil {
		// backward compatibility
		boxed, ok := options.Attributes["hashescrow.hash"]
		if ok {
			recipientHash, ok = boxed.([]byte)
			if !ok {
				return nil, errors.Errorf("expected hashescrow.hash attribute to be []byte, got [%T]", boxed)
			}
		}
		boxed, ok = options.Attributes["hashescrow.recipientHash"]
		if ok {
			recipientHash, ok = boxed.([]byte)
			if !ok {
				return nil, errors.Errorf("expected hashescrow.recipientHash attribute to be []byte, got [%T]", boxed)
			}
		}
		boxed, ok = options.Attributes["hashescrow.senderHash"]
		if ok {
			senderHash, ok = boxed.([]byte)
			if !ok {
				return nil, errors.Errorf("expected hashescrow.senderHash attribute to be []byte, got [%T]", boxed)
			}
		}
		boxed, ok = options.Attributes["hashescrow.hashFunc"]
		if ok {
			hashFunc, ok = boxed.(crypto.Hash)
			if !ok {
				return nil, errors.Errorf("expected hashescrow.hashFunc attribute to be crypto.Hash, got [%T]", boxed)
			}
			if hashFunc == 0 {
				hashFunc = crypto.SHA256
			}
		}
		boxed, ok = options.Attributes["hashescrow.hashEncoding"]
		if ok {
			hashEncoding, ok = boxed.(encoding.Encoding)
			if !ok {
				return nil, errors.Errorf("expected hashescrow.hashEncoding attribute to be Encoding, got [%T]", boxed)
			}
		}
	}

	scriptID, preImages, script, err := t.recipientAsScript(sender, recipient, recipientHash, senderHash, hashFunc, hashEncoding)
	if err != nil {
		return nil, err
	}

	transferOpts := append(opts,
		token.WithTransferMetadata(LockKey(script.RecipientHashInfo.Hash), LockValue(script.RecipientHashInfo.Hash)),
		token.WithTransferMetadata(LockKey(script.SenderHashInfo.Hash), LockValue(script.SenderHashInfo.Hash)),
	)
	_, err = t.TokenRequest.Transfer(
		t.Context,
		wallet,
		typ,
		[]uint64{value},
		[]view.Identity{scriptID},
		transferOpts...,
	)
	if err != nil {
		return nil, err
	}

	return preImages, nil
}

// Claim appends a claim action to the token request.
// Spending can be submitted by anyone; recipient is resolved from the preimage.
func (t *Transaction) Claim(wallet *token.OwnerWallet, tok *token2.UnspentToken, preImage []byte, opts ...token.TransferOption) error {
	if len(preImage) == 0 {
		return errors.New("preImage is nil")
	}

	q, err := token2.ToQuantity(tok.Quantity, t.TokenRequest.TokenService.PublicParametersManager().PublicParameters().Precision())
	if err != nil {
		return errors.Wrapf(err, "failed to convert quantity [%s]", tok.Quantity)
	}
	owner, err := identity.UnmarshalTypedIdentity(tok.Owner)
	if err != nil {
		return err
	}
	if owner.Type != ScriptType {
		return errors.New("invalid owner type, expected hash escrow script")
	}
	script := &Script{}
	if err := json.Unmarshal(owner.Identity, script); err != nil {
		return errors.New("failed to unmarshal TypedIdentity as a hash escrow script")
	}

	claimIdentity, image, err := script.ResolveRecipientForPreImage(preImage)
	if err != nil {
		return errors.Wrap(err, "passed preImage does not match script hashes")
	}

	sigService := t.TokenService().SigService()
	if err := sigService.RegisterEphemeralSigner(
		t.Context,
		tok.Owner,
		&ClaimSigner{
			Preimage: preImage,
		},
		&ClaimVerifier{
			Script: script,
		},
	); err != nil {
		return err
	}

	if err := t.Binder.Bind(t.Context, claimIdentity, tok.Owner); err != nil {
		return err
	}

	return t.Transfer(
		wallet,
		tok.Type,
		[]uint64{q.ToBigInt().Uint64()},
		[]view.Identity{claimIdentity},
		append(opts, token.WithTokenIDs(&tok.Id), token.WithTransferMetadata(ClaimKey(image), preImage))...,
	)
}

func (t *Transaction) recipientAsScript(sender, recipient view.Identity, recipientHash []byte, senderHash []byte, hashFunc crypto.Hash, hashEncoding encoding.Encoding) (view.Identity, *PreImages, *Script, error) {
	preImages := &PreImages{}
	var err error

	recipientHashInfo, recipientPreImage, err := buildHashInfo(recipientHash, hashFunc, hashEncoding)
	if err != nil {
		return nil, nil, nil, errors.WithMessage(err, "failed preparing recipient hash info")
	}
	senderHashInfo, senderPreImage, err := buildHashInfo(senderHash, hashFunc, hashEncoding)
	if err != nil {
		return nil, nil, nil, errors.WithMessage(err, "failed preparing sender hash info")
	}
	preImages.Recipient = recipientPreImage
	preImages.Sender = senderPreImage

	logger.Debugf("recipient pair (pre-image, hash) = (%s,%s)", base64.StdEncoding.EncodeToString(preImages.Recipient), base64.StdEncoding.EncodeToString(recipientHashInfo.Hash))
	logger.Debugf("sender pair (pre-image, hash) = (%s,%s)", base64.StdEncoding.EncodeToString(preImages.Sender), base64.StdEncoding.EncodeToString(senderHashInfo.Hash))

	script := &Script{
		RecipientHashInfo: recipientHashInfo,
		SenderHashInfo:    senderHashInfo,
		Recipient:         recipient,
		Sender:            sender,
	}
	rawScript, err := json.Marshal(script)
	if err != nil {
		return nil, nil, nil, err
	}
	ro := &identity.TypedIdentity{
		Type:     ScriptType,
		Identity: rawScript,
	}
	raw, err := ro.Bytes()
	if err != nil {
		return nil, nil, nil, err
	}

	return raw, preImages, script, nil
}

func buildHashInfo(hash []byte, hashFunc crypto.Hash, hashEncoding encoding.Encoding) (HashInfo, []byte, error) {
	var preImage []byte
	h := hash

	if hashFunc == 0 {
		hashFunc = crypto.SHA256
	}

	if len(h) == 0 {
		var err error
		preImage, err = CreateNonce()
		if err != nil {
			return HashInfo{}, nil, err
		}
		digest := hashFunc.New()
		if _, err := digest.Write(preImage); err != nil {
			return HashInfo{}, nil, err
		}
		h = digest.Sum(nil)
		if hashEncoding != 0 {
			enc := hashEncoding.New()
			if enc == nil {
				return HashInfo{}, nil, errors.New("hashEncoding.New() returned nil")
			}
			h = []byte(enc.EncodeToString(h))
		}
	}

	return HashInfo{
		Hash:         h,
		HashFunc:     hashFunc,
		HashEncoding: hashEncoding,
	}, preImage, nil
}

// CreateNonce generates a nonce.
func CreateNonce() ([]byte, error) {
	nonce, err := getRandomNonce()

	return nonce, errors.WithMessagef(err, "error generating random nonce")
}

func getRandomNonce() ([]byte, error) {
	key := make([]byte, 24)
	_, err := rand.Read(key)
	if err != nil {
		return nil, errors.Wrap(err, "error getting random bytes")
	}

	return key, nil
}
