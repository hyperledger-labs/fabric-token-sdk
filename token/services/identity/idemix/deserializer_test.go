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
	kvs2 "github.com/hyperledger-labs/fabric-token-sdk/token/services/identity/storage/kvs"
	"github.com/hyperledger-labs/fabric-token-sdk/token/services/utils"
	"github.com/stretchr/testify/assert"
)

func TestNewDeserializer(t *testing.T) {
	testNewDeserializer(t, "./testdata/fp256bn_amcl/idemix", math.FP256BN_AMCL, false)
	testNewDeserializer(t, "./testdata/bls12_381_bbs/idemix", math.BLS12_381_BBS_GURVY, true)
}

func testNewDeserializer(t *testing.T, configPath string, curveID math.CurveID, aries bool) {
	t.Helper()
	// init
	backend, err := kvs2.NewInMemory()
	assert.NoError(t, err)
	config, err := crypto2.NewConfig(configPath)
	assert.NoError(t, err)
	keyStore, err := crypto2.NewKeyStore(curveID, kvs2.Keystore(backend))
	assert.NoError(t, err)
	cryptoProvider, err := crypto2.NewBCCSP(keyStore, curveID)
	assert.NoError(t, err)

	// key manager
	keyManager, err := NewKeyManager(config, types.EidNymRhNym, cryptoProvider)
	assert.NoError(t, err)
	assert.NotNil(t, keyManager)

	// get an identity
	identityDescriptor, err := keyManager.Identity(t.Context(), nil)
	assert.NoError(t, err)
	id := identityDescriptor.Identity
	auditInfoRaw := identityDescriptor.AuditInfo

	// instantiate a deserializer and check that it fils
	_, err = NewDeserializer(config.Ipk, -1)
	assert.Error(t, err)
	_, err = NewDeserializer(nil, curveID)
	assert.Error(t, err)
	_, err = NewDeserializer([]byte{}, curveID)
	assert.Error(t, err)
	_, err = NewDeserializer([]byte{0, 1, 2}, curveID)
	assert.Error(t, err)

	// instantiate a deserializer and validate it
	d, err := NewDeserializer(config.Ipk, curveID)
	assert.NoError(t, err)
	assert.NotNil(t, d)
	assert.Equal(t, fmt.Sprintf("Idemix with IPK [%s]", utils.Hashable(d.Ipk).String()), d.String())
	_, err = d.DeserializeVerifier(t.Context(), nil)
	assert.Error(t, err)
	_, err = d.DeserializeVerifier(t.Context(), []byte{})
	assert.Error(t, err)
	_, err = d.DeserializeVerifier(t.Context(), []byte{0, 1, 2, 3})
	assert.Error(t, err)
	verifier1, err := d.DeserializeVerifierAgainstNymEID(id, nil)
	assert.NoError(t, err)
	verifier2, err := d.DeserializeVerifier(t.Context(), id)
	assert.NoError(t, err)
	signer, err := keyManager.DeserializeSigner(t.Context(), id)
	assert.NoError(t, err)

	// sign and verify
	msg := []byte("test message")
	sigma, err := signer.Sign(msg)
	assert.NoError(t, err)
	err = verifier1.Verify(msg, sigma)
	assert.NoError(t, err)
	err = verifier2.Verify(msg, sigma)
	assert.NoError(t, err)

	// check audit info
	auditInfo, err := d.DeserializeAuditInfo(t.Context(), auditInfoRaw)
	assert.NoError(t, err)
	assert.NotNil(t, auditInfo)
	assert.Equal(t, "alice", auditInfo.EnrollmentID())
	assert.Equal(t, "150", auditInfo.RevocationHandle())
	auditInfoDeser := &AuditInfoDeserializer{}
	// check invalid input
	_, err = auditInfoDeser.DeserializeAuditInfo(t.Context(), nil)
	assert.Error(t, err)
	_, err = auditInfoDeser.DeserializeAuditInfo(t.Context(), []byte{})
	assert.Error(t, err)
	_, err = auditInfoDeser.DeserializeAuditInfo(t.Context(), []byte{0, 1, 2, 3})
	assert.Error(t, err)
	auditInfo2, err := auditInfoDeser.DeserializeAuditInfo(t.Context(), auditInfoRaw)
	assert.NoError(t, err)
	assert.Equal(t, "alice", auditInfo2.EnrollmentID())
	assert.Equal(t, "150", auditInfo2.RevocationHandle())

	// match audit info
	auditInfoMatcher, err := d.GetAuditInfoMatcher(t.Context(), id, auditInfoRaw)
	assert.NoError(t, err)
	assert.NotNil(t, auditInfoMatcher)
	assert.NoError(t, auditInfoMatcher.Match(t.Context(), id))
	assert.NoError(t, d.MatchIdentity(t.Context(), id, auditInfoRaw))

	// check info
	info, err := d.Info(t.Context(), id, []byte{})
	assert.NoError(t, err)
	assert.True(t, strings.HasPrefix(info, "Idemix: []"))
	info, err = d.Info(t.Context(), id, nil)
	assert.NoError(t, err)
	assert.True(t, strings.HasPrefix(info, "Idemix: []"))
	_, err = d.Info(t.Context(), id, []byte{0, 1, 2})
	assert.Error(t, err)
	info, err = d.Info(t.Context(), id, auditInfoRaw)
	assert.NoError(t, err)
	assert.True(t, strings.HasPrefix(info, "Idemix: [alice]"))
}
