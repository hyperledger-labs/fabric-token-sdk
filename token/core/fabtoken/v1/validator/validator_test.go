/*
Copyright IBM Corp. All Rights Reserved.

SPDX-License-Identifier: Apache-2.0
*/

package validator_test

import (
	"context"
	"crypto"
	"encoding/json"
	"testing"
	"time"

	"github.com/hyperledger-labs/fabric-token-sdk/token/core/fabtoken/v1/actions"
	"github.com/hyperledger-labs/fabric-token-sdk/token/core/fabtoken/v1/setup"
	"github.com/hyperledger-labs/fabric-token-sdk/token/core/fabtoken/v1/validator"
	validator2 "github.com/hyperledger-labs/fabric-token-sdk/token/core/zkatdlog/nogh/v1/validator"
	"github.com/hyperledger-labs/fabric-token-sdk/token/driver"
	"github.com/hyperledger-labs/fabric-token-sdk/token/driver/mock"
	"github.com/hyperledger-labs/fabric-token-sdk/token/services/identity"
	"github.com/hyperledger-labs/fabric-token-sdk/token/services/interop/encoding"
	"github.com/hyperledger-labs/fabric-token-sdk/token/services/interop/htlc"
	"github.com/hyperledger-labs/fabric-token-sdk/token/services/logging"
	"github.com/hyperledger-labs/fabric-token-sdk/token/token"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestActionDeserializer_DeserializeActions(t *testing.T) {
	ad := &validator.ActionDeserializer{}

	t.Run("Empty", func(t *testing.T) {
		tr := &driver.TokenRequest{}
		ia, ta, err := ad.DeserializeActions(tr)
		require.NoError(t, err)
		assert.Empty(t, ia)
		assert.Empty(t, ta)
	})

	t.Run("WithIssue", func(t *testing.T) {
		ia1 := &actions.IssueAction{Issuer: []byte("issuer1")}
		ia1Bytes, err := ia1.Serialize()
		require.NoError(t, err)

		tr := &driver.TokenRequest{
			Issues: [][]byte{ia1Bytes},
		}
		ia, ta, err := ad.DeserializeActions(tr)
		require.NoError(t, err)
		assert.Len(t, ia, 1)
		assert.Empty(t, ta)
		assert.Equal(t, ia1.Issuer, ia[0].Issuer)
	})

	t.Run("WithTransfer", func(t *testing.T) {
		ta1 := &actions.TransferAction{Issuer: []byte("issuer1")}
		ta1Bytes, err := ta1.Serialize()
		require.NoError(t, err)

		tr := &driver.TokenRequest{
			Transfers: [][]byte{ta1Bytes},
		}
		ia, ta, err := ad.DeserializeActions(tr)
		require.NoError(t, err)
		assert.Empty(t, ia)
		assert.Len(t, ta, 1)
		assert.Equal(t, ta1.Issuer, ta[0].Issuer)
	})

	t.Run("IssueDeserializeError", func(t *testing.T) {
		tr := &driver.TokenRequest{
			Issues: [][]byte{[]byte("invalid")},
		}
		_, _, err := ad.DeserializeActions(tr)
		require.Error(t, err)
	})

	t.Run("TransferDeserializeError", func(t *testing.T) {
		tr := &driver.TokenRequest{
			Transfers: [][]byte{[]byte("invalid")},
		}
		_, _, err := ad.DeserializeActions(tr)
		require.Error(t, err)
	})
}

func TestNewValidator(t *testing.T) {
	logger := logging.MustGetLogger("test")
	pp := &setup.PublicParams{}
	deserializer := &mock.Deserializer{}

	v := validator.NewValidator(logger, pp, deserializer, nil, nil, nil)
	assert.NotNil(t, v)
}

func TestIssueValidate(t *testing.T) {
	ctx := context.Background()
	pp := &setup.PublicParams{
		QuantityPrecision: 64,
		IssuerIDs:         []driver.Identity{[]byte("issuer1")},
	}
	deserializer := &mock.Deserializer{}
	sigProvider := &mock.SignatureProvider{}

	t.Run("EmptyOutputs", func(t *testing.T) {
		ia := &actions.IssueAction{
			Issuer: []byte("issuer1"),
		}
		c := &validator.Context{
			PP:          pp,
			IssueAction: ia,
		}
		err := validator.IssueValidate(ctx, c)
		require.Error(t, err)
		assert.Contains(t, err.Error(), "no outputs in issue action")
	})

	t.Run("ZeroQuantity", func(t *testing.T) {
		ia := &actions.IssueAction{
			Issuer: []byte("issuer1"),
			Outputs: []*actions.Output{
				{
					Quantity: "0",
					Type:     "ABC",
					Owner:    []byte("owner1"),
				},
			},
		}
		c := &validator.Context{
			PP:          pp,
			IssueAction: ia,
		}
		err := validator.IssueValidate(ctx, c)
		require.Error(t, err)
		assert.Contains(t, err.Error(), "quantity is zero")
	})

	t.Run("UnauthorizedIssuer", func(t *testing.T) {
		ia := &actions.IssueAction{
			Issuer: []byte("unauthorized"),
			Outputs: []*actions.Output{
				{
					Quantity: "100",
					Type:     "ABC",
					Owner:    []byte("owner1"),
				},
			},
		}
		c := &validator.Context{
			PP:          pp,
			IssueAction: ia,
		}
		err := validator.IssueValidate(ctx, c)
		require.Error(t, err)
		require.ErrorIs(t, err, validator2.ErrIssuerNotAuthorized)
		assert.Contains(t, err.Error(), validator2.ErrIssuerNotAuthorized.Error())
	})

	t.Run("VerifierError", func(t *testing.T) {
		ia := &actions.IssueAction{
			Issuer: []byte("issuer1"),
			Outputs: []*actions.Output{
				{
					Quantity: "100",
					Type:     "ABC",
					Owner:    []byte("owner1"),
				},
			},
		}
		deserializer.GetIssuerVerifierReturns(nil, assert.AnError)
		c := &validator.Context{
			PP:           pp,
			IssueAction:  ia,
			Deserializer: deserializer,
		}
		err := validator.IssueValidate(ctx, c)
		require.Error(t, err)
		assert.Contains(t, err.Error(), "failed getting verifier for issuer identity")
	})

	t.Run("SignatureVerificationError", func(t *testing.T) {
		ia := &actions.IssueAction{
			Issuer: []byte("issuer1"),
			Outputs: []*actions.Output{
				{
					Quantity: "100",
					Type:     "ABC",
					Owner:    []byte("owner1"),
				},
			},
		}
		deserializer.GetIssuerVerifierReturns(&mock.Verifier{}, nil)
		sigProvider.HasBeenSignedByReturns(nil, assert.AnError)
		c := &validator.Context{
			PP:                pp,
			IssueAction:       ia,
			Deserializer:      deserializer,
			SignatureProvider: sigProvider,
		}
		err := validator.IssueValidate(ctx, c)
		require.Error(t, err)
		assert.Contains(t, err.Error(), "failed verifying signature")
	})

	t.Run("Success", func(t *testing.T) {
		ia := &actions.IssueAction{
			Issuer: []byte("issuer1"),
			Outputs: []*actions.Output{
				{
					Quantity: "100",
					Type:     "ABC",
					Owner:    []byte("owner1"),
				},
			},
		}
		deserializer.GetIssuerVerifierReturns(&mock.Verifier{}, nil)
		sigProvider.HasBeenSignedByReturns([]byte("signature"), nil)
		c := &validator.Context{
			PP:                pp,
			IssueAction:       ia,
			Deserializer:      deserializer,
			SignatureProvider: sigProvider,
		}
		err := validator.IssueValidate(ctx, c)
		require.NoError(t, err)
	})

	t.Run("SuccessNoIssuersInPP", func(t *testing.T) {
		ppNoIssuers := &setup.PublicParams{
			QuantityPrecision: 64,
		}
		ia := &actions.IssueAction{
			Issuer: []byte("any-issuer"),
			Outputs: []*actions.Output{
				{
					Quantity: "100",
					Type:     "ABC",
					Owner:    []byte("owner1"),
				},
			},
		}
		deserializer.GetIssuerVerifierReturns(&mock.Verifier{}, nil)
		sigProvider.HasBeenSignedByReturns([]byte("signature"), nil)
		c := &validator.Context{
			PP:                ppNoIssuers,
			IssueAction:       ia,
			Deserializer:      deserializer,
			SignatureProvider: sigProvider,
		}
		err := validator.IssueValidate(ctx, c)
		require.NoError(t, err)
	})
}

func TestTransferActionValidate(t *testing.T) {
	ctx := context.Background()
	ta := &actions.TransferAction{
		Inputs: []*actions.TransferActionInput{
			{
				ID: &token.ID{TxId: "tx1", Index: 0},
				Input: &actions.Output{
					Type:     "ABC",
					Quantity: "100",
					Owner:    []byte("owner1"),
				},
			},
		},
		Outputs: []*actions.Output{
			{
				Quantity: "100",
				Type:     "ABC",
				Owner:    []byte("owner1"),
			},
		},
	}
	c := &validator.Context{
		TransferAction: ta,
	}
	err := validator.TransferActionValidate(ctx, c)
	require.NoError(t, err)
}

func TestTransferSignatureValidate(t *testing.T) {
	ctx := context.Background()
	logger := logging.MustGetLogger("test")
	pp := &setup.PublicParams{
		IssuerIDs: []driver.Identity{[]byte("issuer1")},
	}

	t.Run("NoInputs", func(t *testing.T) {
		ta := &actions.TransferAction{}
		c := &validator.Context{
			TransferAction: ta,
		}
		err := validator.TransferSignatureValidate(ctx, c)
		require.Error(t, err)
		assert.Contains(t, err.Error(), "expected at least 1")
	})

	t.Run("OwnerVerifierError", func(t *testing.T) {
		deserializer := &mock.Deserializer{}
		ta := &actions.TransferAction{
			Inputs: []*actions.TransferActionInput{
				{
					Input: &actions.Output{
						Owner: []byte("owner1"),
					},
				},
			},
		}
		deserializer.GetOwnerVerifierReturns(nil, assert.AnError)
		c := &validator.Context{
			TransferAction: ta,
			Deserializer:   deserializer,
			Logger:         logger,
		}
		err := validator.TransferSignatureValidate(ctx, c)
		require.Error(t, err)
		assert.Contains(t, err.Error(), "failed deserializing owner")
	})

	t.Run("OwnerSignatureError", func(t *testing.T) {
		deserializer := &mock.Deserializer{}
		sigProvider := &mock.SignatureProvider{}
		ta := &actions.TransferAction{
			Inputs: []*actions.TransferActionInput{
				{
					Input: &actions.Output{
						Owner: []byte("owner1"),
					},
				},
			},
		}
		deserializer.GetOwnerVerifierReturns(&mock.Verifier{}, nil)
		sigProvider.HasBeenSignedByReturns(nil, assert.AnError)
		c := &validator.Context{
			TransferAction:    ta,
			Deserializer:      deserializer,
			SignatureProvider: sigProvider,
			Logger:            logger,
		}
		err := validator.TransferSignatureValidate(ctx, c)
		require.Error(t, err)
		assert.Contains(t, err.Error(), "failed signature verification")
	})

	t.Run("SuccessTransfer", func(t *testing.T) {
		deserializer := &mock.Deserializer{}
		sigProvider := &mock.SignatureProvider{}
		ta := &actions.TransferAction{
			Inputs: []*actions.TransferActionInput{
				{
					Input: &actions.Output{
						Owner: []byte("owner1"),
					},
				},
			},
			Outputs: []*actions.Output{
				{
					Owner: []byte("owner2"),
				},
			},
		}
		deserializer.GetOwnerVerifierReturns(&mock.Verifier{}, nil)
		sigProvider.HasBeenSignedByReturns([]byte("sig1"), nil)
		c := &validator.Context{
			TransferAction:    ta,
			Deserializer:      deserializer,
			SignatureProvider: sigProvider,
			Logger:            logger,
			PP:                pp,
		}
		err := validator.TransferSignatureValidate(ctx, c)
		require.NoError(t, err)
		assert.Len(t, c.Signatures, 1)
		assert.Len(t, c.InputTokens, 1)
	})

	t.Run("RedeemWithoutIssuer", func(t *testing.T) {
		deserializer := &mock.Deserializer{}
		sigProvider := &mock.SignatureProvider{}
		ta := &actions.TransferAction{
			Inputs: []*actions.TransferActionInput{
				{
					Input: &actions.Output{
						Owner: []byte("owner1"),
					},
				},
			},
			Outputs: []*actions.Output{
				{
					Owner: nil, // redeem
				},
			},
		}
		deserializer.GetOwnerVerifierReturns(&mock.Verifier{}, nil)
		sigProvider.HasBeenSignedByReturns([]byte("sig1"), nil)
		c := &validator.Context{
			TransferAction:    ta,
			Deserializer:      deserializer,
			SignatureProvider: sigProvider,
			Logger:            logger,
			PP:                pp,
		}
		err := validator.TransferSignatureValidate(ctx, c)
		require.Error(t, err)
		assert.Contains(t, err.Error(), "must have at least one issuer")
	})

	t.Run("RedeemWithIssuer", func(t *testing.T) {
		deserializer := &mock.Deserializer{}
		sigProvider := &mock.SignatureProvider{}
		ta := &actions.TransferAction{
			Issuer: []byte("issuer1"),
			Inputs: []*actions.TransferActionInput{
				{
					Input: &actions.Output{
						Owner: []byte("owner1"),
					},
				},
			},
			Outputs: []*actions.Output{
				{
					Owner: nil, // redeem
				},
			},
		}
		deserializer.GetOwnerVerifierReturns(&mock.Verifier{}, nil)
		deserializer.GetIssuerVerifierReturns(&mock.Verifier{}, nil)
		sigProvider.HasBeenSignedByReturns([]byte("sig"), nil)
		c := &validator.Context{
			TransferAction:    ta,
			Deserializer:      deserializer,
			SignatureProvider: sigProvider,
			Logger:            logger,
			PP:                pp,
		}
		err := validator.TransferSignatureValidate(ctx, c)
		require.NoError(t, err)
		assert.Len(t, c.Signatures, 2) // one for owner, one for issuer
	})

	t.Run("RedeemIssuerVerifierError", func(t *testing.T) {
		deserializer := &mock.Deserializer{}
		sigProvider := &mock.SignatureProvider{}
		ta := &actions.TransferAction{
			Issuer: []byte("issuer1"),
			Inputs: []*actions.TransferActionInput{
				{
					Input: &actions.Output{
						Owner: []byte("owner1"),
					},
				},
			},
			Outputs: []*actions.Output{
				{
					Owner: nil, // redeem
				},
			},
		}
		deserializer.GetOwnerVerifierReturns(&mock.Verifier{}, nil)
		deserializer.GetIssuerVerifierReturns(nil, assert.AnError)
		sigProvider.HasBeenSignedByReturns([]byte("sig"), nil)
		c := &validator.Context{
			TransferAction:    ta,
			Deserializer:      deserializer,
			SignatureProvider: sigProvider,
			Logger:            logger,
			PP:                pp,
		}
		err := validator.TransferSignatureValidate(ctx, c)
		require.Error(t, err)
		assert.Contains(t, err.Error(), "failed deserializing issuer")
	})

	t.Run("RedeemIssuerSignatureError", func(t *testing.T) {
		deserializer := &mock.Deserializer{}
		sigProvider := &mock.SignatureProvider{}
		ta := &actions.TransferAction{
			Issuer: []byte("issuer1"),
			Inputs: []*actions.TransferActionInput{
				{
					Input: &actions.Output{
						Owner: []byte("owner1"),
					},
				},
			},
			Outputs: []*actions.Output{
				{
					Owner: nil, // redeem
				},
			},
		}
		deserializer.GetOwnerVerifierReturns(&mock.Verifier{}, nil)
		deserializer.GetIssuerVerifierReturns(&mock.Verifier{}, nil)
		// first call (for owner) returns success
		sigProvider.HasBeenSignedByReturnsOnCall(0, []byte("sig-owner"), nil)
		// second call (for issuer) returns error
		sigProvider.HasBeenSignedByReturnsOnCall(1, nil, assert.AnError)

		c := &validator.Context{
			TransferAction:    ta,
			Deserializer:      deserializer,
			SignatureProvider: sigProvider,
			Logger:            logger,
			PP:                pp,
		}
		err := validator.TransferSignatureValidate(ctx, c)
		require.Error(t, err)
		assert.Contains(t, err.Error(), "failed signature verification")
	})
}

func TestTransferBalanceValidate(t *testing.T) {
	ctx := context.Background()
	pp := &setup.PublicParams{
		QuantityPrecision: 64,
	}

	t.Run("NoOutputs", func(t *testing.T) {
		ta := &actions.TransferAction{}
		c := &validator.Context{
			TransferAction: ta,
		}
		err := validator.TransferBalanceValidate(ctx, c)
		require.Error(t, err)
		assert.Contains(t, err.Error(), "there is no output")
	})

	t.Run("NoInputs", func(t *testing.T) {
		ta := &actions.TransferAction{
			Outputs: []*actions.Output{{Quantity: "100"}},
		}
		c := &validator.Context{
			TransferAction: ta,
			InputTokens:    nil,
		}
		err := validator.TransferBalanceValidate(ctx, c)
		require.Error(t, err)
		assert.Contains(t, err.Error(), "there is no input")
	})

	t.Run("NilFirstInput", func(t *testing.T) {
		ta := &actions.TransferAction{
			Outputs: []*actions.Output{{Quantity: "100"}},
		}
		c := &validator.Context{
			TransferAction: ta,
			InputTokens:    []*actions.Output{nil},
		}
		err := validator.TransferBalanceValidate(ctx, c)
		require.Error(t, err)
		assert.Contains(t, err.Error(), "first input is nil")
	})

	t.Run("NilInput", func(t *testing.T) {
		ta := &actions.TransferAction{
			Outputs: []*actions.Output{{Quantity: "100"}},
		}
		c := &validator.Context{
			TransferAction: ta,
			PP:             pp,
			InputTokens:    []*actions.Output{{Quantity: "100"}, nil},
		}
		err := validator.TransferBalanceValidate(ctx, c)
		require.Error(t, err)
		assert.Contains(t, err.Error(), "input 1 is nil")
	})

	t.Run("ParseQuantityInputError", func(t *testing.T) {
		ta := &actions.TransferAction{
			Outputs: []*actions.Output{{Quantity: "100"}},
		}
		c := &validator.Context{
			TransferAction: ta,
			PP:             pp,
			InputTokens:    []*actions.Output{{Quantity: "invalid"}},
		}
		err := validator.TransferBalanceValidate(ctx, c)
		require.Error(t, err)
		assert.Contains(t, err.Error(), "failed parsing quantity")
	})

	t.Run("MismatchedInputType", func(t *testing.T) {
		ta := &actions.TransferAction{
			Outputs: []*actions.Output{{Quantity: "100", Type: "ABC"}},
		}
		c := &validator.Context{
			TransferAction: ta,
			PP:             pp,
			InputTokens:    []*actions.Output{{Quantity: "50", Type: "ABC"}, {Quantity: "50", Type: "XYZ"}},
		}
		err := validator.TransferBalanceValidate(ctx, c)
		require.Error(t, err)
		assert.Contains(t, err.Error(), "does not match type")
	})

	t.Run("ParseQuantityOutputError", func(t *testing.T) {
		ta := &actions.TransferAction{
			Outputs: []*actions.Output{{Quantity: "invalid"}},
		}
		c := &validator.Context{
			TransferAction: ta,
			PP:             pp,
			InputTokens:    []*actions.Output{{Quantity: "100"}},
		}
		err := validator.TransferBalanceValidate(ctx, c)
		require.Error(t, err)
		assert.Contains(t, err.Error(), "failed parsing quantity")
	})

	t.Run("MismatchedOutputType", func(t *testing.T) {
		ta := &actions.TransferAction{
			Outputs: []*actions.Output{{Quantity: "100", Type: "XYZ"}},
		}
		c := &validator.Context{
			TransferAction: ta,
			PP:             pp,
			InputTokens:    []*actions.Output{{Quantity: "100", Type: "ABC"}},
		}
		err := validator.TransferBalanceValidate(ctx, c)
		require.Error(t, err)
		assert.Contains(t, err.Error(), "does not match type")
	})

	t.Run("Unbalanced", func(t *testing.T) {
		ta := &actions.TransferAction{
			Outputs: []*actions.Output{{Quantity: "101", Type: "ABC"}},
		}
		c := &validator.Context{
			TransferAction: ta,
			PP:             pp,
			InputTokens:    []*actions.Output{{Quantity: "100", Type: "ABC"}},
		}
		err := validator.TransferBalanceValidate(ctx, c)
		require.Error(t, err)
		assert.Contains(t, err.Error(), "does not match output sum")
	})

	t.Run("Success", func(t *testing.T) {
		ta := &actions.TransferAction{
			Outputs: []*actions.Output{{Quantity: "100", Type: "ABC"}, {Quantity: "50", Type: "ABC"}},
		}
		c := &validator.Context{
			TransferAction: ta,
			PP:             pp,
			InputTokens:    []*actions.Output{{Quantity: "150", Type: "ABC"}},
		}
		err := validator.TransferBalanceValidate(ctx, c)
		require.NoError(t, err)
	})
}

func TestTransferHTLCValidate(t *testing.T) {
	ctx := context.Background()

	t.Run("NoHTLC", func(t *testing.T) {
		owner1, _ := identity.WrapWithType("x509", []byte("owner1"))
		owner2, _ := identity.WrapWithType("x509", []byte("owner2"))
		ta := &actions.TransferAction{
			Outputs: []*actions.Output{{Owner: owner1}},
		}
		c := &validator.Context{
			TransferAction: ta,
			InputTokens:    []*actions.Output{{Owner: owner2}},
		}
		err := validator.TransferHTLCValidate(ctx, c)
		require.NoError(t, err)
	})

	t.Run("InputIsHTLC_Reclaim_Success", func(t *testing.T) {
		sender, _ := identity.WrapWithType("x509", []byte("sender"))
		recipient, _ := identity.WrapWithType("x509", []byte("recipient"))
		preimage := []byte("preimage")
		hash := crypto.SHA256.New()
		hash.Write(preimage)
		img := hash.Sum(nil)

		script := &htlc.Script{
			Sender:    sender,
			Recipient: recipient,
			Deadline:  time.Now().Add(-1 * time.Hour), // expired
			HashInfo: htlc.HashInfo{
				Hash:         img,
				HashFunc:     crypto.SHA256,
				HashEncoding: encoding.Base64,
			},
		}
		scriptBytes, err := json.Marshal(script)
		require.NoError(t, err)
		htlcOwner, err := identity.WrapWithType(htlc.ScriptType, scriptBytes)
		require.NoError(t, err)

		ta := &actions.TransferAction{
			Outputs: []*actions.Output{
				{Owner: sender, Type: "ABC", Quantity: "100"},
			},
		}
		c := &validator.Context{
			TransferAction:  ta,
			InputTokens:     []*actions.Output{{Owner: htlcOwner, Type: "ABC", Quantity: "100"}},
			Signatures:      [][]byte{[]byte("sig")},
			MetadataCounter: make(map[string]int),
		}
		err = validator.TransferHTLCValidate(ctx, c)
		require.NoError(t, err)
	})

	t.Run("InputIsHTLC_Claim_Success", func(t *testing.T) {
		sender, _ := identity.WrapWithType("x509", []byte("sender"))
		recipient, _ := identity.WrapWithType("x509", []byte("recipient"))
		preimage := []byte("preimage")
		hash := crypto.SHA256.New()
		hash.Write(preimage)
		img := hash.Sum(nil)
		// encoded image for claim key
		imgEncoded := []byte(encoding.Base64.New().EncodeToString(img))

		script := &htlc.Script{
			Sender:    sender,
			Recipient: recipient,
			Deadline:  time.Now().Add(1 * time.Hour), // not expired
			HashInfo: htlc.HashInfo{
				Hash:         imgEncoded,
				HashFunc:     crypto.SHA256,
				HashEncoding: encoding.Base64,
			},
		}
		scriptBytes, err := json.Marshal(script)
		require.NoError(t, err)
		htlcOwner, err := identity.WrapWithType(htlc.ScriptType, scriptBytes)
		require.NoError(t, err)

		claimSig := &htlc.ClaimSignature{
			Preimage:           preimage,
			RecipientSignature: []byte("rec-sig"),
		}
		claimSigBytes, err := json.Marshal(claimSig)
		require.NoError(t, err)

		ta := &actions.TransferAction{
			Outputs: []*actions.Output{
				{Owner: recipient, Type: "ABC", Quantity: "100"},
			},
			Metadata: map[string][]byte{
				htlc.ClaimKey(imgEncoded): preimage,
			},
		}
		c := &validator.Context{
			TransferAction:  ta,
			InputTokens:     []*actions.Output{{Owner: htlcOwner, Type: "ABC", Quantity: "100"}},
			Signatures:      [][]byte{claimSigBytes},
			MetadataCounter: make(map[string]int),
		}
		err = validator.TransferHTLCValidate(ctx, c)
		require.NoError(t, err)
		assert.Equal(t, 1, c.MetadataCounter[htlc.ClaimKey(imgEncoded)])
	})

	t.Run("OutputIsHTLC_Success", func(t *testing.T) {
		sender, _ := identity.WrapWithType("x509", []byte("sender"))
		recipient, _ := identity.WrapWithType("x509", []byte("recipient"))
		preimage := []byte("preimage")
		hash := crypto.SHA256.New()
		hash.Write(preimage)
		img := hash.Sum(nil)

		script := &htlc.Script{
			Sender:    sender,
			Recipient: recipient,
			Deadline:  time.Now().Add(1 * time.Hour), // not expired
			HashInfo: htlc.HashInfo{
				Hash:         img,
				HashFunc:     crypto.SHA256,
				HashEncoding: encoding.Base64,
			},
		}
		scriptBytes, err := json.Marshal(script)
		require.NoError(t, err)
		htlcOwner, err := identity.WrapWithType(htlc.ScriptType, scriptBytes)
		require.NoError(t, err)

		ta := &actions.TransferAction{
			Outputs: []*actions.Output{
				{Owner: htlcOwner, Type: "ABC", Quantity: "100"},
			},
			Metadata: map[string][]byte{
				htlc.LockKey(img): htlc.LockValue(img),
			},
		}
		c := &validator.Context{
			TransferAction:  ta,
			InputTokens:     []*actions.Output{{Owner: sender, Type: "ABC", Quantity: "100"}},
			MetadataCounter: make(map[string]int),
		}
		err = validator.TransferHTLCValidate(ctx, c)
		require.NoError(t, err)
		assert.Equal(t, 1, c.MetadataCounter[htlc.LockKey(img)])
	})

	t.Run("InputIsHTLC_InvalidOwner", func(t *testing.T) {
		htlcOwner := []byte("invalid-typed-identity")
		ta := &actions.TransferAction{
			Outputs: []*actions.Output{{Owner: []byte("rec")}},
		}
		c := &validator.Context{
			TransferAction: ta,
			InputTokens:    []*actions.Output{{Owner: htlcOwner}},
		}
		err := validator.TransferHTLCValidate(ctx, c)
		require.Error(t, err)
		assert.Contains(t, err.Error(), "failed to unmarshal owner")
	})

	t.Run("InputIsHTLC_TwoOutputs_Error", func(t *testing.T) {
		sender, _ := identity.WrapWithType("x509", []byte("sender"))
		recipient, _ := identity.WrapWithType("x509", []byte("recipient"))
		script := &htlc.Script{
			Sender:    sender,
			Recipient: recipient,
			Deadline:  time.Now().Add(1 * time.Hour),
			HashInfo: htlc.HashInfo{
				HashFunc:     crypto.SHA256,
				HashEncoding: encoding.Base64,
			},
		}
		scriptBytes, _ := json.Marshal(script)
		htlcOwner, _ := identity.WrapWithType(htlc.ScriptType, scriptBytes)

		ta := &actions.TransferAction{
			Outputs: []*actions.Output{
				{Owner: recipient, Type: "ABC", Quantity: "100"},
				{Owner: sender, Type: "ABC", Quantity: "50"},
			},
		}
		c := &validator.Context{
			TransferAction: ta,
			InputTokens:    []*actions.Output{{Owner: htlcOwner, Type: "ABC", Quantity: "150"}},
		}
		err := validator.TransferHTLCValidate(ctx, c)
		require.Error(t, err)
		assert.Contains(t, err.Error(), "an htlc script only transfers the ownership of a token")
	})

	t.Run("InputIsHTLC_TypeMismatch_Error", func(t *testing.T) {
		sender, _ := identity.WrapWithType("x509", []byte("sender"))
		recipient, _ := identity.WrapWithType("x509", []byte("recipient"))
		script := &htlc.Script{
			Sender:    sender,
			Recipient: recipient,
			Deadline:  time.Now().Add(1 * time.Hour),
			HashInfo: htlc.HashInfo{
				HashFunc:     crypto.SHA256,
				HashEncoding: encoding.Base64,
			},
		}
		scriptBytes, _ := json.Marshal(script)
		htlcOwner, _ := identity.WrapWithType(htlc.ScriptType, scriptBytes)

		ta := &actions.TransferAction{
			Outputs: []*actions.Output{
				{Owner: recipient, Type: "XYZ", Quantity: "100"},
			},
		}
		c := &validator.Context{
			TransferAction: ta,
			InputTokens:    []*actions.Output{{Owner: htlcOwner, Type: "ABC", Quantity: "100"}},
		}
		err := validator.TransferHTLCValidate(ctx, c)
		require.Error(t, err)
		assert.Contains(t, err.Error(), "type of input does not match type of output")
	})

	t.Run("InputIsHTLC_QuantityMismatch_Error", func(t *testing.T) {
		sender, _ := identity.WrapWithType("x509", []byte("sender"))
		recipient, _ := identity.WrapWithType("x509", []byte("recipient"))
		script := &htlc.Script{
			Sender:    sender,
			Recipient: recipient,
			Deadline:  time.Now().Add(1 * time.Hour),
			HashInfo: htlc.HashInfo{
				HashFunc:     crypto.SHA256,
				HashEncoding: encoding.Base64,
			},
		}
		scriptBytes, _ := json.Marshal(script)
		htlcOwner, _ := identity.WrapWithType(htlc.ScriptType, scriptBytes)

		ta := &actions.TransferAction{
			Outputs: []*actions.Output{
				{Owner: recipient, Type: "ABC", Quantity: "101"},
			},
		}
		c := &validator.Context{
			TransferAction: ta,
			InputTokens:    []*actions.Output{{Owner: htlcOwner, Type: "ABC", Quantity: "100"}},
		}
		err := validator.TransferHTLCValidate(ctx, c)
		require.Error(t, err)
		assert.Contains(t, err.Error(), "quantity of input does not match quantity of output")
	})

	t.Run("InputIsHTLC_Redeem_Error", func(t *testing.T) {
		sender, _ := identity.WrapWithType("x509", []byte("sender"))
		recipient, _ := identity.WrapWithType("x509", []byte("recipient"))
		script := &htlc.Script{
			Sender:    sender,
			Recipient: recipient,
			Deadline:  time.Now().Add(1 * time.Hour),
			HashInfo: htlc.HashInfo{
				HashFunc:     crypto.SHA256,
				HashEncoding: encoding.Base64,
			},
		}
		scriptBytes, _ := json.Marshal(script)
		htlcOwner, _ := identity.WrapWithType(htlc.ScriptType, scriptBytes)

		ta := &actions.TransferAction{
			Outputs: []*actions.Output{
				{Owner: nil, Type: "ABC", Quantity: "100"},
			},
		}
		c := &validator.Context{
			TransferAction: ta,
			InputTokens:    []*actions.Output{{Owner: htlcOwner, Type: "ABC", Quantity: "100"}},
		}
		err := validator.TransferHTLCValidate(ctx, c)
		require.Error(t, err)
		assert.Contains(t, err.Error(), "should not be a redeem")
	})

	t.Run("OutputIsHTLC_Expired_Error", func(t *testing.T) {
		sender, _ := identity.WrapWithType("x509", []byte("sender"))
		recipient, _ := identity.WrapWithType("x509", []byte("recipient"))
		script := &htlc.Script{
			Sender:    sender,
			Recipient: recipient,
			Deadline:  time.Now().Add(-1 * time.Hour), // expired
			HashInfo: htlc.HashInfo{
				HashFunc:     crypto.SHA256,
				HashEncoding: encoding.Base64,
			},
		}
		scriptBytes, _ := json.Marshal(script)
		htlcOwner, _ := identity.WrapWithType(htlc.ScriptType, scriptBytes)

		ta := &actions.TransferAction{
			Outputs: []*actions.Output{
				{Owner: htlcOwner, Type: "ABC", Quantity: "100"},
			},
		}
		c := &validator.Context{
			TransferAction: ta,
			InputTokens:    []*actions.Output{{Owner: sender, Type: "ABC", Quantity: "100"}},
		}
		err := validator.TransferHTLCValidate(ctx, c)
		require.Error(t, err)
		assert.Contains(t, err.Error(), "expiration date has already passed")
	})
}
