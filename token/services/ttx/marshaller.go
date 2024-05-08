/*
Copyright IBM Corp. All Rights Reserved.

SPDX-License-Identifier: Apache-2.0
*/

package ttx

import (
	"encoding/asn1"
	"sort"

	"github.com/hyperledger-labs/fabric-smart-client/platform/view/services/hash"
	"github.com/hyperledger-labs/fabric-token-sdk/token/services/network"
	"go.uber.org/zap/zapcore"

	"github.com/hyperledger-labs/fabric-token-sdk/token"
	jsoniter "github.com/json-iterator/go"
	"github.com/pkg/errors"
)

func Marshal(v interface{}) ([]byte, error) {
	return jsoniter.ConfigCompatibleWithStandardLibrary.Marshal(v)
}

func Unmarshal(data []byte, v interface{}) error {
	return jsoniter.ConfigCompatibleWithStandardLibrary.Unmarshal(data, v)
}

func MarshalMeta(v map[string][]byte) ([]byte, error) {
	metaSer := metaSer{
		Keys: make([]string, len(v)),
		Vals: make([][]byte, len(v)),
	}

	i := 0
	for k := range v {
		metaSer.Keys[i] = k
		i++
	}
	i = 0
	sort.Strings(metaSer.Keys)
	for _, key := range metaSer.Keys {
		metaSer.Vals[i] = v[key]
		i++
	}
	return asn1.Marshal(metaSer)
}

func UnmarshalMeta(raw []byte) (map[string][]byte, error) {
	var metaSer metaSer
	_, err := asn1.Unmarshal(raw, &metaSer)
	if err != nil {
		return nil, err
	}
	v := make(map[string][]byte, len(metaSer.Keys))
	for i, k := range metaSer.Keys {
		v[k] = metaSer.Vals[i]
	}
	return v, nil
}

type metaSer struct {
	Keys []string
	Vals [][]byte
}

func RecipientDataBytes(r *token.RecipientData) ([]byte, error) {
	return Marshal(r)
}

func RecipientDataFromBytes(raw []byte) (*token.RecipientData, error) {
	rd := &token.RecipientData{}
	if err := Unmarshal(raw, rd); err != nil {
		return nil, err
	}
	return rd, nil
}

type GetNetworkFunc = func(network string, channel string) (*network.Network, error)

type TransactionSer struct {
	Nonce        []byte
	Creator      []byte
	ID           string
	Network      string
	Channel      string
	Namespace    string
	Signer       []byte
	Transient    []byte
	TokenRequest []byte
	Envelope     []byte
}

func marshal(t *Transaction, eIDs ...string) ([]byte, error) {
	var err error

	var transientRaw []byte
	if len(t.Payload.Transient) != 0 {
		transientRaw, err = MarshalMeta(t.Payload.Transient)
		if err != nil {
			return nil, errors.Wrap(err, "failed to marshal transient")
		}
	}

	var tokenRequestRaw []byte
	if t.Payload.TokenRequest != nil {
		req := t.Payload.TokenRequest
		// If eIDs are specified, we only marshal the metadata for the passed eIDs
		if len(eIDs) != 0 {
			req, err = t.Payload.TokenRequest.FilterMetadataBy(eIDs...)
			if err != nil {
				return nil, errors.Wrap(err, "failed to filter metadata")
			}
		}
		tokenRequestRaw, err = req.Bytes()
		if err != nil {
			return nil, errors.Wrap(err, "failed to marshal token request")
		}
	}

	var envRaw []byte
	if t.Payload.Envelope != nil {
		envRaw, err = t.Envelope.Bytes()
		if err != nil {
			return nil, errors.Wrap(err, "failed to marshal envelope")
		}
		if logger.IsEnabledFor(zapcore.DebugLevel) {
			logger.Debugf("transaction envelope [%s]", hash.Hashable(t.Envelope.String()))
		}
	}

	res, err := asn1.Marshal(TransactionSer{
		Nonce:        t.Payload.TxID.Nonce,
		Creator:      t.Payload.TxID.Creator,
		ID:           t.Payload.ID,
		Network:      t.Payload.Network,
		Channel:      t.Payload.Channel,
		Namespace:    t.Payload.Namespace,
		Signer:       t.Payload.Signer,
		Transient:    transientRaw,
		TokenRequest: tokenRequestRaw,
		Envelope:     envRaw,
	})
	if err != nil {
		return nil, errors.Wrap(err, "failed to marshal transaction")
	}
	return res, nil
}

func unmarshal(getNetwork GetNetworkFunc, p *Payload, raw []byte) error {
	var ser TransactionSer
	if _, err := asn1.Unmarshal(raw, &ser); err != nil {
		return errors.Wrapf(err, "failed unmarshalling transaction [%s]", string(raw))
	}

	p.TxID.Nonce = ser.Nonce
	p.TxID.Creator = ser.Creator
	p.ID = ser.ID
	p.Network = ser.Network
	p.Channel = ser.Channel
	p.Namespace = ser.Namespace
	p.Signer = ser.Signer
	p.Transient = make(map[string][]byte)
	if len(ser.Transient) != 0 {
		meta, err := UnmarshalMeta(ser.Transient)
		if err != nil {
			return errors.Wrap(err, "failed unmarshalling transient")
		}
		p.Transient = meta
	}
	if len(ser.TokenRequest) != 0 {
		if err := p.TokenRequest.FromBytes(ser.TokenRequest); err != nil {
			return errors.Wrap(err, "failed unmarshalling token request")
		}
	}
	if p.Envelope == nil {
		nws, err := getNetwork(p.Network, p.Channel)
		if err != nil {
			return err
		}
		p.Envelope = nws.NewEnvelope()
	}
	if len(ser.Envelope) != 0 {
		if err := p.Envelope.FromBytes(ser.Envelope); err != nil {
			return errors.Wrapf(err, "failed unmarshalling envelope [%d]", len(ser.Envelope))
		}
	}
	return nil
}
