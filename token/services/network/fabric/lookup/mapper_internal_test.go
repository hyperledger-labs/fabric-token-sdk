/*
Copyright IBM Corp. All Rights Reserved.

SPDX-License-Identifier: Apache-2.0
*/

package lookup

import (
	"testing"

	"github.com/LFDT-Panurus/panurus/token/services/network/common/rws/keys"
	vault2 "github.com/hyperledger-labs/fabric-smart-client/platform/common/core/generic/vault"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// A transaction can write several transfer-metadata keys in one namespace, such as two htlc
// claim keys; every one of them must be reported so that each key's listener is notified
// (#2517).
func TestMapTxInfoReportsEveryMatchingKey(t *testing.T) {
	kt := &keys.Translator{}
	prefix, err := kt.TransferActionMetadataKeyPrefix()
	require.NoError(t, err)
	keyA, err := kt.CreateTransferActionMetadataKey("claim-a")
	require.NoError(t, err)
	keyB, err := kt.CreateTransferActionMetadataKey("claim-b")
	require.NoError(t, err)
	keyC, err := kt.CreateTransferActionMetadataKey("claim-c")
	require.NoError(t, err)

	m := &endorserTxInfoMapper{network: "n", prefixes: []string{prefix}}
	rwSet := vault2.ReadWriteSet{Writes: vault2.Writes{
		"ns1": vault2.NamespaceWrites{
			keyA:        []byte("preimage-a"),
			keyB:        []byte("preimage-b"),
			"other-key": []byte("ignored"),
		},
		"ns2": vault2.NamespaceWrites{
			keyC: []byte("preimage-c"),
		},
	}}

	infos, err := m.mapTxInfo(rwSet, "tx1")
	require.NoError(t, err)

	got := map[string]KeyInfo{}
	for _, info := range infos {
		got[info.Key] = info
	}
	assert.Equal(t, map[string]KeyInfo{
		keyA: {Namespace: "ns1", Key: keyA, Value: []byte("preimage-a")},
		keyB: {Namespace: "ns1", Key: keyB, Value: []byte("preimage-b")},
		keyC: {Namespace: "ns2", Key: keyC, Value: []byte("preimage-c")},
	}, got)
}
