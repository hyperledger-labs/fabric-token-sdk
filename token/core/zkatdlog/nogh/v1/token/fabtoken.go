/*
Copyright IBM Corp. All Rights Reserved.

SPDX-License-Identifier: Apache-2.0
*/

package token

import (
	"github.com/hyperledger-labs/fabric-smart-client/pkg/utils/errors"
	"github.com/hyperledger-labs/fabric-token-sdk/token/core/fabtoken/v1/actions"
	"github.com/hyperledger-labs/fabric-token-sdk/token/token"
)

func ParseFabtokenToken(tok []byte, precision uint64, maxPrecision uint64) (*actions.Output, uint64, error) {
	if precision < maxPrecision {
		return nil, 0, errors.Errorf("unsupported precision [%d], max [%d]", precision, maxPrecision)
	}

	output := &actions.Output{}
	err := output.Deserialize(tok)
	if err != nil {
		return nil, 0, errors.Wrap(err, "failed to unmarshal fabtoken")
	}
	q, err := token.NewUBigQuantity(output.Quantity, precision)
	if err != nil {
		return nil, 0, errors.Wrap(err, "failed to create quantity")
	}

	return output, q.Uint64(), nil
}
