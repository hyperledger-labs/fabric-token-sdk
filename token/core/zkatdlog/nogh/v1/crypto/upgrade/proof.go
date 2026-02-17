/*
Copyright IBM Corp. All Rights Reserved.

SPDX-License-Identifier: Apache-2.0
*/

package upgrade

import (
	errors2 "errors"

	"github.com/hyperledger-labs/fabric-smart-client/pkg/utils/errors"
	"github.com/hyperledger-labs/fabric-token-sdk/token/core/common/encoding/json"
	"github.com/hyperledger-labs/fabric-token-sdk/token/driver"
	"github.com/hyperledger-labs/fabric-token-sdk/token/services/utils"
	"github.com/hyperledger-labs/fabric-token-sdk/token/token"
)

type Proof struct {
	Challenge  driver.TokensUpgradeChallenge
	Tokens     []token.LedgerToken
	Signatures []Signature
}

func (p *Proof) Serialize() ([]byte, error) {
	return json.Marshal(p)
}

func (p *Proof) Deserialize(raw []byte) error {
	return json.Unmarshal(raw, p)
}

func SHA256Digest(ch driver.TokensUpgradeChallenge, tokens []token.LedgerToken) ([]byte, error) {
	h := utils.NewSHA256Hasher()
	err := h.AddBytes(ch)
	if err != nil {
		return nil, errors.Wrapf(err, "failed to write challenge to hash")
	}
	for _, token := range tokens {
		if err := errors2.Join(
			h.AddString(token.ID.TxId),
			h.AddUInt64(token.ID.Index),
			h.AddBytes(token.Token),
			h.AddBytes(token.TokenMetadata),
			h.AddString(string(token.Format)),
		); err != nil {
			return nil, errors.Wrapf(err, "failed to write token to hash")
		}
	}

	return h.Digest(), nil
}
