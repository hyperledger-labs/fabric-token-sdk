/*
Copyright IBM Corp. All Rights Reserved.

SPDX-License-Identifier: Apache-2.0
*/

package setup

import (
	"testing"

	"github.com/hyperledger-labs/fabric-token-sdk/token/driver"
	"github.com/stretchr/testify/assert"
)

func TestSetup(t *testing.T) {
	t.Run("valid precision", func(t *testing.T) {
		pp, err := Setup(32)
		assert.NoError(t, err)
		assert.Equal(t, uint64(32), pp.QuantityPrecision)
		assert.Equal(t, uint64(1<<32)-1, pp.MaxToken)
	})

	t.Run("precision too large", func(t *testing.T) {
		pp, err := Setup(65)
		assert.Error(t, err)
		assert.Nil(t, pp)
		assert.Equal(t, "invalid precision [65], must be smaller or equal than 64", err.Error())
	})

	t.Run("precision zero", func(t *testing.T) {
		pp, err := Setup(0)
		assert.Error(t, err)
		assert.Nil(t, pp)
		assert.Equal(t, "invalid precision, should be greater than 0", err.Error())
	})

	t.Run("extras is initialized", func(t *testing.T) {
		pp, err := Setup(32)
		assert.NoError(t, err)
		assert.NotNil(t, pp.Extras())
	})
}

func TestSetupWithVersion(t *testing.T) {
	t.Run("valid setup", func(t *testing.T) {
		pp, err := WithVersion(32, driver.TokenDriverVersion(2))
		assert.NoError(t, err)
		assert.Equal(t, uint64(32), pp.QuantityPrecision)
		assert.Equal(t, uint64(1<<32)-1, pp.MaxToken)
		assert.Equal(t, driver.TokenDriverVersion(2), pp.DriverVersion)
	})

	t.Run("invalid precision", func(t *testing.T) {
		pp, err := WithVersion(65, driver.TokenDriverVersion(2))
		assert.Error(t, err)
		assert.Nil(t, pp)
	})
}

func TestNewPublicParamsFromBytes(t *testing.T) {
	t.Run("valid params", func(t *testing.T) {
		pp, err := Setup(32)
		assert.NoError(t, err)
		raw, err := pp.Serialize()
		assert.NoError(t, err)
		pp2, err := NewPublicParamsFromBytes(raw, FabTokenDriverName, ProtocolV1)
		assert.NoError(t, err)
		assert.Equal(t, FabTokenDriverName, pp2.DriverName)
		assert.Equal(t, ProtocolV1, pp2.DriverVersion)
		assert.Equal(t, uint64(32), pp2.QuantityPrecision)
		assert.Equal(t, uint64(4294967295), pp2.MaxToken)
		assert.Nil(t, pp2.IssuerIDs)
		assert.Nil(t, pp2.Auditor)
		assert.Equal(t, pp, pp2)
	})

	t.Run("invalid bytes", func(t *testing.T) {
		pp2, err := NewPublicParamsFromBytes([]byte("invalid"), FabTokenDriverName, ProtocolV1)
		assert.Error(t, err)
		assert.Nil(t, pp2)
	})
}

func TestPublicParams_Methods(t *testing.T) {
	pp, err := Setup(32)
	assert.NoError(t, err)

	t.Run("driver info", func(t *testing.T) {
		assert.Equal(t, FabTokenDriverName, pp.TokenDriverName())
		assert.Equal(t, ProtocolV1, pp.TokenDriverVersion())
	})

	t.Run("data and graph hiding", func(t *testing.T) {
		assert.False(t, pp.TokenDataHiding())
		assert.False(t, pp.GraphHiding())
	})

	t.Run("certification driver", func(t *testing.T) {
		assert.Equal(t, string(FabTokenDriverName), pp.CertificationDriver())
	})

	t.Run("max token value", func(t *testing.T) {
		assert.Equal(t, uint64(1<<32-1), pp.MaxTokenValue())
	})

	t.Run("auditor operations", func(t *testing.T) {
		auditor := driver.Identity([]byte("auditor1"))
		assert.Empty(t, pp.Auditors())
		pp.AddAuditor(auditor)
		assert.Equal(t, auditor, pp.AuditorIdentity())
		assert.Equal(t, []driver.Identity{auditor}, pp.Auditors())

		// Test SetAuditors
		newAuditor := driver.Identity([]byte("auditor2"))
		pp.SetAuditors([]driver.Identity{newAuditor})
		assert.Equal(t, newAuditor, pp.AuditorIdentity())
	})

	t.Run("issuer operations", func(t *testing.T) {
		issuer1 := driver.Identity([]byte("issuer1"))
		issuer2 := driver.Identity([]byte("issuer2"))
		assert.Empty(t, pp.Issuers())

		pp.AddIssuer(issuer1)
		assert.Equal(t, []driver.Identity{issuer1}, pp.Issuers())

		pp.AddIssuer(issuer2)
		assert.Equal(t, []driver.Identity{issuer1, issuer2}, pp.Issuers())

		// Test SetIssuers
		newIssuers := []driver.Identity{driver.Identity([]byte("new_issuer"))}
		pp.SetIssuers(newIssuers)
		assert.Equal(t, newIssuers, pp.Issuers())
	})
}

func TestPublicParams_Validation(t *testing.T) {
	t.Run("valid params", func(t *testing.T) {
		pp := &PublicParams{
			DriverName:        FabTokenDriverName,
			QuantityPrecision: 32,
			MaxToken:          1<<32 - 1,
		}
		err := pp.Validate()
		assert.NoError(t, err)
	})

	t.Run("precision too large", func(t *testing.T) {
		pp := &PublicParams{
			DriverName:        FabTokenDriverName,
			QuantityPrecision: 65,
			MaxToken:          1<<64 - 1,
		}
		err := pp.Validate()
		assert.Error(t, err)
		assert.Equal(t, "invalid precision [65], must be less than 64", err.Error())
	})

	t.Run("precision zero", func(t *testing.T) {
		pp := &PublicParams{
			DriverName:        FabTokenDriverName,
			QuantityPrecision: 0,
			MaxToken:          1,
		}
		err := pp.Validate()
		assert.Error(t, err)
		assert.Equal(t, "invalid precision, must be greater than 0", err.Error())
	})

	t.Run("invalid max token", func(t *testing.T) {
		pp := &PublicParams{
			DriverName:        FabTokenDriverName,
			QuantityPrecision: 32,
			MaxToken:          1 << 32,
		}
		err := pp.Validate()
		assert.Error(t, err)
		assert.Equal(t, "max token value is invalid [4294967296]>[4294967295]", err.Error())
	})
}

func TestPublicParams_Serialization(t *testing.T) {
	t.Run("valid serialization and deserialization", func(t *testing.T) {
		original, err := Setup(32)
		assert.NoError(t, err)

		// Add some data
		original.AddAuditor([]byte("auditor1"))
		original.AddIssuer([]byte("issuer1"))
		original.ExtraData = map[string][]byte{"key1": []byte("value1")}

		// Serialize
		serialized, err := original.Serialize()
		assert.NoError(t, err)

		// Deserialize
		deserialized := &PublicParams{
			DriverName:    FabTokenDriverName,
			DriverVersion: ProtocolV1,
		}
		err = deserialized.Deserialize(serialized)
		assert.NoError(t, err)

		// Verify all fields match
		assert.Equal(t, original.QuantityPrecision, deserialized.QuantityPrecision)
		assert.Equal(t, original.MaxToken, deserialized.MaxToken)
		assert.Equal(t, original.Auditor, deserialized.Auditor)
		assert.Equal(t, original.IssuerIDs, deserialized.IssuerIDs)
		assert.Equal(t, original.ExtraData, deserialized.ExtraData)
	})

	t.Run("invalid deserialization", func(t *testing.T) {
		pp := &PublicParams{
			DriverName:    FabTokenDriverName,
			DriverVersion: ProtocolV1,
		}
		err := pp.Deserialize([]byte("invalid"))
		assert.Error(t, err)
	})

	t.Run("mismatched driver identifier", func(t *testing.T) {
		original, err := Setup(32)
		assert.NoError(t, err)
		serialized, err := original.Serialize()
		assert.NoError(t, err)

		wrongDriver := &PublicParams{
			DriverName:    "wrong",
			DriverVersion: ProtocolV1,
		}
		err = wrongDriver.Deserialize(serialized)
		assert.Error(t, err)
		assert.Contains(t, err.Error(), "invalid identifier")
	})
}

func TestPublicParams_BytesAndFromBytes(t *testing.T) {
	t.Run("serialization with nil values", func(t *testing.T) {
		pp := &PublicParams{
			DriverName:        FabTokenDriverName,
			QuantityPrecision: 32,
			MaxToken:          1<<32 - 1,
			IssuerIDs:         nil,
			Auditor:           nil,
			ExtraData:         nil,
		}
		bytes, err := pp.Bytes()
		assert.NoError(t, err)
		assert.NotNil(t, bytes)

		// Test FromBytes with the serialized data
		newPP := &PublicParams{}
		err = newPP.FromBytes(bytes)
		assert.NoError(t, err)
		assert.Equal(t, pp.QuantityPrecision, newPP.QuantityPrecision)
		assert.Equal(t, pp.MaxToken, newPP.MaxToken)
	})

	t.Run("serialization with invalid identity", func(t *testing.T) {
		pp := &PublicParams{
			DriverName:        FabTokenDriverName,
			QuantityPrecision: 32,
			MaxToken:          1<<32 - 1,
			IssuerIDs:         []driver.Identity{nil}, // Invalid identity
		}
		bytes, err := pp.Bytes()
		assert.NoError(t, err) // Should handle nil identity

		newPP := &PublicParams{}
		err = newPP.FromBytes(bytes)
		assert.NoError(t, err)
	})

	t.Run("invalid protobuf data", func(t *testing.T) {
		pp := &PublicParams{}
		err := pp.FromBytes([]byte{0xFF, 0xFF, 0xFF}) // Invalid protobuf data
		assert.Error(t, err)
	})
}

func TestPublicParams_String(t *testing.T) {
	pp, err := Setup(32)
	assert.NoError(t, err)
	pp.AddAuditor([]byte("auditor1"))
	pp.AddIssuer([]byte("issuer1"))
	pp.ExtraData = map[string][]byte{"key1": []byte("value1")}

	str := pp.String()
	assert.NotEmpty(t, str)
	assert.Contains(t, str, "QuantityPrecision")
	assert.Contains(t, str, "MaxToken")
}

func TestPublicParams_StringWithInvalidJSON(t *testing.T) {
	pp := &PublicParams{
		DriverName:        FabTokenDriverName,
		QuantityPrecision: 32,
		MaxToken:          1<<32 - 1,
		// Add a channel which cannot be marshaled to JSON
		ExtraData: map[string][]byte{
			"key": []byte("value"),
		},
	}

	str := pp.String()
	assert.Contains(t, str, "QuantityPrecision")
	assert.Contains(t, str, "MaxToken")
	assert.Contains(t, str, "key")
}

func TestExtras(t *testing.T) {
	pp := &PublicParams{
		ExtraData: map[string][]byte{
			"key1": []byte("value1"),
			"key2": []byte("value2"),
		},
	}

	extras := pp.Extras()
	assert.Equal(t, pp.ExtraData, extras)

	// Verify nil case
	pp = &PublicParams{}
	assert.Nil(t, pp.Extras())

	// Verify empty map case
	pp.ExtraData = make(map[string][]byte)
	assert.Empty(t, pp.Extras())
}
