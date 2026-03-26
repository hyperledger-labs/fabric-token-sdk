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
	"github.com/hyperledger-labs/fabric-token-sdk/token/services/identity/x509"
	mock3 "github.com/hyperledger-labs/fabric-token-sdk/token/driver/mock"
	benchmark2 "github.com/hyperledger-labs/fabric-token-sdk/token/services/benchmark"
	"github.com/hyperledger-labs/fabric-token-sdk/token/services/identity"
	"github.com/hyperledger-labs/fabric-token-sdk/token/services/interop/encoding"
	"github.com/hyperledger-labs/fabric-token-sdk/token/services/interop/htlc"
	"github.com/hyperledger-labs/fabric-token-sdk/token/services/logging"
	"github.com/stretchr/testify/require"
)
var (
	testUseCaseExtra = &benchmark2.Case{
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
	return nil, nil
func (m *mockSignatureProvider) Signatures() [][]byte {
	return nil
func TestIssueValidateErrors(t *testing.T) {
	configurations, err := benchmark.NewSetupConfigurations("./../testdata", []uint64{testUseCaseExtra.Bits}, []math.CurveID{testUseCaseExtra.CurveID})
	require.NoError(t, err)
	env, err := testing2.NewEnv(testUseCaseExtra, configurations)
	issueAction := &issue.Action{}
	err = issueAction.Deserialize(env.TRWithIssue.Issues[0])
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
	require.Contains(t, err.Error(), "issuer is not authorized")
	ctx.PP.IssuerIDs = oldIssuerIDs
	// Case 4: Deserializer.GetIssuerVerifier fails
	ctx = newCtx()
	mockDes := &mock3.Deserializer{}
	mockDes.GetIssuerVerifierReturns(nil, errors.New("failed getting verifier"))
	ctx.Deserializer = mockDes
	require.Contains(t, err.Error(), "failed getting verifier for issuer")
	// Case 5: SignatureProvider.HasBeenSignedBy fails
	mockSigProv := &mockSignatureProvider{
		HasBeenSignedByFunc: func(ctx context.Context, id driver.Identity, verifier driver.Verifier) ([]byte, error) {
			return nil, errors.New("signature verification failed")
		},
	ctx.SignatureProvider = mockSigProv
	require.Contains(t, err.Error(), "failed verifying signature")
	// Case 6: action.Validate() fails again (nil output)
	issueAction.Outputs = []*token.Token{nil}
	// Case 7: Verify failure
	issueAction.Proof = []byte("invalid-proof")
	// Reset to valid state
	// Case 8: Success with issuers list
	ctx.PP.IssuerIDs = []driver.Identity{issueAction.Issuer}
func TestTransferSignatureValidateErrors(t *testing.T) {
	pp, err := v1.Setup(32, []byte("idemix"), math.BLS12_381_BBS_GURVY)
	ctx := &validator.Context{
		Logger:         logging.MustGetLogger(),
		TransferAction: &transfer.Action{},
		PP:             pp,
	// Case: len(ctx.TransferAction.Inputs) == 0
	err = validator.TransferSignatureValidate(context.Background(), ctx)
	require.Contains(t, err.Error(), "invalid number of token inputs, expected at least 1")
	// Case: Deserializer.GetOwnerVerifier fails
	ctx.TransferAction.Inputs = []*transfer.ActionInput{{}}
	ctx.TransferAction.Inputs[0].Token = &token.Token{Owner: []byte("owner1")}
	mockDes.GetOwnerVerifierReturns(nil, errors.New("deserialization failed"))
	require.Contains(t, err.Error(), "failed deserializing owner")
	// Case: SignatureProvider.HasBeenSignedBy fails for owner
	mockDes.GetOwnerVerifierReturns(&mock3.Verifier{}, nil)
	require.Contains(t, err.Error(), "failed signature verification")
	// Case: Redeem action, issuer missing
	mockSigProv.HasBeenSignedByFunc = func(ctx context.Context, id driver.Identity, verifier driver.Verifier) ([]byte, error) {
		return []byte("sig"), nil
	ctx.TransferAction.Outputs = []*token.Token{{Owner: nil, Data: pp.PedersenGenerators[0]}}
	ctx.PP.IssuerIDs = []driver.Identity{[]byte("issuer1")}
	require.Contains(t, err.Error(), "On Redeem action, must have at least one issuer")
	// Case: Redeem action, issuer verifier fails
	ctx.TransferAction.Issuer = []byte("issuer1")
	mockDes.GetIssuerVerifierReturns(nil, errors.New("failed getting issuer verifier"))
	require.Contains(t, err.Error(), "failed deserializing issuer")
	// Case: Redeem action, issuer signature fails
	mockDes.GetIssuerVerifierReturns(&mock3.Verifier{}, nil)
	mockSigProv = &mockSignatureProvider{
			if bytes.Equal(id, []byte("issuer1")) {
				return nil, errors.New("issuer signature failed")
			}
			return []byte("sig"), nil
func TestTransferUpgradeWitnessValidateErrors(t *testing.T) {
	action := &transfer.Action{
		Inputs: []*transfer.ActionInput{
			{
				UpgradeWitness: &token.UpgradeWitness{},
		TransferAction: action,
	// Case: witness.FabToken is nil
	err = validator.TransferUpgradeWitnessValidate(context.Background(), ctx)
	require.Contains(t, err.Error(), "fabtoken token not found in witness")
	// Case: token2.ToQuantity fails
	action.Inputs[0].UpgradeWitness.FabToken = &fv1.Output{
		Quantity: "invalid",
	require.Contains(t, err.Error(), "failed to unmarshal quantity")
	// Case: recomputed commitment does not match
	curve := math.Curves[pp.Curve]
	rand, _ := curve.Rand()
	bf := curve.NewRandomZr(rand)
		Quantity: "0x10",
		Type:     "ABC",
		Owner:    []byte("owner1"),
	action.Inputs[0].UpgradeWitness.BlindingFactor = bf
	action.Inputs[0].Token = &token.Token{
		Data:  pp.PedersenGenerators[0], // wrong commitment
		Owner: []byte("owner1"),
	require.Contains(t, err.Error(), "recomputed commitment does not match")
	// Case: owners do not correspond
	toks, _, err := token.GetTokensWithWitnessAndBF([]uint64{16}, []*math.Zr{bf}, "ABC", pp.PedersenGenerators, curve)
	action.Inputs[0].Token.Data = toks[0]
	action.Inputs[0].Token.Owner = []byte("owner2") // different owner
	require.Contains(t, err.Error(), "owners do not correspond")
func TestTransferZKProofValidateErrors(t *testing.T) {
		PP: pp,
		InputTokens: []*token.Token{
			{Data: pp.PedersenGenerators[0]},
		TransferAction: &transfer.Action{
			Outputs: []*token.Token{
				{Data: pp.PedersenGenerators[1]},
			Proof: []byte("invalid-proof"),
	err = validator.TransferZKProofValidate(context.Background(), ctx)
func TestTransferHTLCValidateErrors(t *testing.T) {
	validSenderID, _ := identity.WrapWithType(x509.IdentityType, []byte("owner1"))
	validRecipientID, _ := identity.WrapWithType(x509.IdentityType, []byte("recipient"))
			Logger: logging.MustGetLogger(),
			InputTokens: []*token.Token{
				{Owner: validSenderID},
			TransferAction: &transfer.Action{
				Inputs: []*transfer.ActionInput{{}},
			Signatures:      [][]byte{[]byte("sig")},
			MetadataCounter: make(map[string]int),
	// Case: identity.UnmarshalTypedIdentity(in.Owner) fails
	ctx.InputTokens[0].Owner = []byte("invalid-identity")
	err := validator.TransferHTLCValidate(context.Background(), ctx)
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
	scriptBytes, _ := json.Marshal(script)
	lockID, _ := identity.WrapWithType(htlc.ScriptType, scriptBytes)
	// Case: HTLC script, but invalid number of inputs/outputs
	ctx.InputTokens[0].Owner = lockID
	ctx.TransferAction.Inputs = []*transfer.ActionInput{{}, {}} // 2 inputs
	ctx.InputTokens = append(ctx.InputTokens, &token.Token{Owner: validSenderID})
	ctx.Signatures = append(ctx.Signatures, []byte("sig2"))
	err = validator.TransferHTLCValidate(context.Background(), ctx)
	require.Contains(t, err.Error(), "an htlc script only transfers the ownership of a token")
	// Case: output owner type script invalid
	ctx.InputTokens[0].Owner = validSenderID
	ctx.TransferAction.Outputs = []*token.Token{
		{Owner: lockID, Data: &math.G1{}},
	invalidScriptID, _ := identity.WrapWithType(htlc.ScriptType, []byte("{}"))
	ctx.TransferAction.Outputs[0].Owner = invalidScriptID
	require.Contains(t, err.Error(), "htlc script invalid")
	// Case: successful HTLC lock
	ctx.TransferAction.Outputs = []*token.Token{{Owner: lockID, Data: &math.G1{}}}
	ctx.TransferAction.Metadata = map[string][]byte{
		htlc.LockKey(script.HashInfo.Hash): htlc.LockValue(script.HashInfo.Hash),
	// Case: successful HTLC claim
	scriptActual := &htlc.Script{}
	require.NoError(t, json.Unmarshal(scriptBytes, scriptActual))
	ctx.TransferAction.Outputs = []*token.Token{{Owner: scriptActual.Recipient.Bytes(), Data: &math.G1{}}}
	preimage := []byte("preimage")
	image, _ := script.HashInfo.Image(preimage)
	claimSigBytes, _ := json.Marshal(&htlc.ClaimSignature{Preimage: preimage, RecipientSignature: []byte("sig")})
	ctx.Signatures = [][]byte{claimSigBytes}
	ctx.TransferAction.Metadata = map[string][]byte{htlc.ClaimKey(image): preimage}
	// Case: HTLC reclaim
	script.Deadline = time.Now().Add(-100 * time.Hour) // expired
	scriptBytes, _ = json.Marshal(script)
	lockID, _ = identity.WrapWithType(htlc.ScriptType, scriptBytes)
	ctx.TransferAction.Outputs = []*token.Token{{Owner: scriptActual.Sender.Bytes(), Data: &math.G1{}}}
	// Case: MetadataLockKeyCheck fails
	script.Deadline = time.Now().Add(1 * time.Hour) // not expired
	ctx.TransferAction.Metadata = nil
	require.Contains(t, err.Error(), "failed to check htlc metadata")
	// Case: invalid output type (nil)
	ctx.InputTokens[0].Owner = lockID // to enter first loop branch and then panic if no nil check
	ctx.TransferAction.Outputs = []*token.Token{nil}
	require.Contains(t, err.Error(), "invalid transfer action: an htlc script only transfers the ownership of a token, output not found")
	// Case: output is redeem
	ctx.TransferAction.Outputs = []*token.Token{{Owner: nil}}
	// Case: output owner identity unmarshal fails
	ctx.TransferAction.Outputs = []*token.Token{{Owner: []byte("invalid")}}
	// Case: script.FromBytes failure
	invalidJSONID, _ := identity.WrapWithType(htlc.ScriptType, []byte("invalid"))
	ctx.TransferAction.Outputs = []*token.Token{{Owner: invalidJSONID}}
func TestDeserializeActionsErrors(t *testing.T) {
	ad := &validator.ActionDeserializer{}
	tr := &driver.TokenRequest{
		Issues: [][]byte{[]byte("invalid")},
	_, _, err := ad.DeserializeActions(tr)
	tr = &driver.TokenRequest{
		Transfers: [][]byte{[]byte("invalid")},
	_, _, err = ad.DeserializeActions(tr)
