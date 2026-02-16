/*
Copyright IBM Corp. All Rights Reserved.

SPDX-License-Identifier: Apache-2.0
*/

package idemix

import (
	"fmt"
	"strings"
	"testing"

	"github.com/IBM/idemix/bccsp/types"
	math "github.com/IBM/mathlib"
	crypto2 "github.com/hyperledger-labs/fabric-token-sdk/token/services/identity/idemix/crypto"
	kvs2 "github.com/hyperledger-labs/fabric-token-sdk/token/services/storage/db/kvs"
	"github.com/hyperledger-labs/fabric-token-sdk/token/services/utils"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestNewDeserializer(t *testing.T) {
	testNewDeserializer(t, "./testdata/fp256bn_amcl/idemix", math.FP256BN_AMCL)
	testNewDeserializer(t, "./testdata/bls12_381_bbs/idemix", math.BLS12_381_BBS_GURVY)
}

func testNewDeserializer(t *testing.T, configPath string, curveID math.CurveID) {
	t.Helper()
	// init
	backend, err := kvs2.NewInMemory()
	require.NoError(t, err)
	config, err := crypto2.NewConfig(configPath)
	require.NoError(t, err)
	keyStore, err := crypto2.NewKeyStore(curveID, kvs2.Keystore(backend))
	require.NoError(t, err)
	cryptoProvider, err := crypto2.NewBCCSP(keyStore, curveID)
	require.NoError(t, err)

	// key manager
	keyManager, err := NewKeyManager(config, types.EidNymRhNym, cryptoProvider)
	require.NoError(t, err)
	assert.NotNil(t, keyManager)

	// get an identity
	identityDescriptor, err := keyManager.Identity(t.Context(), nil)
	require.NoError(t, err)
	id := identityDescriptor.Identity
	auditInfoRaw := identityDescriptor.AuditInfo

	// instantiate a deserializer and check that it fils
	_, err = NewDeserializer(config.Ipk, -1)
	require.Error(t, err)
	_, err = NewDeserializer(nil, curveID)
	require.Error(t, err)
	_, err = NewDeserializer([]byte{}, curveID)
	require.Error(t, err)
	_, err = NewDeserializer([]byte{0, 1, 2}, curveID)
	require.Error(t, err)

	// instantiate a deserializer and validate it
	d, err := NewDeserializer(config.Ipk, curveID)
	require.NoError(t, err)
	assert.NotNil(t, d)
	assert.Equal(t, fmt.Sprintf("Idemix with IPK [%s]", utils.Hashable(d.Ipk).String()), d.String())
	_, err = d.DeserializeVerifier(t.Context(), nil)
	require.Error(t, err)
	_, err = d.DeserializeVerifier(t.Context(), []byte{})
	require.Error(t, err)
	_, err = d.DeserializeVerifier(t.Context(), []byte{0, 1, 2, 3})
	require.Error(t, err)
	verifier1, err := d.DeserializeVerifierAgainstNymEID(id, nil)
	require.NoError(t, err)
	verifier2, err := d.DeserializeVerifier(t.Context(), id)
	require.NoError(t, err)
	signer, err := keyManager.DeserializeSigner(t.Context(), id)
	require.NoError(t, err)

	// sign and verify
	msg := []byte("test message")
	sigma, err := signer.Sign(msg)
	require.NoError(t, err)
	err = verifier1.Verify(msg, sigma)
	require.NoError(t, err)
	err = verifier2.Verify(msg, sigma)
	require.NoError(t, err)

	// check audit info
	auditInfo, err := d.DeserializeAuditInfo(t.Context(), auditInfoRaw)
	require.NoError(t, err)
	assert.NotNil(t, auditInfo)
	assert.Equal(t, "alice", auditInfo.EnrollmentID())
	assert.Equal(t, "150", auditInfo.RevocationHandle())
	auditInfoDeser := &AuditInfoDeserializer{}
	// check invalid input
	_, err = auditInfoDeser.DeserializeAuditInfo(t.Context(), nil)
	require.Error(t, err)
	_, err = auditInfoDeser.DeserializeAuditInfo(t.Context(), []byte{})
	require.Error(t, err)
	_, err = auditInfoDeser.DeserializeAuditInfo(t.Context(), []byte{0, 1, 2, 3})
	require.Error(t, err)
	auditInfo2, err := auditInfoDeser.DeserializeAuditInfo(t.Context(), auditInfoRaw)
	require.NoError(t, err)
	assert.Equal(t, "alice", auditInfo2.EnrollmentID())
	assert.Equal(t, "150", auditInfo2.RevocationHandle())

	// match audit info
	auditInfoMatcher, err := d.GetAuditInfoMatcher(t.Context(), id, auditInfoRaw)
	require.NoError(t, err)
	assert.NotNil(t, auditInfoMatcher)
	require.NoError(t, auditInfoMatcher.Match(t.Context(), id))
	require.NoError(t, d.MatchIdentity(t.Context(), id, auditInfoRaw))

	// check info
	info, err := d.Info(t.Context(), id, []byte{})
	require.NoError(t, err)
	assert.True(t, strings.HasPrefix(info, "Idemix: []"))
	info, err = d.Info(t.Context(), id, nil)
	require.NoError(t, err)
	assert.True(t, strings.HasPrefix(info, "Idemix: []"))
	_, err = d.Info(t.Context(), id, []byte{0, 1, 2})
	require.Error(t, err)
	info, err = d.Info(t.Context(), id, auditInfoRaw)
	require.NoError(t, err)
	assert.True(t, strings.HasPrefix(info, "Idemix: [alice]"))
}
