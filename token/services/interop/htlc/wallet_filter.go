/*
Copyright IBM Corp. All Rights Reserved.

SPDX-License-Identifier: Apache-2.0
*/

package htlc

import (
	"bytes"
	"encoding/base64"
	"time"

	"github.com/hyperledger-labs/fabric-smart-client/platform/view/view"
	"github.com/hyperledger-labs/fabric-token-sdk/token/token"
	"go.uber.org/zap/zapcore"
)

// PickFunction is a prototype for (token,script) pair selection
type PickFunction = func(*token.UnspentToken, *Script) (bool, error)

type PreImageFilter struct {
	preImage []byte
}

func (f *PreImageFilter) Filter(tok *token.UnspentToken, script *Script) (bool, error) {
	logger.Debugf("token [%s,%s,%s,%s] contains a script? Yes", tok.Id, view.Identity(tok.Owner.Raw).UniqueID(), tok.Type, tok.Quantity)

	if !script.HashInfo.HashFunc.Available() {
		logger.Errorf("script hash function not available [%d]", script.HashInfo.HashFunc)
		return false, nil
	}
	hash := script.HashInfo.HashFunc.New()
	if _, err := hash.Write(f.preImage); err != nil {
		return false, err
	}
	h := hash.Sum(nil)
	h = []byte(script.HashInfo.HashEncoding.New().EncodeToString(h))

	if logger.IsEnabledFor(zapcore.DebugLevel) {
		logger.Debugf("searching for script matching (pre-image, image) = (%s,%s)",
			base64.StdEncoding.EncodeToString(f.preImage),
			base64.StdEncoding.EncodeToString(h),
		)
	}

	// does the preimage match?
	logger.Debugf("token [%s,%s,%s,%s] does hashes match?", tok.Id, view.Identity(tok.Owner.Raw).UniqueID(), tok.Type, tok.Quantity,
		base64.StdEncoding.EncodeToString(h), base64.StdEncoding.EncodeToString(script.HashInfo.Hash))

	return bytes.Equal(h, script.HashInfo.Hash), nil
}

func DeadlineBefore(tok *token.UnspentToken, script *Script) (bool, error) {
	logger.Debugf("token [%s,%s,%s,%s] contains a script? Yes", tok.Id, view.Identity(tok.Owner.Raw).UniqueID(), tok.Type, tok.Quantity)

	// is this expired and I am the sender?
	now := time.Now()
	logger.Debugf("[%v]<=[%v], sender [%s], recipient [%s]?", script.Deadline, now, script.Sender.UniqueID(), script.Recipient.UniqueID())
	return script.Deadline.Before(now), nil
}

func DeadlineAfter(tok *token.UnspentToken, script *Script) (bool, error) {
	now := time.Now()
	logger.Debugf("[%v]>=[%v], sender [%s], recipient [%s]?", script.Deadline, now, script.Sender.UniqueID(), script.Recipient.UniqueID())
	return script.Deadline.After(now), nil
}
