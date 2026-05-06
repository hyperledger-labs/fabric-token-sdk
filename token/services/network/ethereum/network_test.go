/*
Copyright IBM Corp. All Rights Reserved.

SPDX-License-Identifier: Apache-2.0
*/

package ethereum

import (
	"context"
	"testing"
	"time"

	token2 "github.com/hyperledger-labs/fabric-token-sdk/token"
	"github.com/hyperledger-labs/fabric-token-sdk/token/services/network/driver"
	"github.com/hyperledger-labs/fabric-token-sdk/token/token"
	"github.com/stretchr/testify/require"
)

func TestNewDriver(t *testing.T) {
	d := NewDriver()
	require.NotNil(t, d)

	nw, err := d.New("evm-net", "chain-a")
	require.NoError(t, err)
	require.IsType(t, &Network{}, nw)
}

func TestEnvelopeRoundTrip(t *testing.T) {
	env := &Envelope{Tx: "tx-id"}

	raw, err := env.Bytes()
	require.NoError(t, err)

	other := &Envelope{}
	require.NoError(t, other.FromBytes(raw))
	require.Equal(t, "tx-id", other.TxID())
	require.Contains(t, other.String(), "tx-id")
}

func TestNetworkScaffold(t *testing.T) {
	n := NewNetwork("evm-net", "chain-a")

	require.Equal(t, "evm-net", n.Name())
	require.Equal(t, "chain-a", n.Channel())

	opts, err := n.Normalize(&token2.ServiceOptions{})
	require.NoError(t, err)
	require.Equal(t, "evm-net", opts.Network)
	require.Equal(t, "chain-a", opts.Channel)

	txID := n.ComputeTxID(&driver.TxID{Nonce: []byte("nonce"), Creator: []byte("creator")})
	require.Len(t, txID, 64)

	_, err = n.Connect("ns")
	require.ErrorIs(t, err, ErrNotImplemented)

	err = n.Broadcast(context.Background(), []byte("payload"))
	require.ErrorIs(t, err, ErrNotImplemented)

	_, err = n.RequestApproval(nil, nil, nil, nil, driver.TxID{})
	require.ErrorIs(t, err, ErrNotImplemented)

	_, err = n.FetchPublicParameters("ns")
	require.ErrorIs(t, err, ErrNotImplemented)

	_, err = n.QueryTokens(context.Background(), "ns", []*token.ID{{TxId: "tx", Index: 1}})
	require.ErrorIs(t, err, ErrNotImplemented)

	_, err = n.AreTokensSpent(context.Background(), "ns", []*token.ID{{TxId: "tx", Index: 1}}, nil)
	require.ErrorIs(t, err, ErrNotImplemented)

	err = n.AddFinalityListener("ns", "tx", nil)
	require.ErrorIs(t, err, ErrNotImplemented)

	status, tokenRequestHash, message, err := n.GetTransactionStatus(context.Background(), "ns", "tx")
	require.ErrorIs(t, err, ErrNotImplemented)
	require.Equal(t, driver.Unknown, status)
	require.Nil(t, tokenRequestHash)
	require.Empty(t, message)

	_, err = n.LookupTransferMetadataKey("ns", "k", time.Second)
	require.ErrorIs(t, err, ErrNotImplemented)

	_, err = n.Ledger()
	require.ErrorIs(t, err, ErrNotImplemented)

	require.NotNil(t, n.NewEnvelope())
	require.NotNil(t, n.LocalMembership())

	_, err = n.LocalMembership().AnonymousIdentity()
	require.ErrorIs(t, err, ErrNotImplemented)
}
