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
	"github.com/hyperledger-labs/fabric-token-sdk/token/services/identity/x509"
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
		tr := &driver.TokenRequest{
			Issues: [][]byte{ia1Bytes},
		}
		assert.Len(t, ia, 1)
		assert.Equal(t, ia1.Issuer, ia[0].Issuer)
	t.Run("WithTransfer", func(t *testing.T) {
		ta1 := &actions.TransferAction{Issuer: []byte("issuer1")}
		ta1Bytes, err := ta1.Serialize()
			Transfers: [][]byte{ta1Bytes},
		assert.Len(t, ta, 1)
		assert.Equal(t, ta1.Issuer, ta[0].Issuer)
	t.Run("IssueDeserializeError", func(t *testing.T) {
			Issues: [][]byte{[]byte("invalid")},
		_, _, err := ad.DeserializeActions(tr)
		require.Error(t, err)
	t.Run("TransferDeserializeError", func(t *testing.T) {
			Transfers: [][]byte{[]byte("invalid")},
}
func TestNewValidator(t *testing.T) {
	logger := logging.MustGetLogger("test")
	pp := &setup.PublicParams{}
	deserializer := &mock.Deserializer{}
	v := validator.NewValidator(logger, pp, deserializer, nil, nil, nil)
	assert.NotNil(t, v)
func TestIssueValidate(t *testing.T) {
	ctx := context.Background()
	pp := &setup.PublicParams{
		QuantityPrecision: 64,
		IssuerIDs:         []driver.Identity{[]byte("issuer1")},
	}
	sigProvider := &mock.SignatureProvider{}
	t.Run("EmptyOutputs", func(t *testing.T) {
		ia := &actions.IssueAction{
			Issuer: []byte("issuer1"),
		c := &validator.Context{
			PP:          pp,
			IssueAction: ia,
		err := validator.IssueValidate(ctx, c)
		assert.Contains(t, err.Error(), "no outputs in issue action")
	t.Run("ZeroQuantity", func(t *testing.T) {
			Outputs: []*actions.Output{
				{
					Quantity: "0",
					Type:     "ABC",
					Owner:    []byte("owner1"),
				},
			},
		assert.Contains(t, err.Error(), "quantity is zero")
	t.Run("UnauthorizedIssuer", func(t *testing.T) {
			Issuer: []byte("unauthorized"),
					Quantity: "100",
		require.ErrorIs(t, err, validator2.ErrIssuerNotAuthorized)
		assert.Contains(t, err.Error(), validator2.ErrIssuerNotAuthorized.Error())
	t.Run("VerifierError", func(t *testing.T) {
		deserializer.GetIssuerVerifierReturns(nil, assert.AnError)
			PP:           pp,
			IssueAction:  ia,
			Deserializer: deserializer,
		assert.Contains(t, err.Error(), "failed getting verifier for issuer identity")
	t.Run("SignatureVerificationError", func(t *testing.T) {
		deserializer.GetIssuerVerifierReturns(&mock.Verifier{}, nil)
		sigProvider.HasBeenSignedByReturns(nil, assert.AnError)
			PP:                pp,
			IssueAction:       ia,
			Deserializer:      deserializer,
			SignatureProvider: sigProvider,
		assert.Contains(t, err.Error(), "failed verifying signature")
	t.Run("Success", func(t *testing.T) {
		sigProvider.HasBeenSignedByReturns([]byte("signature"), nil)
	t.Run("SuccessNoIssuersInPP", func(t *testing.T) {
		ppNoIssuers := &setup.PublicParams{
			QuantityPrecision: 64,
			Issuer: []byte("any-issuer"),
			PP:                ppNoIssuers,
func TestTransferActionValidate(t *testing.T) {
	ta := &actions.TransferAction{
		Inputs: []*actions.TransferActionInput{
			{
				ID: &token.ID{TxId: "tx1", Index: 0},
				Input: &actions.Output{
		},
		Outputs: []*actions.Output{
				Quantity: "100",
				Type:     "ABC",
				Owner:    []byte("owner1"),
	c := &validator.Context{
		TransferAction: ta,
	err := validator.TransferActionValidate(ctx, c)
	require.NoError(t, err)
func TestTransferSignatureValidate(t *testing.T) {
		IssuerIDs: []driver.Identity{[]byte("issuer1")},
	t.Run("NoInputs", func(t *testing.T) {
		ta := &actions.TransferAction{}
			TransferAction: ta,
		err := validator.TransferSignatureValidate(ctx, c)
		assert.Contains(t, err.Error(), "expected at least 1")
	t.Run("OwnerVerifierError", func(t *testing.T) {
		deserializer := &mock.Deserializer{}
		ta := &actions.TransferAction{
			Inputs: []*actions.TransferActionInput{
					Input: &actions.Output{
						Owner: []byte("owner1"),
					},
		deserializer.GetOwnerVerifierReturns(nil, assert.AnError)
			Deserializer:   deserializer,
			Logger:         logger,
		assert.Contains(t, err.Error(), "failed deserializing owner")
	t.Run("OwnerSignatureError", func(t *testing.T) {
		sigProvider := &mock.SignatureProvider{}
		deserializer.GetOwnerVerifierReturns(&mock.Verifier{}, nil)
			TransferAction:    ta,
			Logger:            logger,
		assert.Contains(t, err.Error(), "failed signature verification")
	t.Run("SuccessTransfer", func(t *testing.T) {
					Owner: []byte("owner2"),
		sigProvider.HasBeenSignedByReturns([]byte("sig1"), nil)
		assert.Len(t, c.Signatures, 1)
		assert.Len(t, c.InputTokens, 1)
	t.Run("RedeemWithoutIssuer", func(t *testing.T) {
					Owner: nil, // redeem
		assert.Contains(t, err.Error(), "must have at least one issuer")
	t.Run("RedeemWithIssuer", func(t *testing.T) {
		sigProvider.HasBeenSignedByReturns([]byte("sig"), nil)
		assert.Len(t, c.Signatures, 2) // one for owner, one for issuer
	t.Run("RedeemIssuerVerifierError", func(t *testing.T) {
		assert.Contains(t, err.Error(), "failed deserializing issuer")
	t.Run("RedeemIssuerSignatureError", func(t *testing.T) {
		// first call (for owner) returns success
		sigProvider.HasBeenSignedByReturnsOnCall(0, []byte("sig-owner"), nil)
		// second call (for issuer) returns error
		sigProvider.HasBeenSignedByReturnsOnCall(1, nil, assert.AnError)
func TestTransferBalanceValidate(t *testing.T) {
	t.Run("NoOutputs", func(t *testing.T) {
		err := validator.TransferBalanceValidate(ctx, c)
		assert.Contains(t, err.Error(), "there is no output")
			Outputs: []*actions.Output{{Quantity: "100"}},
			InputTokens:    nil,
		assert.Contains(t, err.Error(), "there is no input")
	t.Run("NilFirstInput", func(t *testing.T) {
			InputTokens:    []*actions.Output{nil},
		assert.Contains(t, err.Error(), "first input is nil")
	t.Run("NilInput", func(t *testing.T) {
			PP:             pp,
			InputTokens:    []*actions.Output{{Quantity: "100"}, nil},
		assert.Contains(t, err.Error(), "input 1 is nil")
	t.Run("ParseQuantityInputError", func(t *testing.T) {
			InputTokens:    []*actions.Output{{Quantity: "invalid"}},
		assert.Contains(t, err.Error(), "failed parsing quantity")
	t.Run("MismatchedInputType", func(t *testing.T) {
			Outputs: []*actions.Output{{Quantity: "100", Type: "ABC"}},
			InputTokens:    []*actions.Output{{Quantity: "50", Type: "ABC"}, {Quantity: "50", Type: "XYZ"}},
		assert.Contains(t, err.Error(), "does not match type")
	t.Run("ParseQuantityOutputError", func(t *testing.T) {
			Outputs: []*actions.Output{{Quantity: "invalid"}},
			InputTokens:    []*actions.Output{{Quantity: "100"}},
	t.Run("MismatchedOutputType", func(t *testing.T) {
			Outputs: []*actions.Output{{Quantity: "100", Type: "XYZ"}},
			InputTokens:    []*actions.Output{{Quantity: "100", Type: "ABC"}},
	t.Run("Unbalanced", func(t *testing.T) {
			Outputs: []*actions.Output{{Quantity: "101", Type: "ABC"}},
		assert.Contains(t, err.Error(), "does not match output sum")
			Outputs: []*actions.Output{{Quantity: "100", Type: "ABC"}, {Quantity: "50", Type: "ABC"}},
			InputTokens:    []*actions.Output{{Quantity: "150", Type: "ABC"}},
func TestTransferHTLCValidate(t *testing.T) {
	t.Run("NoHTLC", func(t *testing.T) {
		owner1, _ := identity.WrapWithType(x509.IdentityType, []byte("owner1"))
		owner2, _ := identity.WrapWithType(x509.IdentityType, []byte("owner2"))
			Outputs: []*actions.Output{{Owner: owner1}},
			InputTokens:    []*actions.Output{{Owner: owner2}},
		err := validator.TransferHTLCValidate(ctx, c)
	t.Run("InputIsHTLC_Reclaim_Success", func(t *testing.T) {
		sender, _ := identity.WrapWithType(x509.IdentityType, []byte("sender"))
		recipient, _ := identity.WrapWithType(x509.IdentityType, []byte("recipient"))
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
		scriptBytes, err := json.Marshal(script)
		htlcOwner, err := identity.WrapWithType(htlc.ScriptType, scriptBytes)
				{Owner: sender, Type: "ABC", Quantity: "100"},
			TransferAction:  ta,
			InputTokens:     []*actions.Output{{Owner: htlcOwner, Type: "ABC", Quantity: "100"}},
			Signatures:      [][]byte{[]byte("sig")},
			MetadataCounter: make(map[string]int),
		err = validator.TransferHTLCValidate(ctx, c)
	t.Run("InputIsHTLC_Claim_Success", func(t *testing.T) {
		// encoded image for claim key
		imgEncoded := []byte(encoding.Base64.New().EncodeToString(img))
			Deadline:  time.Now().Add(1 * time.Hour), // not expired
				Hash:         imgEncoded,
		claimSig := &htlc.ClaimSignature{
			Preimage:           preimage,
			RecipientSignature: []byte("rec-sig"),
		claimSigBytes, err := json.Marshal(claimSig)
				{Owner: recipient, Type: "ABC", Quantity: "100"},
			Metadata: map[string][]byte{
				htlc.ClaimKey(imgEncoded): preimage,
			Signatures:      [][]byte{claimSigBytes},
		assert.Equal(t, 1, c.MetadataCounter[htlc.ClaimKey(imgEncoded)])
	t.Run("OutputIsHTLC_Success", func(t *testing.T) {
				{Owner: htlcOwner, Type: "ABC", Quantity: "100"},
				htlc.LockKey(img): htlc.LockValue(img),
			InputTokens:     []*actions.Output{{Owner: sender, Type: "ABC", Quantity: "100"}},
		assert.Equal(t, 1, c.MetadataCounter[htlc.LockKey(img)])
	t.Run("InputIsHTLC_InvalidOwner", func(t *testing.T) {
		htlcOwner := []byte("invalid-typed-identity")
			Outputs: []*actions.Output{{Owner: []byte("rec")}},
			InputTokens:    []*actions.Output{{Owner: htlcOwner}},
		assert.Contains(t, err.Error(), "failed to unmarshal owner")
	t.Run("InputIsHTLC_TwoOutputs_Error", func(t *testing.T) {
			Deadline:  time.Now().Add(1 * time.Hour),
		scriptBytes, _ := json.Marshal(script)
		htlcOwner, _ := identity.WrapWithType(htlc.ScriptType, scriptBytes)
				{Owner: sender, Type: "ABC", Quantity: "50"},
			InputTokens:    []*actions.Output{{Owner: htlcOwner, Type: "ABC", Quantity: "150"}},
		assert.Contains(t, err.Error(), "an htlc script only transfers the ownership of a token")
	t.Run("InputIsHTLC_TypeMismatch_Error", func(t *testing.T) {
				{Owner: recipient, Type: "XYZ", Quantity: "100"},
			InputTokens:    []*actions.Output{{Owner: htlcOwner, Type: "ABC", Quantity: "100"}},
		assert.Contains(t, err.Error(), "type of input does not match type of output")
	t.Run("InputIsHTLC_QuantityMismatch_Error", func(t *testing.T) {
				{Owner: recipient, Type: "ABC", Quantity: "101"},
		assert.Contains(t, err.Error(), "quantity of input does not match quantity of output")
	t.Run("InputIsHTLC_Redeem_Error", func(t *testing.T) {
				{Owner: nil, Type: "ABC", Quantity: "100"},
		assert.Contains(t, err.Error(), "should not be a redeem")
	t.Run("OutputIsHTLC_Expired_Error", func(t *testing.T) {
			InputTokens:    []*actions.Output{{Owner: sender, Type: "ABC", Quantity: "100"}},
		assert.Contains(t, err.Error(), "expiration date has already passed")
