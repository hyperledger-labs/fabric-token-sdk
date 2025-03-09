/*
Copyright IBM Corp. All Rights Reserved.

SPDX-License-Identifier: Apache-2.0
*/

package idemix

import (
	"strings"
	"testing"

	"github.com/IBM/idemix/bccsp/types"
	math "github.com/IBM/mathlib"
	"github.com/hyperledger-labs/fabric-token-sdk/token"
	crypto2 "github.com/hyperledger-labs/fabric-token-sdk/token/services/identity/idemix/crypto"
	"github.com/hyperledger-labs/fabric-token-sdk/token/services/identity/sig"
	kvs2 "github.com/hyperledger-labs/fabric-token-sdk/token/services/identity/storage/kvs"
	"github.com/stretchr/testify/assert"
)

func TestNewDeserializer(t *testing.T) {
	// init
	backend, err := kvs2.NewInMemory()
	assert.NoError(t, err)
	sigService := sig.NewService(sig.NewMultiplexDeserializer(), kvs2.NewIdentityDB(backend, token.TMSID{Network: "pineapple"}))
	config, err := crypto2.NewConfig("./testdata/idemix")
	assert.NoError(t, err)
	keyStore, err := crypto2.NewKeyStore(math.FP256BN_AMCL, backend)
	assert.NoError(t, err)
	cryptoProvider, err := crypto2.NewBCCSP(keyStore, math.FP256BN_AMCL, false)
	assert.NoError(t, err)

	// key manager
	keyManager, err := NewKeyManager(config, sigService, types.EidNymRhNym, cryptoProvider)
	assert.NoError(t, err)
	assert.NotNil(t, keyManager)

	// get an identity
	id, auditInfoRaw, err := keyManager.Identity(nil)
	assert.NoError(t, err)

	// instantiate a deserializer and check that it fils
	_, err = NewDeserializer(config.Ipk, -1)
	assert.Error(t, err)
	_, err = NewDeserializer(nil, math.FP256BN_AMCL)
	assert.Error(t, err)
	_, err = NewDeserializer([]byte{}, math.FP256BN_AMCL)
	assert.Error(t, err)
	_, err = NewDeserializer([]byte{0, 1, 2}, math.FP256BN_AMCL)
	assert.Error(t, err)

	// instantiate a deserializer and validate it
	d, err := NewDeserializer(config.Ipk, math.FP256BN_AMCL)
	assert.NoError(t, err)
	assert.NotNil(t, d)
	assert.Equal(t, "Idemix with IPK [dJZK5i5D2i5B8S9DsVWDFzdHSJE/jcTLk9VaJzFB4fo=]", d.String())
	_, err = d.DeserializeVerifier(nil)
	assert.Error(t, err)
	_, err = d.DeserializeVerifier([]byte{})
	assert.Error(t, err)
	_, err = d.DeserializeVerifier([]byte{0, 1, 2, 3})
	assert.Error(t, err)
	verifier1, err := d.DeserializeVerifierAgainstNymEID(id, nil)
	assert.NoError(t, err)
	verifier2, err := d.DeserializeVerifier(id)
	assert.NoError(t, err)
	signer, err := keyManager.DeserializeSigner(id)
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
	auditInfo, err := d.DeserializeAuditInfo(auditInfoRaw)
	assert.NoError(t, err)
	assert.NotNil(t, auditInfo)
	assert.Equal(t, "alice", auditInfo.EnrollmentID())
	assert.Equal(t, "150", auditInfo.RevocationHandle())
	auditInfoDeser := &AuditInfoDeserializer{}
	// check invalid input
	_, err = auditInfoDeser.DeserializeAuditInfo(nil)
	assert.Error(t, err)
	_, err = auditInfoDeser.DeserializeAuditInfo([]byte{})
	assert.Error(t, err)
	_, err = auditInfoDeser.DeserializeAuditInfo([]byte{0, 1, 2, 3})
	assert.Error(t, err)
	auditInfo2, err := auditInfoDeser.DeserializeAuditInfo(auditInfoRaw)
	assert.NoError(t, err)
	assert.Equal(t, "alice", auditInfo2.EnrollmentID())
	assert.Equal(t, "150", auditInfo2.RevocationHandle())

	// match audit info
	auditInfoMatcher, err := d.GetAuditInfoMatcher(id, auditInfoRaw)
	assert.NoError(t, err)
	assert.NotNil(t, auditInfoMatcher)
	assert.NoError(t, auditInfoMatcher.Match(id))
	assert.NoError(t, d.MatchIdentity(id, auditInfoRaw))

	// check info
	info, err := d.Info(id, []byte{})
	assert.NoError(t, err)
	assert.True(t, strings.HasPrefix(info, "Idemix: []"))
	info, err = d.Info(id, nil)
	assert.NoError(t, err)
	assert.True(t, strings.HasPrefix(info, "Idemix: []"))
	_, err = d.Info(id, []byte{0, 1, 2})
	assert.Error(t, err)
	info, err = d.Info(id, auditInfoRaw)
	assert.NoError(t, err)
	assert.True(t, strings.HasPrefix(info, "Idemix: [alice]"))
}
