/*
Copyright IBM Corp. All Rights Reserved.

SPDX-License-Identifier: Apache-2.0
*/

package validator_test

import (
	"bytes"
	"context"
	"crypto"
	"encoding/json"
	"testing"
	"time"

	math "github.com/IBM/mathlib"
	"github.com/hyperledger-labs/fabric-smart-client/pkg/utils/errors"
	fv1 "github.com/hyperledger-labs/fabric-token-sdk/token/core/fabtoken/v1/actions"
	"github.com/hyperledger-labs/fabric-token-sdk/token/core/zkatdlog/nogh/v1/benchmark"
	"github.com/hyperledger-labs/fabric-token-sdk/token/core/zkatdlog/nogh/v1/issue"
	v1 "github.com/hyperledger-labs/fabric-token-sdk/token/core/zkatdlog/nogh/v1/setup"
	"github.com/hyperledger-labs/fabric-token-sdk/token/core/zkatdlog/nogh/v1/token"
	"github.com/hyperledger-labs/fabric-token-sdk/token/core/zkatdlog/nogh/v1/transfer"
	"github.com/hyperledger-labs/fabric-token-sdk/token/core/zkatdlog/nogh/v1/validator"
	testing2 "github.com/hyperledger-labs/fabric-token-sdk/token/core/zkatdlog/nogh/v1/validator/testutils"
	"github.com/hyperledger-labs/fabric-token-sdk/token/driver"
	mock3 "github.com/hyperledger-labs/fabric-token-sdk/token/driver/mock"
	benchmark2 "github.com/hyperledger-labs/fabric-token-sdk/token/services/benchmark"
	"github.com/hyperledger-labs/fabric-token-sdk/token/services/identity"
	"github.com/hyperledger-labs/fabric-token-sdk/token/services/identity/idemixnym"
	"github.com/hyperledger-labs/fabric-token-sdk/token/services/interop/encoding"
	"github.com/hyperledger-labs/fabric-token-sdk/token/services/interop/htlc"
	"github.com/hyperledger-labs/fabric-token-sdk/token/services/logging"
	"github.com/stretchr/testify/require"
)

var testUseCaseExtra = &benchmark2.Case{
	Bits:       32,
	CurveID:    math.BLS12_381_BBS_GURVY,
	NumInputs:  2,
	NumOutputs: 2,
}

type mockSignatureProvider struct {
	HasBeenSignedByFunc func(ctx context.Context, id driver.Identity, verifier driver.Verifier) ([]byte, error)
}

func (m *mockSignatureProvider) HasBeenSignedBy(ctx context.Context, id driver.Identity, verifier driver.Verifier) ([]byte, error) {
	if m.HasBeenSignedByFunc != nil {
		return m.HasBeenSignedByFunc(ctx, id, verifier)
	}

	return nil, nil
}

func (m *mockSignatureProvider) Signatures() [][]byte {
	return nil
}

func TestIssueValidateErrors(t *testing.T) {
	configurations, err := benchmark.NewSetupConfigurations("./../testdata", []uint64{testUseCaseExtra.Bits}, []math.CurveID{testUseCaseExtra.CurveID}, idemixnym.IdentityType)
	require.NoError(t, err)
	env, err := testing2.NewEnv(testUseCaseExtra, configurations)
	require.NoError(t, err)

	issueAction := &issue.Action{}
	err = issueAction.Deserialize(env.TRWithIssue.Issues[0])
	require.NoError(t, err)

	newCtx := func() *validator.Context {
		return &validator.Context{
			Logger:       logging.MustGetLogger(),
			PP:           env.Engine.PublicParams,
			IssueAction:  issueAction,
			Deserializer: env.Engine.Deserializer,
			SignatureProvider: &mockSignatureProvider{
				HasBeenSignedByFunc: func(ctx context.Context, id driver.Identity, verifier driver.Verifier) ([]byte, error) {
					return []byte("sig"), nil
				},
			},
		}
	}

	// Case 1: action.Validate() fails
	oldOutputs := issueAction.Outputs
	issueAction.Outputs = nil
	err = validator.IssueValidate(context.Background(), newCtx())
	require.Error(t, err)
	require.Contains(t, err.Error(), "failed validating issue action")
	issueAction.Outputs = oldOutputs

	// Case 3: Issuer not in PP.Issuers()
	ctx := newCtx()
	oldIssuerIDs := ctx.PP.IssuerIDs
	ctx.PP.IssuerIDs = []driver.Identity{[]byte("other-issuer")}
	err = validator.IssueValidate(context.Background(), ctx)
	require.Error(t, err)
	require.Contains(t, err.Error(), "issuer is not authorized")
	ctx.PP.IssuerIDs = oldIssuerIDs

	// Case 4: Deserializer.GetIssuerVerifier fails
	ctx = newCtx()
	mockDes := &mock3.Deserializer{}
	mockDes.GetIssuerVerifierReturns(nil, errors.New("failed getting verifier"))
	ctx.Deserializer = mockDes
	err = validator.IssueValidate(context.Background(), ctx)
	require.Error(t, err)
	require.Contains(t, err.Error(), "failed getting verifier for issuer")

	// Case 5: SignatureProvider.HasBeenSignedBy fails
	ctx = newCtx()
	mockSigProv := &mockSignatureProvider{
		HasBeenSignedByFunc: func(ctx context.Context, id driver.Identity, verifier driver.Verifier) ([]byte, error) {
			return nil, errors.New("signature verification failed")
		},
	}
	ctx.SignatureProvider = mockSigProv
	err = validator.IssueValidate(context.Background(), ctx)
	require.Error(t, err)
	require.Contains(t, err.Error(), "failed verifying signature")

	// Case 6: action.Validate() fails again (nil output)
	issueAction.Outputs = []*token.Token{nil}
	err = validator.IssueValidate(context.Background(), newCtx())
	require.Error(t, err)
	require.Contains(t, err.Error(), "failed validating issue action")
	issueAction.Outputs = oldOutputs

	// Case 7: Verify failure
	issueAction.Proof = []byte("invalid-proof")
	err = validator.IssueValidate(context.Background(), newCtx())
	require.Error(t, err)
	// Reset to valid state
	err = issueAction.Deserialize(env.TRWithIssue.Issues[0])
	require.NoError(t, err)

	// Case 8: Success with issuers list
	ctx = newCtx()
	ctx.PP.IssuerIDs = []driver.Identity{issueAction.Issuer}
	err = validator.IssueValidate(context.Background(), ctx)
	require.NoError(t, err)
}

func TestTransferSignatureValidateErrors(t *testing.T) {
	pp, err := v1.Setup(32, []byte("idemix"), math.BLS12_381_BBS_GURVY)
	require.NoError(t, err)

	ctx := &validator.Context{
		Logger:         logging.MustGetLogger(),
		TransferAction: &transfer.Action{},
		PP:             pp,
	}

	// Case: len(ctx.TransferAction.Inputs) == 0
	err = validator.TransferSignatureValidate(context.Background(), ctx)
	require.Error(t, err)
	require.Contains(t, err.Error(), "invalid number of token inputs, expected at least 1")

	// Case: Deserializer.GetOwnerVerifier fails
	ctx.TransferAction.Inputs = []*transfer.ActionInput{{}}
	ctx.TransferAction.Inputs[0].Token = &token.Token{Owner: []byte("owner1")}
	mockDes := &mock3.Deserializer{}
	mockDes.GetOwnerVerifierReturns(nil, errors.New("deserialization failed"))
	ctx.Deserializer = mockDes
	err = validator.TransferSignatureValidate(context.Background(), ctx)
	require.Error(t, err)
	require.Contains(t, err.Error(), "failed deserializing owner")

	// Case: SignatureProvider.HasBeenSignedBy fails for owner
	mockDes.GetOwnerVerifierReturns(&mock3.Verifier{}, nil)
	mockSigProv := &mockSignatureProvider{
		HasBeenSignedByFunc: func(ctx context.Context, id driver.Identity, verifier driver.Verifier) ([]byte, error) {
			return nil, errors.New("signature verification failed")
		},
	}
	ctx.SignatureProvider = mockSigProv
	err = validator.TransferSignatureValidate(context.Background(), ctx)
	require.Error(t, err)
	require.Contains(t, err.Error(), "failed signature verification")

	// Case: Redeem action, issuer missing
	mockSigProv.HasBeenSignedByFunc = func(ctx context.Context, id driver.Identity, verifier driver.Verifier) ([]byte, error) {
		return []byte("sig"), nil
	}
	ctx.TransferAction.Outputs = []*token.Token{{Owner: nil, Data: pp.PedersenGenerators[0]}}
	ctx.PP.IssuerIDs = []driver.Identity{[]byte("issuer1")}
	err = validator.TransferSignatureValidate(context.Background(), ctx)
	require.Error(t, err)
	require.Contains(t, err.Error(), "On Redeem action, must have at least one issuer")

	// Case: Redeem action, issuer verifier fails
	ctx.TransferAction.Issuer = []byte("issuer1")
	mockDes.GetIssuerVerifierReturns(nil, errors.New("failed getting issuer verifier"))
	err = validator.TransferSignatureValidate(context.Background(), ctx)
	require.Error(t, err)
	require.Contains(t, err.Error(), "failed deserializing issuer")

	// Case: Redeem action, issuer signature fails
	mockDes.GetIssuerVerifierReturns(&mock3.Verifier{}, nil)
	mockSigProv = &mockSignatureProvider{
		HasBeenSignedByFunc: func(ctx context.Context, id driver.Identity, verifier driver.Verifier) ([]byte, error) {
			if bytes.Equal(id, []byte("issuer1")) {
				return nil, errors.New("issuer signature failed")
			}

			return []byte("sig"), nil
		},
	}
	ctx.SignatureProvider = mockSigProv
	err = validator.TransferSignatureValidate(context.Background(), ctx)
	require.Error(t, err)
	require.Contains(t, err.Error(), "failed signature verification")
}

func TestTransferUpgradeWitnessValidateErrors(t *testing.T) {
	pp, err := v1.Setup(32, []byte("idemix"), math.BLS12_381_BBS_GURVY)
	require.NoError(t, err)

	action := &transfer.Action{
		Inputs: []*transfer.ActionInput{
			{
				UpgradeWitness: &token.UpgradeWitness{},
			},
		},
	}
	ctx := &validator.Context{
		PP:             pp,
		TransferAction: action,
	}

	// Case: witness.FabToken is nil
	err = validator.TransferUpgradeWitnessValidate(context.Background(), ctx)
	require.Error(t, err)
	require.Contains(t, err.Error(), "fabtoken token not found in witness")

	// Case: token2.ToQuantity fails
	action.Inputs[0].UpgradeWitness.FabToken = &fv1.Output{
		Quantity: "invalid",
	}
	err = validator.TransferUpgradeWitnessValidate(context.Background(), ctx)
	require.Error(t, err)
	require.Contains(t, err.Error(), "failed to unmarshal quantity")

	// Case: recomputed commitment does not match
	curve := math.Curves[pp.Curve]
	rand, _ := curve.Rand()
	bf := curve.NewRandomZr(rand)
	action.Inputs[0].UpgradeWitness.FabToken = &fv1.Output{
		Quantity: "0x10",
		Type:     "ABC",
		Owner:    []byte("owner1"),
	}
	action.Inputs[0].UpgradeWitness.BlindingFactor = bf
	action.Inputs[0].Token = &token.Token{
		Data:  pp.PedersenGenerators[0], // wrong commitment
		Owner: []byte("owner1"),
	}
	err = validator.TransferUpgradeWitnessValidate(context.Background(), ctx)
	require.Error(t, err)
	require.Contains(t, err.Error(), "recomputed commitment does not match")

	// Case: owners do not correspond
	toks, _, err := token.GetTokensWithWitnessAndBF([]uint64{16}, []*math.Zr{bf}, "ABC", pp.PedersenGenerators, curve)
	require.NoError(t, err)
	action.Inputs[0].Token.Data = toks[0]
	action.Inputs[0].Token.Owner = []byte("owner2") // different owner
	err = validator.TransferUpgradeWitnessValidate(context.Background(), ctx)
	require.Error(t, err)
	require.Contains(t, err.Error(), "owners do not correspond")
}

func TestTransferZKProofValidateErrors(t *testing.T) {
	pp, err := v1.Setup(32, []byte("idemix"), math.BLS12_381_BBS_GURVY)
	require.NoError(t, err)

	ctx := &validator.Context{
		PP: pp,
		InputTokens: []*token.Token{
			{Data: pp.PedersenGenerators[0]},
		},
		TransferAction: &transfer.Action{
			Outputs: []*token.Token{
				{Data: pp.PedersenGenerators[1]},
			},
			Proof: []byte("invalid-proof"),
		},
	}

	err = validator.TransferZKProofValidate(context.Background(), ctx)
	require.Error(t, err)
}

func TestTransferHTLCValidateErrors(t *testing.T) {
	validSenderID, _ := identity.WrapWithType("x509", []byte("owner1"))
	validRecipientID, _ := identity.WrapWithType("x509", []byte("recipient"))

	newCtx := func() *validator.Context {
		return &validator.Context{
			Logger: logging.MustGetLogger(),
			InputTokens: []*token.Token{
				{Owner: validSenderID},
			},
			TransferAction: &transfer.Action{
				Inputs: []*transfer.ActionInput{{}},
			},
			Signatures:      [][]byte{[]byte("sig")},
			MetadataCounter: make(map[string]int),
		}
	}

	// Case: identity.UnmarshalTypedIdentity(in.Owner) fails
	ctx := newCtx()
	ctx.InputTokens[0].Owner = []byte("invalid-identity")
	err := validator.TransferHTLCValidate(context.Background(), ctx)
	require.Error(t, err)
	require.Contains(t, err.Error(), "failed to unmarshal owner of input token")

	// Preparation
	script := &htlc.Script{
		Sender:    validSenderID,
		Recipient: validRecipientID,
		Deadline:  time.Now().Add(100 * time.Hour),
		HashInfo: htlc.HashInfo{
			Hash:         []byte("hash"),
			HashFunc:     crypto.SHA256,
			HashEncoding: encoding.Hex,
		},
	}
	scriptBytes, _ := json.Marshal(script)
	lockID, _ := identity.WrapWithType(htlc.ScriptType, scriptBytes)

	// Case: HTLC script, but invalid number of inputs/outputs
	ctx = newCtx()
	ctx.InputTokens[0].Owner = lockID
	ctx.TransferAction.Inputs = []*transfer.ActionInput{{}, {}} // 2 inputs
	ctx.InputTokens = append(ctx.InputTokens, &token.Token{Owner: validSenderID})
	ctx.Signatures = append(ctx.Signatures, []byte("sig2"))
	err = validator.TransferHTLCValidate(context.Background(), ctx)
	require.Error(t, err)
	require.Contains(t, err.Error(), "an htlc script only transfers the ownership of a token")

	// Case: output owner type script invalid
	ctx = newCtx()
	ctx.InputTokens[0].Owner = validSenderID
	ctx.TransferAction.Inputs = []*transfer.ActionInput{{}}
	ctx.TransferAction.Outputs = []*token.Token{
		{Owner: lockID, Data: &math.G1{}},
	}
	invalidScriptID, _ := identity.WrapWithType(htlc.ScriptType, []byte("{}"))
	ctx.TransferAction.Outputs[0].Owner = invalidScriptID
	err = validator.TransferHTLCValidate(context.Background(), ctx)
	require.Error(t, err)
	require.Contains(t, err.Error(), "htlc script invalid")

	// Case: successful HTLC lock
	ctx = newCtx()
	ctx.InputTokens[0].Owner = validSenderID
	ctx.TransferAction.Outputs = []*token.Token{{Owner: lockID, Data: &math.G1{}}}
	ctx.TransferAction.Metadata = map[string][]byte{
		htlc.LockKey(script.HashInfo.Hash): htlc.LockValue(script.HashInfo.Hash),
	}
	err = validator.TransferHTLCValidate(context.Background(), ctx)
	require.NoError(t, err)

	// Case: successful HTLC claim
	ctx = newCtx()
	ctx.InputTokens[0].Owner = lockID
	scriptActual := &htlc.Script{}
	require.NoError(t, json.Unmarshal(scriptBytes, scriptActual))
	ctx.TransferAction.Outputs = []*token.Token{{Owner: scriptActual.Recipient.Bytes(), Data: &math.G1{}}}
	preimage := []byte("preimage")
	image, _ := script.HashInfo.Image(preimage)
	claimSigBytes, _ := json.Marshal(&htlc.ClaimSignature{Preimage: preimage, RecipientSignature: []byte("sig")})
	ctx.Signatures = [][]byte{claimSigBytes}
	ctx.TransferAction.Metadata = map[string][]byte{htlc.ClaimKey(image): preimage}
	err = validator.TransferHTLCValidate(context.Background(), ctx)
	require.NoError(t, err)

	// Case: HTLC reclaim
	ctx = newCtx()
	script.Deadline = time.Now().Add(-100 * time.Hour) // expired
	scriptBytes, _ = json.Marshal(script)
	lockID, _ = identity.WrapWithType(htlc.ScriptType, scriptBytes)
	ctx.InputTokens[0].Owner = lockID
	require.NoError(t, json.Unmarshal(scriptBytes, scriptActual))
	ctx.TransferAction.Outputs = []*token.Token{{Owner: scriptActual.Sender.Bytes(), Data: &math.G1{}}}
	err = validator.TransferHTLCValidate(context.Background(), ctx)
	require.NoError(t, err)

	// Case: MetadataLockKeyCheck fails
	ctx = newCtx()
	script.Deadline = time.Now().Add(1 * time.Hour) // not expired
	scriptBytes, _ = json.Marshal(script)
	lockID, _ = identity.WrapWithType(htlc.ScriptType, scriptBytes)
	ctx.InputTokens[0].Owner = lockID
	require.NoError(t, json.Unmarshal(scriptBytes, scriptActual))
	ctx.TransferAction.Outputs = []*token.Token{{Owner: scriptActual.Recipient.Bytes(), Data: &math.G1{}}}
	ctx.TransferAction.Metadata = nil
	err = validator.TransferHTLCValidate(context.Background(), ctx)
	require.Error(t, err)
	require.Contains(t, err.Error(), "failed to check htlc metadata")

	// Case: invalid output type (nil)
	ctx = newCtx()
	ctx.InputTokens[0].Owner = lockID // to enter first loop branch and then panic if no nil check
	ctx.TransferAction.Outputs = []*token.Token{nil}
	err = validator.TransferHTLCValidate(context.Background(), ctx)
	require.Error(t, err)
	require.Contains(t, err.Error(), "invalid transfer action: an htlc script only transfers the ownership of a token, output not found")

	// Case: output is redeem
	ctx = newCtx()
	ctx.TransferAction.Outputs = []*token.Token{{Owner: nil}}
	err = validator.TransferHTLCValidate(context.Background(), ctx)
	require.NoError(t, err)

	// Case: output owner identity unmarshal fails
	ctx = newCtx()
	ctx.TransferAction.Outputs = []*token.Token{{Owner: []byte("invalid")}}
	err = validator.TransferHTLCValidate(context.Background(), ctx)
	require.Error(t, err)

	// Case: script.FromBytes failure
	ctx = newCtx()
	invalidJSONID, _ := identity.WrapWithType(htlc.ScriptType, []byte("invalid"))
	ctx.TransferAction.Outputs = []*token.Token{{Owner: invalidJSONID}}
	err = validator.TransferHTLCValidate(context.Background(), ctx)
	require.Error(t, err)
}

func TestDeserializeActionsErrors(t *testing.T) {
	ad := &validator.ActionDeserializer{}
	tr := &driver.TokenRequest{
		Issues: [][]byte{[]byte("invalid")},
	}
	_, _, err := ad.DeserializeActions(tr)
	require.Error(t, err)

	tr = &driver.TokenRequest{
		Transfers: [][]byte{[]byte("invalid")},
	}
	_, _, err = ad.DeserializeActions(tr)
	require.Error(t, err)
}

// TestTransferVerifierCache tests that when multiple inputs have the same owner,
// the verifier is only deserialized once (cache hit)
func TestTransferVerifierCache(t *testing.T) {
	ctx := context.Background()

	// Create a transfer action with 3 inputs, where 2 have the same owner
	sameOwner := []byte("same-owner-identity")
	differentOwner := []byte("different-owner-identity")

	transferAction := &transfer.Action{
		Inputs: []*transfer.ActionInput{
			{Token: &token.Token{Owner: sameOwner}},      // Input 0: same owner
			{Token: &token.Token{Owner: sameOwner}},      // Input 1: same owner (should hit cache)
			{Token: &token.Token{Owner: differentOwner}}, // Input 2: different owner
		},
		Outputs: []*token.Token{
			{Owner: []byte("output-owner")},
		},
	}

	// Track deserializer calls
	deserializerCallCount := 0
	ownerCallsByIdentity := make(map[string]int)

	mockDes := &mock3.Deserializer{}
	mockDes.GetOwnerVerifierCalls(func(ctx context.Context, id driver.Identity) (driver.Verifier, error) {
		deserializerCallCount++
		key := string(id)
		ownerCallsByIdentity[key]++

		return &mock3.Verifier{}, nil
	})

	mockSigProv := &mockSignatureProvider{
		HasBeenSignedByFunc: func(ctx context.Context, id driver.Identity, verifier driver.Verifier) ([]byte, error) {
			return []byte("signature"), nil
		},
	}

	validatorCtx := &validator.Context{
		Logger:            &logging.MockLogger{},
		TransferAction:    transferAction,
		Deserializer:      mockDes,
		SignatureProvider: mockSigProv,
		PP:                &v1.PublicParams{},
	}

	// Execute validation
	err := validator.TransferSignatureValidate(ctx, validatorCtx)
	require.NoError(t, err)

	// Verify results - WITH cache implementation
	t.Logf("Total GetOwnerVerifier calls: %d", deserializerCallCount)
	t.Logf("Calls for same owner: %d", ownerCallsByIdentity[string(sameOwner)])
	t.Logf("Calls for different owner: %d", ownerCallsByIdentity[string(differentOwner)])

	// With verifier cache: deserializer is called only once per unique owner
	// Expected: 2 calls total (once for sameOwner, once for differentOwner)
	require.Equal(t, 2, deserializerCallCount, "With cache: should call deserializer only for unique owners")
	require.Equal(t, 1, ownerCallsByIdentity[string(sameOwner)], "With cache: same owner called only once")
	require.Equal(t, 1, ownerCallsByIdentity[string(differentOwner)], "Different owner called once")

	// Calculate cache effectiveness
	uniqueOwners := 2
	totalInputs := 3
	cacheHits := totalInputs - uniqueOwners
	cacheHitRate := float64(cacheHits) / float64(totalInputs) * 100

	t.Logf("Cache hit rate: %.1f%% (%d cache hits out of %d inputs)",
		cacheHitRate, cacheHits, totalInputs)
}

// TestTransferVerifierCacheError tests that cache doesn't interfere with error handling
func TestTransferVerifierCacheError(t *testing.T) {
	ctx := context.Background()

	sameOwner := []byte("same-owner-identity")

	transferAction := &transfer.Action{
		Inputs: []*transfer.ActionInput{
			{Token: &token.Token{Owner: sameOwner}},
			{Token: &token.Token{Owner: sameOwner}}, // Should use cached verifier
		},
		Outputs: []*token.Token{
			{Owner: []byte("output-owner")},
		},
	}

	// Mock deserializer that returns error on first call
	callCount := 0
	mockDes := &mock3.Deserializer{}
	mockDes.GetOwnerVerifierCalls(func(ctx context.Context, id driver.Identity) (driver.Verifier, error) {
		callCount++
		if callCount == 1 {
			return nil, errors.New("deserializer error")
		}

		return &mock3.Verifier{}, nil
	})

	mockSigProv := &mockSignatureProvider{
		HasBeenSignedByFunc: func(ctx context.Context, id driver.Identity, verifier driver.Verifier) ([]byte, error) {
			return []byte("signature"), nil
		},
	}

	validatorCtx := &validator.Context{
		Logger:            &logging.MockLogger{},
		TransferAction:    transferAction,
		Deserializer:      mockDes,
		SignatureProvider: mockSigProv,
		PP:                &v1.PublicParams{},
	}

	// Execute validation - should fail on first input
	err := validator.TransferSignatureValidate(ctx, validatorCtx)
	require.Error(t, err)
	require.Contains(t, err.Error(), "failed deserializing owner")
	require.Equal(t, 1, callCount, "Should stop on first error")
}

// TestTransferVerifierCacheSignatureError tests signature verification error with cache
func TestTransferVerifierCacheSignatureError(t *testing.T) {
	ctx := context.Background()

	sameOwner := []byte("same-owner-identity")

	transferAction := &transfer.Action{
		Inputs: []*transfer.ActionInput{
			{Token: &token.Token{Owner: sameOwner}},
			{Token: &token.Token{Owner: sameOwner}}, // Should use cached verifier
		},
		Outputs: []*token.Token{
			{Owner: []byte("output-owner")},
		},
	}

	mockDes := &mock3.Deserializer{}
	mockDes.GetOwnerVerifierCalls(func(ctx context.Context, id driver.Identity) (driver.Verifier, error) {
		return &mock3.Verifier{}, nil
	})

	// Mock signature provider that fails on second call
	sigCallCount := 0
	mockSigProv := &mockSignatureProvider{
		HasBeenSignedByFunc: func(ctx context.Context, id driver.Identity, verifier driver.Verifier) ([]byte, error) {
			sigCallCount++
			if sigCallCount == 2 {
				return nil, errors.New("signature verification failed")
			}

			return []byte("signature"), nil
		},
	}

	validatorCtx := &validator.Context{
		Logger:            &logging.MockLogger{},
		TransferAction:    transferAction,
		Deserializer:      mockDes,
		SignatureProvider: mockSigProv,
		PP:                &v1.PublicParams{},
	}

	// Execute validation - should fail on second input signature verification
	err := validator.TransferSignatureValidate(ctx, validatorCtx)
	require.Error(t, err)
	require.Contains(t, err.Error(), "failed signature verification")
}
