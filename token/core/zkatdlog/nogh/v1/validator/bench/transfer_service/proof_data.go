/*
Copyright IBM Corp All Rights Reserved.

SPDX-License-Identifier: Apache-2.0
*/
package bench

import (
	"fmt"

	v1 "github.com/hyperledger-labs/fabric-token-sdk/token/core/zkatdlog/nogh/v1/setup"
)

type TokenData struct {
	PubParams       *v1.PublicParams
	TokenRequestRaw []byte
	TxID            string
}

func (p *TokenData) ToWire() (*WireTokenData, error) {
	ppRaw, err := p.PubParams.Serialize()
	if err != nil {
		return nil, fmt.Errorf("failed to serialize public parameters: %w", err)
	}

	return &WireTokenData{
		PubParamsRaw:    ppRaw,
		TokenRequestRaw: p.TokenRequestRaw,
		TxID:            p.TxID,
	}, nil
}

// WireTokenData is the JSON-safe representation of ProofData for transport.
type WireTokenData struct {
	PubParamsRaw    []byte `json:"pub_params_raw"`
	TokenRequestRaw []byte `json:"token_request_raw"`
	TxID            string `json:"tx_id"`
}

func (w *WireTokenData) Deserialize() (*TokenData, error) {
	pp, err := v1.NewPublicParamsFromBytes(w.PubParamsRaw, v1.DLogNoGHDriverName, v1.ProtocolV1)
	if err != nil {
		return nil, fmt.Errorf("failed to deserialize public parameters: %w", err)
	}

	return &TokenData{
		PubParams:       pp,
		TokenRequestRaw: w.TokenRequestRaw,
		TxID:            w.TxID,
	}, nil
}
