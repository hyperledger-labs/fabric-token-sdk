/*
Copyright IBM Corp. All Rights Reserved.

SPDX-License-Identifier: Apache-2.0
*/

package htlc

import (
	"bytes"
	"context"
	"encoding/json"
	"time"

	"github.com/hyperledger-labs/fabric-smart-client/platform/common/utils/collections/iterators"
	"github.com/hyperledger-labs/fabric-smart-client/platform/view/view"
	"github.com/hyperledger-labs/fabric-token-sdk/token/services/identity"
	"github.com/hyperledger-labs/fabric-token-sdk/token/services/logging"
	"github.com/hyperledger-labs/fabric-token-sdk/token/token"
)

// SelectFunction is the prototype of a function to select pairs (token,script)
type SelectFunction = func(context.Context, *token.UnspentToken, *Script) (bool, error)

// PreImageSelector selects htlc-tokens that match a given pre-image
type PreImageSelector struct {
	preImage []byte
}

func (f *PreImageSelector) Filter(ctx context.Context, tok *token.UnspentToken, script *Script) (bool, error) {
	logger.DebugfContext(ctx, "token [%s,%s,%s,%s] contains a script? Yes", tok.Id, view.Identity(tok.Owner).UniqueID(), tok.Type, tok.Quantity)

	if !script.HashInfo.HashFunc.Available() {
		logger.ErrorfContext(ctx, "script hash function not available [%d]", script.HashInfo.HashFunc)
		return false, nil
	}
	hash := script.HashInfo.HashFunc.New()
	if _, err := hash.Write(f.preImage); err != nil {
		return false, err
	}
	h := hash.Sum(nil)
	h = []byte(script.HashInfo.HashEncoding.New().EncodeToString(h))

	logger.DebugfContext(ctx, "searching for script matching (pre-image, image) = (%s,%s)", logging.Base64(f.preImage), logging.Base64(h))

	// does the preimage match?
	logger.DebugfContext(ctx, "token [%s,%s,%s,%s] does hashes match?", tok.Id, view.Identity(tok.Owner), tok.Type, tok.Quantity, logging.Base64(h), logging.Base64(script.HashInfo.Hash))

	return bytes.Equal(h, script.HashInfo.Hash), nil
}

// SelectExpired selects expired htlc-tokens
func SelectExpired(ctx context.Context, tok *token.UnspentToken, script *Script) (bool, error) {
	logger.DebugfContext(ctx, "token [%s,%s,%s,%s] contains a script? Yes", tok.Id, view.Identity(tok.Owner).UniqueID(), tok.Type, tok.Quantity)
	now := time.Now()
	logger.DebugfContext(ctx, "[%v]<=[%v], sender [%s], recipient [%s]?", script.Deadline, now, script.Sender.UniqueID(), script.Recipient.UniqueID())
	return script.Deadline.Before(now), nil
}

// SelectNonExpired selects non-expired htlc-tokens
func SelectNonExpired(ctx context.Context, tok *token.UnspentToken, script *Script) (bool, error) {
	now := time.Now()
	logger.DebugfContext(ctx, "[%v]>=[%v], sender [%s], recipient [%s]?", script.Deadline, now, script.Sender.UniqueID(), script.Recipient.UniqueID())
	return script.Deadline.After(now), nil
}

// ExpiredAndHashSelector selects expired htlc-tokens with a specific hash
type ExpiredAndHashSelector struct {
	Hash []byte
}

func (s *ExpiredAndHashSelector) Select(ctx context.Context, tok *token.UnspentToken, script *Script) (bool, error) {
	logger.DebugfContext(ctx, "token [%s,%s,%s,%s] contains a script? Yes", tok.Id, view.Identity(tok.Owner).UniqueID(), tok.Type, tok.Quantity)
	now := time.Now()
	logger.DebugfContext(ctx, "[%v]<=[%v], sender [%s], recipient [%s]?", script.Deadline, now, script.Sender.UniqueID(), script.Recipient.UniqueID())
	return script.Deadline.Before(now) && bytes.Equal(script.HashInfo.Hash, s.Hash), nil
}

func IsScript(ctx context.Context, selector SelectFunction) iterators.Predicate[*token.UnspentToken] {
	return func(tok *token.UnspentToken) bool {
		owner, err := identity.UnmarshalTypedIdentity(tok.Owner)
		if err != nil {
			logger.DebugfContext(ctx, "Is Mine [%s,%s,%s]? No, failed unmarshalling [%s]", view.Identity(tok.Owner), tok.Type, tok.Quantity, err)
			return false
		}
		if owner.Type != ScriptType {
			return false
		}

		script := &Script{}
		if err := json.Unmarshal(owner.Identity, script); err != nil {
			logger.DebugfContext(ctx, "token [%s,%s,%s,%s] contains a script? No", tok.Id, view.Identity(tok.Owner).UniqueID(), tok.Type, tok.Quantity)
			return false
		}
		if script.Sender.IsNone() {
			logger.DebugfContext(ctx, "token [%s,%s,%s,%s] contains a script? No", tok.Id, view.Identity(tok.Owner).UniqueID(), tok.Type, tok.Quantity)
			return false
		}
		logger.DebugfContext(ctx, "token [%s,%s,%s,%s] contains a script? Yes", tok.Id, view.Identity(tok.Owner).UniqueID(), tok.Type, tok.Quantity)

		pickItem, err := selector(ctx, tok, script)
		if err != nil {
			logger.ErrorfContext(ctx, "failed to select (token,script)[%v:%v] pair: %w", tok, script, err)
			return false
		}
		return pickItem
	}
}
