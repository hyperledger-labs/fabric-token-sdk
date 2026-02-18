/*
Copyright IBM Corp. All Rights Reserved.

SPDX-License-Identifier: Apache-2.0
*/

package audit_test

import (
	"os"
	"testing"

	"github.com/IBM/idemix/bccsp/types"
	math "github.com/IBM/mathlib"
	"github.com/hyperledger-labs/fabric-smart-client/pkg/utils/errors"
	"github.com/hyperledger-labs/fabric-token-sdk/token/core/zkatdlog/nogh/v1/audit"
	"github.com/hyperledger-labs/fabric-token-sdk/token/core/zkatdlog/nogh/v1/audit/mock"
	"github.com/hyperledger-labs/fabric-token-sdk/token/core/zkatdlog/nogh/v1/issue"
	v1 "github.com/hyperledger-labs/fabric-token-sdk/token/core/zkatdlog/nogh/v1/setup"
	"github.com/hyperledger-labs/fabric-token-sdk/token/core/zkatdlog/nogh/v1/token"
	"github.com/hyperledger-labs/fabric-token-sdk/token/core/zkatdlog/nogh/v1/transfer"
	"github.com/hyperledger-labs/fabric-token-sdk/token/driver"
	"github.com/hyperledger-labs/fabric-token-sdk/token/services/identity"
	"github.com/hyperledger-labs/fabric-token-sdk/token/services/identity/deserializer"
	"github.com/hyperledger-labs/fabric-token-sdk/token/services/identity/idemix"
	"github.com/hyperledger-labs/fabric-token-sdk/token/services/identity/idemix/crypto"
	"github.com/hyperledger-labs/fabric-token-sdk/token/services/logging"
	kvs2 "github.com/hyperledger-labs/fabric-token-sdk/token/services/storage/db/kvs"
	"github.com/hyperledger-labs/fabric-token-sdk/token/services/utils/slices"
	token3 "github.com/hyperledger-labs/fabric-token-sdk/token/token"
	"github.com/stretchr/testify/require"
	"go.opentelemetry.io/otel/trace/noop"
)

func TestAuditor(t *testing.T) {
	setup := func(t *testing.T) (*mock.SigningIdentity, *v1.PublicParams, *audit.Auditor) {
		t.Helper()
		fakeSigningIdentity := &mock.SigningIdentity{}
		ipk, err := os.ReadFile("./testdata/bls12_381_bbs/idemix/msp/IssuerPublicKey")
		require.NoError(t, err)
		pp, err := v1.Setup(32, ipk, math.BLS12_381_BBS_GURVY)
		require.NoError(t, err)
		idemixDes, err := idemix.NewDeserializer(slices.GetUnique(pp.IdemixIssuerPublicKeys).PublicKey, math.BLS12_381_BBS_GURVY)
		require.NoError(t, err)
		des := deserializer.NewTypedVerifierDeserializerMultiplex()
		des.AddTypedVerifierDeserializer(idemix.IdentityType, deserializer.NewTypedIdentityVerifierDeserializer(idemixDes, idemixDes))
		auditor := audit.NewAuditor(
			logging.MustGetLogger(),
			&noop.Tracer{},
			des,
			pp.PedersenGenerators,
			math.Curves[pp.Curve],
		)
		fakeSigningIdentity.SignReturns([]byte("auditor-signature"), nil)

		return fakeSigningIdentity, pp, auditor
	}

	t.Run("audit information is computed correctly", func(t *testing.T) {
		_, pp, auditor := setup(t)
		transfer, metadata, tokens := createTransfer(t, pp)
		raw, err := transfer.Serialize()
		require.NoError(t, err)
		err = auditor.Check(t.Context(), &driver.TokenRequest{Transfers: [][]byte{raw}}, &driver.TokenRequestMetadata{Transfers: []*driver.TransferMetadata{metadata}}, tokens, "1")
		require.NoError(t, err)
	})

	t.Run("token info does not match output", func(t *testing.T) {
		fakeSigningIdentity, pp, auditor := setup(t)
		transfer, metadata, tokens := createTransferWithBogusOutput(t, pp)
		raw, err := transfer.Serialize()
		require.NoError(t, err)
		err = auditor.Check(
			t.Context(),
			&driver.TokenRequest{Transfers: [][]byte{raw}},
			&driver.TokenRequestMetadata{Transfers: []*driver.TransferMetadata{metadata}},
			tokens,
			"1",
		)
		require.Error(t, err)
		require.Equal(t, 0, fakeSigningIdentity.SignCallCount())
	})

	t.Run("sender audit info does not match input", func(t *testing.T) {
		fakeSigningIdentity, pp, auditor := setup(t)
		transfer, metadata, tokens := createTransfer(t, pp)
		// test idemix info
		_, auditinfo := getIdemixInfo(t, "./testdata/bls12_381_bbs/idemix")
		raw, err := auditinfo.Bytes()
		require.NoError(t, err)
		metadata.Inputs[0].Senders[0].AuditInfo = raw
		raw, err = transfer.Serialize()
		require.NoError(t, err)
		err = auditor.Check(t.Context(), &driver.TokenRequest{Transfers: [][]byte{raw}}, &driver.TokenRequestMetadata{Transfers: []*driver.TransferMetadata{metadata}}, tokens, "1")
		require.Error(t, err)
		require.Contains(t, err.Error(), "owner at index [0] does not match the provided opening")
		require.NotContains(t, err.Error(), "attribute mistmatch")
		require.Equal(t, 0, fakeSigningIdentity.SignCallCount())
	})

	t.Run("recipient audit info does not match output", func(t *testing.T) {
		fakeSigningIdentity, pp, auditor := setup(t)
		transfer, metadata, tokens := createTransfer(t, pp)
		// test idemix info
		_, auditinfo := getIdemixInfo(t, "./testdata/bls12_381_bbs/idemix")
		raw, err := auditinfo.Bytes()
		require.NoError(t, err)
		metadata.Outputs[0].OutputAuditInfo = raw
		raw, err = transfer.Serialize()
		require.NoError(t, err)
		err = auditor.Check(t.Context(), &driver.TokenRequest{Transfers: [][]byte{raw}}, &driver.TokenRequestMetadata{Transfers: []*driver.TransferMetadata{metadata}}, tokens, "1")
		require.Error(t, err)
		require.Contains(t, err.Error(), "owner at index [0] does not match the provided opening")
		require.Contains(t, err.Error(), "does not match the provided opening")
		require.Equal(t, 0, fakeSigningIdentity.SignCallCount())
	})

	t.Run("audit an issue", func(t *testing.T) {
		_, pp, auditor := setup(t)
		ia, metadata := createIssue(t, pp)
		raw, err := ia.Serialize()
		require.NoError(t, err)
		err = auditor.Check(t.Context(), &driver.TokenRequest{Issues: [][]byte{raw}}, &driver.TokenRequestMetadata{Issues: []*driver.IssueMetadata{metadata}}, nil, "1")
		require.NoError(t, err)
	})
}

func TestAuditor_Errors(t *testing.T) {
	setup := func(t *testing.T) (*mock.SigningIdentity, *v1.PublicParams, *audit.Auditor) {
		t.Helper()
		fakeSigningIdentity := &mock.SigningIdentity{}
		ipk, err := os.ReadFile("./testdata/bls12_381_bbs/idemix/msp/IssuerPublicKey")
		require.NoError(t, err)
		pp, err := v1.Setup(32, ipk, math.BLS12_381_BBS_GURVY)
		require.NoError(t, err)
		idemixDes, err := idemix.NewDeserializer(slices.GetUnique(pp.IdemixIssuerPublicKeys).PublicKey, math.BLS12_381_BBS_GURVY)
		require.NoError(t, err)
		des := deserializer.NewTypedVerifierDeserializerMultiplex()
		des.AddTypedVerifierDeserializer(idemix.IdentityType, deserializer.NewTypedIdentityVerifierDeserializer(idemixDes, idemixDes))
		auditor := audit.NewAuditor(
			logging.MustGetLogger(),
			&noop.Tracer{},
			des,
			pp.PedersenGenerators,
			math.Curves[pp.Curve],
		)
		fakeSigningIdentity.SignReturns([]byte("auditor-signature"), nil)

		return fakeSigningIdentity, pp, auditor
	}

	t.Run("GetAuditInfoForIssues length mismatch", func(t *testing.T) {
		_, _, auditor := setup(t)
		_, _, err := auditor.GetAuditInfoForIssues([][]byte{{1}}, nil)
		require.Error(t, err)
		require.Contains(t, err.Error(), "number of issues does not match number of provided metadata")
	})

	t.Run("GetAuditInfoForTransfers length mismatch", func(t *testing.T) {
		_, _, auditor := setup(t)
		_, _, err := auditor.GetAuditInfoForTransfers([][]byte{{1}}, nil, nil)
		require.Error(t, err)
		require.Contains(t, err.Error(), "number of transfers does not match the number of provided metadata")

		_, _, err = auditor.GetAuditInfoForTransfers([][]byte{{1}}, []*driver.TransferMetadata{{}}, nil)
		require.Error(t, err)
		require.Contains(t, err.Error(), "number of inputs does not match the number of provided metadata")
	})

	t.Run("InspectIdentity error cases", func(t *testing.T) {
		_, _, auditor := setup(t)
		err := auditor.InspectIdentity(t.Context(), nil, &audit.InspectableIdentity{Identity: nil}, 0)
		require.Error(t, err)
		require.Contains(t, err.Error(), "identity at index [0] is nil")

		err = auditor.InspectIdentity(t.Context(), nil, &audit.InspectableIdentity{Identity: []byte("id")}, 0)
		require.Error(t, err)
		require.Contains(t, err.Error(), "audit info is nil")

		err = auditor.InspectIdentity(t.Context(), nil, &audit.InspectableIdentity{
			Identity:         []byte("id1"),
			IdentityFromMeta: []byte("id2"),
			AuditInfo:        []byte("ai"),
		}, 0)
		require.Error(t, err)
		require.Contains(t, err.Error(), "identity does not match the identity form metadata")
	})

	t.Run("InspectIdentity MatchIdentity error", func(t *testing.T) {
		_, _, auditor := setup(t)
		fakeMatcher := &mock.InfoMatcher{}
		fakeMatcher.MatchIdentityReturns(errors.New("match failed"))
		err := auditor.InspectIdentity(t.Context(), fakeMatcher, &audit.InspectableIdentity{
			Identity:  []byte("id"),
			AuditInfo: []byte("ai"),
		}, 0)
		require.Error(t, err)
		require.Contains(t, err.Error(), "match failed")
	})

	t.Run("InspectOutput error cases", func(t *testing.T) {
		_, _, auditor := setup(t)
		err := auditor.InspectOutput(t.Context(), nil, 0)
		require.Error(t, err)
		require.Contains(t, err.Error(), "invalid output at index [0]")

		err = auditor.InspectOutput(t.Context(), &audit.InspectableToken{}, 0)
		require.Error(t, err)
		require.Contains(t, err.Error(), "invalid output at index [0]")
	})

	t.Run("InspectInputs error cases", func(t *testing.T) {
		_, _, auditor := setup(t)
		err := auditor.InspectInputs(t.Context(), []*audit.InspectableToken{nil})
		require.Error(t, err)
		require.Contains(t, err.Error(), "invalid input at index [0]")
	})
}

func TestAuditor_GetAuditInfo_Errors(t *testing.T) {
	setup := func(t *testing.T) (*mock.SigningIdentity, *v1.PublicParams, *audit.Auditor) {
		t.Helper()
		fakeSigningIdentity := &mock.SigningIdentity{}
		ipk, err := os.ReadFile("./testdata/bls12_381_bbs/idemix/msp/IssuerPublicKey")
		require.NoError(t, err)
		pp, err := v1.Setup(32, ipk, math.BLS12_381_BBS_GURVY)
		require.NoError(t, err)
		idemixDes, err := idemix.NewDeserializer(slices.GetUnique(pp.IdemixIssuerPublicKeys).PublicKey, math.BLS12_381_BBS_GURVY)
		require.NoError(t, err)
		des := deserializer.NewTypedVerifierDeserializerMultiplex()
		des.AddTypedVerifierDeserializer(idemix.IdentityType, deserializer.NewTypedIdentityVerifierDeserializer(idemixDes, idemixDes))
		auditor := audit.NewAuditor(
			logging.MustGetLogger(),
			&noop.Tracer{},
			des,
			pp.PedersenGenerators,
			math.Curves[pp.Curve],
		)
		fakeSigningIdentity.SignReturns([]byte("auditor-signature"), nil)

		return fakeSigningIdentity, pp, auditor
	}

	t.Run("GetAuditInfoForIssues deserialization error", func(t *testing.T) {
		_, _, auditor := setup(t)
		_, _, err := auditor.GetAuditInfoForIssues([][]byte{{1, 2, 3}}, []*driver.IssueMetadata{{}})
		require.Error(t, err)
		require.Contains(t, err.Error(), "failed to deserialize issue action")
	})

	t.Run("GetAuditInfoForIssues output count mismatch", func(t *testing.T) {
		_, pp, auditor := setup(t)
		ia, _ := createIssue(t, pp)
		raw, _ := ia.Serialize()
		_, _, err := auditor.GetAuditInfoForIssues([][]byte{raw}, []*driver.IssueMetadata{{Outputs: []*driver.IssueOutputMetadata{{}, {}}}})
		require.Error(t, err)
		require.Contains(t, err.Error(), "number of output does not match number of provided metadata")
	})

	t.Run("GetAuditInfoForIssues nil metadata output", func(t *testing.T) {
		_, pp, auditor := setup(t)
		ia, _ := createIssue(t, pp)
		raw, _ := ia.Serialize()
		_, _, err := auditor.GetAuditInfoForIssues([][]byte{raw}, []*driver.IssueMetadata{{Outputs: []*driver.IssueOutputMetadata{nil}}})
		require.Error(t, err)
		require.Contains(t, err.Error(), "output at index [0] is nil")
	})

	t.Run("GetAuditInfoForIssues metadata deserialization error", func(t *testing.T) {
		_, pp, auditor := setup(t)
		ia, _ := createIssue(t, pp)
		raw, _ := ia.Serialize()
		_, _, err := auditor.GetAuditInfoForIssues([][]byte{raw}, []*driver.IssueMetadata{{Outputs: []*driver.IssueOutputMetadata{{OutputMetadata: []byte{1, 2, 3}}}}})
		require.Error(t, err)
		require.Contains(t, err.Error(), "failed deserializing metadata")
	})

	t.Run("GetAuditInfoForIssues nil issue output", func(t *testing.T) {
		_, pp, auditor := setup(t)
		ia, meta := createIssue(t, pp)
		ia.Outputs[0] = nil
		raw, _ := ia.Serialize()
		_, _, err := auditor.GetAuditInfoForIssues([][]byte{raw}, []*driver.IssueMetadata{meta})
		require.Error(t, err)
		require.Contains(t, err.Error(), "output token at index [0] is nil")
	})

	t.Run("GetAuditInfoForIssues issue redeem error", func(t *testing.T) {
		_, pp, auditor := setup(t)
		ia, meta := createIssue(t, pp)
		ia.Outputs[0].Owner = nil // redeem
		raw, _ := ia.Serialize()
		_, _, err := auditor.GetAuditInfoForIssues([][]byte{raw}, []*driver.IssueMetadata{meta})
		require.Error(t, err)
		require.Contains(t, err.Error(), "issue cannot redeem tokens")
	})

	t.Run("GetAuditInfoForIssues no receivers", func(t *testing.T) {
		_, pp, auditor := setup(t)
		ia, meta := createIssue(t, pp)
		meta.Outputs[0].Receivers = nil
		raw, _ := ia.Serialize()
		_, _, err := auditor.GetAuditInfoForIssues([][]byte{raw}, []*driver.IssueMetadata{meta})
		require.Error(t, err)
		require.Contains(t, err.Error(), "issue must have at least one receiver")
	})

	t.Run("GetAuditInfoForTransfers nil input token", func(t *testing.T) {
		_, _, auditor := setup(t)
		_, _, err := auditor.GetAuditInfoForTransfers([][]byte{{}}, []*driver.TransferMetadata{{Inputs: []*driver.TransferInputMetadata{{}}}}, [][]*token.Token{{nil}})
		require.Error(t, err)
		require.Contains(t, err.Error(), "input[0][0] is nil")
	})

	t.Run("GetAuditInfoForTransfers invalid input metadata", func(t *testing.T) {
		_, _, auditor := setup(t)
		_, _, err := auditor.GetAuditInfoForTransfers([][]byte{{}}, []*driver.TransferMetadata{{Inputs: []*driver.TransferInputMetadata{nil}}}, [][]*token.Token{{&token.Token{}}})
		require.Error(t, err)
		require.Contains(t, err.Error(), "invalid metadata for input[0][0]")
	})

	t.Run("GetAuditInfoForTransfers transfer deserialization error", func(t *testing.T) {
		_, _, auditor := setup(t)
		inputs := []*driver.TransferInputMetadata{{Senders: []*driver.AuditableIdentity{{AuditInfo: []byte{1}}}}}
		_, _, err := auditor.GetAuditInfoForTransfers([][]byte{{1, 2, 3}}, []*driver.TransferMetadata{{Inputs: inputs}}, [][]*token.Token{{&token.Token{}}})
		require.Error(t, err)
		require.Contains(t, err.Error(), "failed to deserialize transfer action")
	})

	t.Run("GetAuditInfoForTransfers output count mismatch", func(t *testing.T) {
		_, pp, auditor := setup(t)
		transfer, meta, _ := createTransfer(t, pp)
		raw, _ := transfer.Serialize()
		meta.Outputs = meta.Outputs[:len(meta.Outputs)-1]
		_, _, err := auditor.GetAuditInfoForTransfers([][]byte{raw}, []*driver.TransferMetadata{meta}, [][]*token.Token{{&token.Token{}, &token.Token{}}})
		require.Error(t, err)
		require.Contains(t, err.Error(), "number of outputs does not match the number of output metadata")
	})

	t.Run("GetAuditInfoForTransfers nil output token", func(t *testing.T) {
		_, pp, auditor := setup(t)
		transfer, meta, _ := createTransfer(t, pp)
		transfer.Outputs[0] = nil
		raw, _ := transfer.Serialize()
		_, _, err := auditor.GetAuditInfoForTransfers([][]byte{raw}, []*driver.TransferMetadata{meta}, [][]*token.Token{{&token.Token{}, &token.Token{}}})
		require.Error(t, err)
		require.Contains(t, err.Error(), "output token at index [0] is nil")
	})

	t.Run("GetAuditInfoForTransfers nil output metadata", func(t *testing.T) {
		_, pp, auditor := setup(t)
		transfer, meta, _ := createTransfer(t, pp)
		meta.Outputs[0] = nil
		raw, _ := transfer.Serialize()
		_, _, err := auditor.GetAuditInfoForTransfers([][]byte{raw}, []*driver.TransferMetadata{meta}, [][]*token.Token{{&token.Token{}, &token.Token{}}})
		require.Error(t, err)
		require.Contains(t, err.Error(), "metadata for output token at index [0] is nil")
	})

	t.Run("GetAuditInfoForTransfers output metadata deserialization error", func(t *testing.T) {
		_, pp, auditor := setup(t)
		transfer, meta, _ := createTransfer(t, pp)
		meta.Outputs[0].OutputMetadata = []byte{1, 2, 3}
		raw, _ := transfer.Serialize()
		_, _, err := auditor.GetAuditInfoForTransfers([][]byte{raw}, []*driver.TransferMetadata{meta}, [][]*token.Token{{&token.Token{}, &token.Token{}}})
		require.Error(t, err)
		require.Contains(t, err.Error(), "failed deserializing metadata")
	})
}

func TestAuditor_Check_Errors(t *testing.T) {
	setup := func(t *testing.T) (*mock.SigningIdentity, *v1.PublicParams, *audit.Auditor) {
		t.Helper()
		fakeSigningIdentity := &mock.SigningIdentity{}
		ipk, err := os.ReadFile("./testdata/bls12_381_bbs/idemix/msp/IssuerPublicKey")
		require.NoError(t, err)
		pp, err := v1.Setup(32, ipk, math.BLS12_381_BBS_GURVY)
		require.NoError(t, err)
		idemixDes, err := idemix.NewDeserializer(slices.GetUnique(pp.IdemixIssuerPublicKeys).PublicKey, math.BLS12_381_BBS_GURVY)
		require.NoError(t, err)
		des := deserializer.NewTypedVerifierDeserializerMultiplex()
		des.AddTypedVerifierDeserializer(idemix.IdentityType, deserializer.NewTypedIdentityVerifierDeserializer(idemixDes, idemixDes))
		auditor := audit.NewAuditor(
			logging.MustGetLogger(),
			&noop.Tracer{},
			des,
			pp.PedersenGenerators,
			math.Curves[pp.Curve],
		)
		fakeSigningIdentity.SignReturns([]byte("auditor-signature"), nil)

		return fakeSigningIdentity, pp, auditor
	}

	t.Run("Check issue audit info error", func(t *testing.T) {
		_, _, auditor := setup(t)
		err := auditor.Check(t.Context(), &driver.TokenRequest{Issues: [][]byte{{1, 2, 3}}}, &driver.TokenRequestMetadata{Issues: []*driver.IssueMetadata{{}}}, nil, "1")
		require.Error(t, err)
		require.Contains(t, err.Error(), "failed getting audit info for issues")
	})

	t.Run("Check issue request validation error", func(t *testing.T) {
		_, pp, auditor := setup(t)
		ia, meta := createIssue(t, pp)
		ia.Outputs[0].Data = pp.PedersenGenerators[0] // wrong data
		raw, _ := ia.Serialize()
		err := auditor.Check(t.Context(), &driver.TokenRequest{Issues: [][]byte{raw}}, &driver.TokenRequestMetadata{Issues: []*driver.IssueMetadata{meta}}, nil, "1")
		require.Error(t, err)
		require.Contains(t, err.Error(), "failed checking issues")
	})

	t.Run("Check issue identity validation error", func(t *testing.T) {
		_, pp, auditor := setup(t)
		ia, meta := createIssue(t, pp)
		meta.Issuer.AuditInfo = []byte("wrong")
		raw, _ := ia.Serialize()
		err := auditor.Check(t.Context(), &driver.TokenRequest{Issues: [][]byte{raw}}, &driver.TokenRequestMetadata{Issues: []*driver.IssueMetadata{meta}}, nil, "1")
		require.Error(t, err)
		require.Contains(t, err.Error(), "failed checking identity for issue")
	})

	t.Run("Check transfer audit info error", func(t *testing.T) {
		_, _, auditor := setup(t)
		err := auditor.Check(t.Context(), &driver.TokenRequest{Transfers: [][]byte{{1, 2, 3}}}, &driver.TokenRequestMetadata{Transfers: []*driver.TransferMetadata{{}}}, nil, "1")
		require.Error(t, err)
		require.Contains(t, err.Error(), "failed getting audit info for transfers")
	})
}

func createTransfer(t *testing.T, pp *v1.PublicParams) (*transfer.Action, *driver.TransferMetadata, [][]*token.Token) {
	t.Helper()
	id, auditInfo := getIdemixInfo(t, "./testdata/bls12_381_bbs/idemix")
	transfer, meta, inputs := prepareTransfer(t, pp, id)

	auditInfoRaw, err := auditInfo.Bytes()
	require.NoError(t, err)

	metadata := &driver.TransferMetadata{}
	for range len(transfer.Inputs) {
		metadata.Inputs = append(metadata.Inputs, &driver.TransferInputMetadata{
			TokenID: nil,
			Senders: []*driver.AuditableIdentity{
				{
					Identity:  nil,
					AuditInfo: auditInfoRaw,
				},
			},
		})
	}

	for i := range len(transfer.Outputs) {
		marshalledMeta, err := meta[i].Serialize()
		require.NoError(t, err)
		metadata.Outputs = append(metadata.Outputs, &driver.TransferOutputMetadata{
			OutputMetadata:  marshalledMeta,
			OutputAuditInfo: auditInfoRaw,
			Receivers: []*driver.AuditableIdentity{
				{
					Identity:  nil,
					AuditInfo: auditInfoRaw,
				},
			},
		})
	}

	tokns := make([][]*token.Token, 1)
	tokns[0] = append(tokns[0], inputs...)

	return transfer, metadata, tokns
}

func createTransferWithBogusOutput(t *testing.T, pp *v1.PublicParams) (*transfer.Action, *driver.TransferMetadata, [][]*token.Token) {
	t.Helper()
	id, auditInfo := getIdemixInfo(t, "./testdata/bls12_381_bbs/idemix")
	transfer, inf, inputs := prepareTransfer(t, pp, id)

	c := math.Curves[pp.Curve]
	inf[0].Value = c.NewZrFromInt(15)
	auditInfoRaw, err := auditInfo.Bytes()
	require.NoError(t, err)

	metadata := &driver.TransferMetadata{}
	for range len(transfer.Inputs) {
		metadata.Inputs = append(metadata.Inputs, &driver.TransferInputMetadata{
			TokenID: nil,
			Senders: []*driver.AuditableIdentity{
				{
					Identity:  nil,
					AuditInfo: auditInfoRaw,
				},
			},
		})
	}

	for i := range len(transfer.Outputs) {
		marshalledMeta, err := inf[i].Serialize()
		require.NoError(t, err)
		metadata.Outputs = append(metadata.Outputs, &driver.TransferOutputMetadata{
			OutputMetadata:  marshalledMeta,
			OutputAuditInfo: auditInfoRaw,
			Receivers: []*driver.AuditableIdentity{
				{
					Identity:  nil,
					AuditInfo: auditInfoRaw,
				},
			},
		})
	}

	tokns := make([][]*token.Token, 1)
	tokns[0] = append(tokns[0], inputs...)

	return transfer, metadata, tokns
}

func getIdemixInfo(t *testing.T, dir string) (driver.Identity, *crypto.AuditInfo) {
	t.Helper()
	backend, err := kvs2.NewInMemory()
	require.NoError(t, err)
	config, err := crypto.NewConfig(dir)
	require.NoError(t, err)
	curveID := math.BLS12_381_BBS_GURVY

	keyStore, err := crypto.NewKeyStore(curveID, kvs2.Keystore(backend))
	require.NoError(t, err)
	cryptoProvider, err := crypto.NewBCCSP(keyStore, curveID)
	require.NoError(t, err)
	p, err := idemix.NewKeyManager(config, types.EidNymRhNym, cryptoProvider)
	require.NoError(t, err)
	require.NotNil(t, p)

	identityDescriptor, err := p.Identity(t.Context(), nil)
	require.NoError(t, err)
	id := identityDescriptor.Identity
	audit := identityDescriptor.AuditInfo
	require.NotNil(t, id)
	require.NotNil(t, audit)

	auditInfo, err := p.DeserializeAuditInfo(t.Context(), audit)
	require.NoError(t, err)
	err = auditInfo.Match(t.Context(), id)
	require.NoError(t, err)

	id, err = identity.WrapWithType(idemix.IdentityType, id)
	require.NoError(t, err)

	return id, auditInfo
}

func createInputs(t *testing.T, pp *v1.PublicParams, id driver.Identity) ([]*token.Token, []*token.Metadata) {
	t.Helper()
	c := math.Curves[pp.Curve]
	inputs := make([]*token.Token, 2)
	infos := make([]*token.Metadata, 2)
	values := []*math.Zr{c.NewZrFromInt(25), c.NewZrFromInt(35)}
	rand, err := c.Rand()
	require.NoError(t, err)
	ttype := c.HashToZr([]byte("ABC"))

	for i := 0; i < len(inputs); i++ {
		infos[i] = &token.Metadata{}
		infos[i].BlindingFactor = c.NewRandomZr(rand)
		infos[i].Value = values[i]
		infos[i].Type = "ABC"
		inputs[i] = &token.Token{}
		inputs[i].Data = commit([]*math.Zr{ttype, values[i], infos[i].BlindingFactor}, pp.PedersenGenerators, c)
		inputs[i].Owner = id
	}

	return inputs, infos
}

func prepareTransfer(t *testing.T, pp *v1.PublicParams, id driver.Identity) (*transfer.Action, []*token.Metadata, []*token.Token) {
	t.Helper()
	inputs, tokenInfos := createInputs(t, pp, id)

	fakeSigner := &mock.SigningIdentity{}

	sender, err := transfer.NewSender([]driver.Signer{fakeSigner, fakeSigner}, inputs, []*token3.ID{{TxId: "0"}, {TxId: "1"}}, tokenInfos, pp)
	require.NoError(t, err)
	transfer, inf, err := sender.GenerateZKTransfer(t.Context(), []uint64{40, 20}, [][]byte{id, id})
	require.NoError(t, err)

	return transfer, inf, inputs
}

func createIssue(t *testing.T, pp *v1.PublicParams) (*issue.Action, *driver.IssueMetadata) {
	t.Helper()
	id, auditInfo := getIdemixInfo(t, "./testdata/bls12_381_bbs/idemix")
	c := math.Curves[pp.Curve]
	value := c.NewZrFromInt(100)
	bf := c.NewRandomZr(nil)
	ttype := token3.Type("ABC")
	com := commit([]*math.Zr{c.HashToZr([]byte(ttype)), value, bf}, pp.PedersenGenerators, c)

	ia := &issue.Action{
		Issuer:  id,
		Outputs: []*token.Token{{Owner: id, Data: com}},
	}

	meta := &token.Metadata{Type: ttype, Value: value, BlindingFactor: bf}
	metaRaw, err := meta.Serialize()
	require.NoError(t, err)

	auditInfoRaw, err := auditInfo.Bytes()
	require.NoError(t, err)

	metadata := &driver.IssueMetadata{
		Issuer: driver.AuditableIdentity{Identity: id, AuditInfo: auditInfoRaw},
		Outputs: []*driver.IssueOutputMetadata{{
			OutputMetadata: metaRaw,
			Receivers:      []*driver.AuditableIdentity{{Identity: id, AuditInfo: auditInfoRaw}},
		}},
	}

	return ia, metadata
}

func commit(vector []*math.Zr, generators []*math.G1, c *math.Curve) *math.G1 {
	com := c.NewG1()
	for i := range vector {
		com.Add(generators[i].Mul(vector[i]))
	}

	return com
}
