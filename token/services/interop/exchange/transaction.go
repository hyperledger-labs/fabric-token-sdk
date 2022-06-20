/*
Copyright IBM Corp. All Rights Reserved.

SPDX-License-Identifier: Apache-2.0
*/

package exchange

import (
	"crypto"
	"crypto/rand"
	"encoding/base64"
	"encoding/json"
	"time"

	"github.com/hyperledger-labs/fabric-smart-client/platform/view/view"
	"github.com/hyperledger-labs/fabric-token-sdk/token"
	"github.com/hyperledger-labs/fabric-token-sdk/token/core/identity"
	"github.com/hyperledger-labs/fabric-token-sdk/token/services/interop/encoding"
	"github.com/hyperledger-labs/fabric-token-sdk/token/services/ttx"
	"github.com/pkg/errors"
)

const (
	ScriptTypeExchange    = "exchange" // exchange script
	defaultDeadlineOffset = time.Hour
)

func compileTransferOptions(opts ...token.TransferOption) (*token.TransferOptions, error) {
	txOptions := &token.TransferOptions{}
	for _, opt := range opts {
		if err := opt(txOptions); err != nil {
			return nil, err
		}
	}
	return txOptions, nil
}

type Transaction struct {
	*ttx.Transaction
}

func (t *Transaction) Exchange(wallet *token.OwnerWallet, sender view.Identity, typ string, value uint64, recipient view.Identity, deadline time.Duration, opts ...token.TransferOption) ([]byte, error) {
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
			return nil, errors.WithMessagef(err, "failed getting owner identity")
		}
	}

	var hash []byte
	var hashFunc crypto.Hash
	var hashEncoding encoding.Encoding
	if options.Attributes != nil {
		boxed, ok := options.Attributes["exchange.hash"]
		if ok {
			hash, ok = boxed.([]byte)
			if !ok {
				return nil, errors.Errorf("expected exchange.hash attribute to be []byte, got [%T]", boxed)
			}
		}
		boxed, ok = options.Attributes["exchange.hashFunc"]
		if ok {
			hashFunc, ok = boxed.(crypto.Hash)
			if !ok {
				return nil, errors.Errorf("expected exchange.hashFunc attribute to be crypto.Hash, got [%T]", boxed)
			}
		}
		boxed, ok = options.Attributes["exchange.hashEncoding"]
		if ok {
			hashEncoding, ok = boxed.(encoding.Encoding)
			if !ok {
				return nil, errors.Errorf("expected exchange.hashEncoding attribute to be Encoding, got [%T]", boxed)
			}
		}
	}
	if hashFunc == 0 {
		hashFunc = crypto.SHA256 // default hash function
	}
	script, preImage, err := t.recipientAsScript(sender, recipient, deadline, hash, hashFunc, hashEncoding)
	if err != nil {
		return nil, err
	}
	_, err = t.TokenRequest.Transfer(wallet, typ, []uint64{value}, []view.Identity{script}, opts...)
	if err != nil {
		return nil, err
	}

	return preImage, nil
}

func (t *Transaction) recipientAsScript(sender, recipient view.Identity, deadline time.Duration, h []byte, hashFunc crypto.Hash, hashEncoding encoding.Encoding) (view.Identity, []byte, error) {
	// sample pre-image and its hash
	var preImage []byte
	var err error
	if len(h) == 0 {
		preImage, err = CreateNonce()
		if err != nil {
			return nil, nil, err
		}
		hash := hashFunc.New()
		if _, err := hash.Write(preImage); err != nil {
			return nil, nil, err
		}
		h = hash.Sum(nil)
		// no need to encode if encoding is none (=0)
		if hashEncoding != 0 {
			h = []byte(hashEncoding.New().EncodeToString(h))
		}
	}

	logger.Debugf("pair (pre-image, hash) = (%s,%s)", base64.StdEncoding.EncodeToString(preImage), base64.StdEncoding.EncodeToString(h))

	script := Script{
		HashFunc:     hashFunc,
		HashEncoding: hashEncoding,
		Hash:         h,
		Deadline:     time.Now().Add(deadline),
		Recipient:    recipient,
		Sender:       sender,
	}
	rawScript, err := json.Marshal(script)
	if err != nil {
		return nil, nil, err
	}
	ro := &identity.RawOwner{
		Type:     ScriptTypeExchange,
		Identity: rawScript,
	}
	raw, err := identity.MarshallRawOwner(ro)
	if err != nil {
		return nil, nil, err
	}
	return raw, preImage, nil
}

// CreateNonce generates a nonce using the common/crypto package.
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
