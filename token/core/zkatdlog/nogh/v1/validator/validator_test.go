/*
Copyright IBM Corp. All Rights Reserved.

SPDX-License-Identifier: Apache-2.0
*/

package validator_test

import (
	"bytes"
	"context"
	"crypto/ecdsa"
	"crypto/elliptic"
	"crypto/rand"
	"crypto/rsa"
	"crypto/sha256"
	"crypto/x509"
	"encoding/asn1"
	"encoding/pem"
	"fmt"
	"math/big"
	"os"
	"testing"

	"github.com/IBM/idemix/bccsp/types"
	math "github.com/IBM/mathlib"
	"github.com/hyperledger-labs/fabric-smart-client/pkg/utils/errors"
	"github.com/hyperledger-labs/fabric-token-sdk/token/core/zkatdlog/nogh/v1/audit"
	zkatdlog "github.com/hyperledger-labs/fabric-token-sdk/token/core/zkatdlog/nogh/v1/driver"
	issue2 "github.com/hyperledger-labs/fabric-token-sdk/token/core/zkatdlog/nogh/v1/issue"
	v1 "github.com/hyperledger-labs/fabric-token-sdk/token/core/zkatdlog/nogh/v1/setup"
	tokn "github.com/hyperledger-labs/fabric-token-sdk/token/core/zkatdlog/nogh/v1/token"
	"github.com/hyperledger-labs/fabric-token-sdk/token/core/zkatdlog/nogh/v1/transfer"
	enginedlog "github.com/hyperledger-labs/fabric-token-sdk/token/core/zkatdlog/nogh/v1/validator"
	"github.com/hyperledger-labs/fabric-token-sdk/token/core/zkatdlog/nogh/v1/validator/mock"
	"github.com/hyperledger-labs/fabric-token-sdk/token/driver"
	"github.com/hyperledger-labs/fabric-token-sdk/token/services/identity"
	"github.com/hyperledger-labs/fabric-token-sdk/token/services/identity/deserializer"
	idemix2 "github.com/hyperledger-labs/fabric-token-sdk/token/services/identity/idemix"
	"github.com/hyperledger-labs/fabric-token-sdk/token/services/identity/idemix/crypto"
	"github.com/hyperledger-labs/fabric-token-sdk/token/services/identity/storage/kvs"
	ix509 "github.com/hyperledger-labs/fabric-token-sdk/token/services/identity/x509"
	"github.com/hyperledger-labs/fabric-token-sdk/token/services/logging"
	utils2 "github.com/hyperledger-labs/fabric-token-sdk/token/services/utils"
	"github.com/hyperledger-labs/fabric-token-sdk/token/services/utils/slices"
	token2 "github.com/hyperledger-labs/fabric-token-sdk/token/token"
	"github.com/hyperledger/fabric-lib-go/bccsp/utils"
	"github.com/stretchr/testify/require"
	"go.opentelemetry.io/otel/trace/noop"
)

var fakeLedger *mock.Ledger

type Env struct {
	ir                *driver.TokenRequest
	engine            *enginedlog.Validator
	inputsForTransfer []*tokn.Token
	tr                *driver.TokenRequest
	inputsForRedeem   []*tokn.Token
	rr                *driver.TokenRequest
	ar                *driver.TokenRequest
	sender            *transfer.Sender
}

func NewEnv() (*Env, error) {
	var (
		engine *enginedlog.Validator
		pp     *v1.PublicParams

		inputsForRedeem   []*tokn.Token
		inputsForTransfer []*tokn.Token

		sender  *transfer.Sender
		auditor *audit.Auditor
		ipk     []byte

		ir *driver.TokenRequest // regular issue request
		rr *driver.TokenRequest // redeem request
		tr *driver.TokenRequest // transfer request
		ar *driver.TokenRequest // atomic action request
	)
	fakeLedger = &mock.Ledger{}
	var err error
	// prepare public parameters
	ipk, err = os.ReadFile("./testdata/bls12_381_bbs/idemix/msp/IssuerPublicKey")
	if err != nil {
		return nil, err
	}
	pp, err = v1.Setup(32, ipk, math.BLS12_381_BBS_GURVY)
	if err != nil {
		return nil, err
	}

	c := math.Curves[pp.Curve]

	asigner, _, err := prepareECDSASigner()
	if err != nil {
		return nil, err
	}
	idemixDes, err := idemix2.NewDeserializer(slices.GetUnique(pp.IdemixIssuerPublicKeys).PublicKey, math.BLS12_381_BBS_GURVY)
	if err != nil {
		return nil, err
	}
	multiplexer := deserializer.NewTypedVerifierDeserializerMultiplex()
	multiplexer.AddTypedVerifierDeserializer(idemix2.IdentityType, deserializer.NewTypedIdentityVerifierDeserializer(idemixDes, idemixDes))
	multiplexer.AddTypedVerifierDeserializer(ix509.IdentityType, deserializer.NewTypedIdentityVerifierDeserializer(&Deserializer{}, &Deserializer{}))
	auditor = audit.NewAuditor(logging.MustGetLogger(), &noop.Tracer{}, multiplexer, pp.PedersenGenerators, asigner, c)
	araw, err := asigner.Serialize()
	if err != nil {
		return nil, err
	}
	pp.SetAuditors([]driver.Identity{araw})

	// initialize enginw with pp
	des, err := zkatdlog.NewDeserializer(pp)
	if err != nil {
		return nil, err
	}
	engine = enginedlog.New(
		logging.MustGetLogger(),
		pp,
		des,
		nil,
		nil,
		nil,
	)

	// non-anonymous issue
	_, ir, _, err = prepareNonAnonymousIssueRequest(pp, auditor)
	if err != nil {
		return nil, err
	}

	// prepare redeem
	sender, rr, _, inputsForRedeem, err = prepareRedeemRequest(pp, auditor)
	if err != nil {
		return nil, err
	}

	// prepare transfer
	var trmetadata *driver.TokenRequestMetadata
	sender, tr, trmetadata, inputsForTransfer, err = prepareTransferRequest(pp, auditor)
	if err != nil {
		return nil, err
	}

	// atomic action request
	ar = &driver.TokenRequest{Transfers: tr.Transfers}
	raw, err := ar.MarshalToMessageToSign([]byte("2"))
	if err != nil {
		return nil, err
	}

	// sender signs request
	signatures, err := sender.SignTokenActions(raw)
	if err != nil {
		return nil, err
	}

	// auditor inspect token
	metadata := &driver.TokenRequestMetadata{}
	metadata.Transfers = []*driver.TransferMetadata{trmetadata.Transfers[0]}

	tokns := make([][]*tokn.Token, 1)
	for i := range 2 {
		tokns[0] = append(tokns[0], inputsForTransfer[i])
	}
	err = auditor.Check(context.Background(), ar, metadata, tokns, "2")
	if err != nil {
		return nil, err
	}
	sigma, err := auditor.Endorse(ar, "2")
	if err != nil {
		return nil, err
	}
	ar.AuditorSignatures = append(ar.AuditorSignatures, &driver.AuditorSignature{
		Identity:  araw,
		Signature: sigma,
	})

	ar.Signatures = append(ar.Signatures, signatures...)

	return &Env{
		ir:                ir,
		tr:                tr,
		engine:            engine,
		inputsForTransfer: inputsForTransfer,
		inputsForRedeem:   inputsForRedeem,
		rr:                rr,
		ar:                ar,
		sender:            sender,
	}, nil
}

func TestValidator(t *testing.T) {
	t.Run("Validator is called correctly with a non-anonymous issue action", func(t *testing.T) {
		env, err := NewEnv()
		require.NoError(t, err)

		raw, err := env.ir.Bytes()
		require.NoError(t, err)
		actions, _, err := env.engine.VerifyTokenRequestFromRaw(t.Context(), fakeLedger.GetStateStub, "1", raw)
		require.NoError(t, err)
		require.Len(t, actions, 1)
	})
	t.Run("validator is called correctly with a transfer action", func(t *testing.T) {
		env, err := NewEnv()
		require.NoError(t, err)

		raw, err := env.inputsForTransfer[0].Serialize()
		require.NoError(t, err)
		fakeLedger.GetStateReturnsOnCall(0, raw, nil)

		raw, err = env.inputsForTransfer[1].Serialize()
		require.NoError(t, err)
		fakeLedger.GetStateReturnsOnCall(1, raw, nil)

		raw, err = env.inputsForTransfer[0].Serialize()
		require.NoError(t, err)
		fakeLedger.GetStateReturnsOnCall(2, raw, nil)

		raw, err = env.inputsForTransfer[1].Serialize()
		require.NoError(t, err)
		fakeLedger.GetStateReturnsOnCall(3, raw, nil)

		fakeLedger.GetStateReturnsOnCall(4, nil, nil)
		fakeLedger.GetStateReturnsOnCall(5, nil, nil)

		raw, err = env.tr.Bytes()
		require.NoError(t, err)
		actions, _, err := env.engine.VerifyTokenRequestFromRaw(t.Context(), getState, "1", raw)
		require.NoError(t, err)
		require.Len(t, actions, 1)
	})
	t.Run("validator is called correctly with a redeem action", func(t *testing.T) {
		env, err := NewEnv()
		require.NoError(t, err)
		raw, err := env.inputsForRedeem[0].Serialize()
		require.NoError(t, err)
		fakeLedger.GetStateReturnsOnCall(0, raw, nil)

		raw, err = env.inputsForRedeem[1].Serialize()
		require.NoError(t, err)
		fakeLedger.GetStateReturnsOnCall(1, raw, nil)

		raw, err = env.inputsForRedeem[0].Serialize()
		require.NoError(t, err)
		fakeLedger.GetStateReturnsOnCall(2, raw, nil)

		raw, err = env.inputsForRedeem[1].Serialize()
		require.NoError(t, err)
		fakeLedger.GetStateReturnsOnCall(3, raw, nil)

		fakeLedger.GetStateReturnsOnCall(4, nil, nil)

		raw, err = env.rr.Bytes()
		require.NoError(t, err)

		actions, _, err := env.engine.VerifyTokenRequestFromRaw(t.Context(), getState, "1", raw)
		require.NoError(t, err)
		require.Len(t, actions, 1)
	})
	t.Run("engine is called correctly with atomic swap", func(t *testing.T) {
		env, err := NewEnv()
		require.NoError(t, err)

		raw, err := env.inputsForTransfer[0].Serialize()
		require.NoError(t, err)
		fakeLedger.GetStateReturnsOnCall(0, raw, nil)

		raw, err = env.inputsForTransfer[1].Serialize()
		require.NoError(t, err)
		fakeLedger.GetStateReturnsOnCall(1, raw, nil)

		fakeLedger.GetStateReturnsOnCall(2, nil, nil)

		raw, err = env.inputsForTransfer[0].Serialize()
		require.NoError(t, err)
		fakeLedger.GetStateReturnsOnCall(3, raw, nil)

		raw, err = env.inputsForTransfer[1].Serialize()
		require.NoError(t, err)
		fakeLedger.GetStateReturnsOnCall(4, raw, nil)

		fakeLedger.GetStateReturnsOnCall(5, nil, nil)
		fakeLedger.GetStateReturnsOnCall(6, nil, nil)

		raw, err = env.ar.Bytes()
		require.NoError(t, err)

		actions, _, err := env.engine.VerifyTokenRequestFromRaw(t.Context(), getState, "2", raw)
		require.NoError(t, err)
		require.Len(t, actions, 1)
	})
	t.Run("when the sender's signature is not valid: wrong txID", func(t *testing.T) {
		env, err := NewEnv()
		require.NoError(t, err)

		raw, err := env.inputsForTransfer[0].Serialize()
		require.NoError(t, err)
		fakeLedger.GetStateReturnsOnCall(0, raw, nil)

		raw, err = env.inputsForTransfer[1].Serialize()
		require.NoError(t, err)
		fakeLedger.GetStateReturnsOnCall(1, raw, nil)

		fakeLedger.GetStateReturnsOnCall(2, nil, nil)

		raw, err = env.inputsForTransfer[0].Serialize()
		require.NoError(t, err)
		fakeLedger.GetStateReturnsOnCall(3, raw, nil)

		raw, err = env.inputsForTransfer[1].Serialize()
		require.NoError(t, err)
		fakeLedger.GetStateReturnsOnCall(4, raw, nil)

		fakeLedger.GetStateReturnsOnCall(5, nil, nil)
		fakeLedger.GetStateReturnsOnCall(6, nil, nil)
		raw, err = env.ar.Bytes()
		require.NoError(t, err)

		request := &driver.TokenRequest{Issues: env.ar.Issues, Transfers: env.ar.Transfers}
		raw, err = request.MarshalToMessageToSign([]byte("3"))
		require.NoError(t, err)

		signatures, err := env.sender.SignTokenActions(raw)
		require.NoError(t, err)
		env.ar.Signatures[1] = signatures[0]

		raw, err = env.ar.Bytes()
		require.NoError(t, err)

		_, _, err = env.engine.VerifyTokenRequestFromRaw(t.Context(), getState, "2", raw)
		require.Error(t, err)
		require.ErrorContains(t, err, "failed signature verification")
	})
}

func prepareECDSASigner() (*Signer, *Verifier, error) {
	signer, err := NewECDSASigner()
	if err != nil {
		return nil, nil, err
	}
	return signer, signer.Verifier, nil
}

func prepareNonAnonymousIssueRequest(pp *v1.PublicParams, auditor *audit.Auditor) (*issue2.Issuer, *driver.TokenRequest, *driver.TokenRequestMetadata, error) {
	signer, err := NewECDSASigner()
	if err != nil {
		return nil, nil, nil, err
	}

	issuer := issue2.NewIssuer("ABC", signer, pp)
	issuerIdentity, err := signer.Serialize()
	if err != nil {
		return nil, nil, nil, err
	}
	ir, metadata, err := prepareIssue(auditor, issuer, issuerIdentity)
	if err != nil {
		return nil, nil, nil, err
	}

	return issuer, ir, metadata, nil
}

func prepareRedeemRequest(pp *v1.PublicParams, auditor *audit.Auditor) (*transfer.Sender, *driver.TokenRequest, *driver.TokenRequestMetadata, []*tokn.Token, error) {
	id, auditInfo, signer, err := getIdemixInfo("./testdata/bls12_381_bbs/idemix")
	if err != nil {
		return nil, nil, nil, nil, err
	}
	owners := make([][]byte, 2)
	owners[0] = id

	issuerSigner, err := NewECDSASigner()
	if err != nil {
		return nil, nil, nil, nil, err
	}

	issuer := issue2.NewIssuer("ABC", issuerSigner, pp)
	issuerIdentity, err := issuerSigner.Serialize()
	if err != nil {
		return nil, nil, nil, nil, err
	}

	return prepareTransfer(pp, signer, auditor, auditInfo, id, owners, issuer, issuerIdentity)
}

func prepareTransferRequest(pp *v1.PublicParams, auditor *audit.Auditor) (*transfer.Sender, *driver.TokenRequest, *driver.TokenRequestMetadata, []*tokn.Token, error) {
	id, auditInfo, signer, err := getIdemixInfo("./testdata/bls12_381_bbs/idemix")
	if err != nil {
		return nil, nil, nil, nil, err
	}
	owners := make([][]byte, 2)
	owners[0] = id
	owners[1] = id

	return prepareTransfer(pp, signer, auditor, auditInfo, id, owners, nil, nil)
}

func prepareTokens(values, bf []*math.Zr, ttype string, pp []*math.G1, curve *math.Curve) []*math.G1 {
	tokens := make([]*math.G1, len(values))
	for i := range values {
		tokens[i] = prepareToken(values[i], bf[i], ttype, pp, curve)
	}
	return tokens
}

func prepareToken(value *math.Zr, rand *math.Zr, ttype string, pp []*math.G1, curve *math.Curve) *math.G1 {
	token := curve.NewG1()
	token.Add(pp[0].Mul(curve.HashToZr([]byte(ttype))))
	token.Add(pp[1].Mul(value))
	token.Add(pp[2].Mul(rand))
	return token
}

func getIdemixInfo(dir string) (driver.Identity, *crypto.AuditInfo, driver.SigningIdentity, error) {
	backend, err := kvs.NewInMemory()
	if err != nil {
		return nil, nil, nil, err
	}
	config, err := crypto.NewConfig(dir)
	if err != nil {
		return nil, nil, nil, err
	}
	curveID := math.BLS12_381_BBS_GURVY
	keyStore, err := crypto.NewKeyStore(curveID, kvs.Keystore(backend))
	if err != nil {
		return nil, nil, nil, err
	}
	cryptoProvider, err := crypto.NewBCCSP(keyStore, curveID)
	if err != nil {
		return nil, nil, nil, err
	}
	p, err := idemix2.NewKeyManager(config, types.EidNymRhNym, cryptoProvider)
	if err != nil {
		return nil, nil, nil, err
	}

	identityDescriptor, err := p.Identity(context.Background(), nil)
	if err != nil {
		return nil, nil, nil, err
	}
	id := identityDescriptor.Identity
	audit := identityDescriptor.AuditInfo

	auditInfo, err := p.DeserializeAuditInfo(context.Background(), audit)
	if err != nil {
		return nil, nil, nil, err
	}
	err = auditInfo.Match(context.Background(), id)
	if err != nil {
		return nil, nil, nil, err
	}

	signer, err := p.DeserializeSigningIdentity(context.Background(), id)
	if err != nil {
		return nil, nil, nil, err
	}

	id, err = identity.WrapWithType(idemix2.IdentityType, id)
	if err != nil {
		return nil, nil, nil, err
	}

	return id, auditInfo, signer, nil
}

func prepareIssue(auditor *audit.Auditor, issuer *issue2.Issuer, issuerIdentity []byte) (*driver.TokenRequest, *driver.TokenRequestMetadata, error) {
	id, auditInfo, _, err := getIdemixInfo("./testdata/bls12_381_bbs/idemix")
	if err != nil {
		return nil, nil, err
	}
	owners := make([][]byte, 1)
	owners[0] = id
	values := []uint64{40}

	issue, inf, err := issuer.GenerateZKIssue(values, owners)
	if err != nil {
		return nil, nil, err
	}

	auditInfoRaw, err := auditInfo.Bytes()
	if err != nil {
		return nil, nil, err
	}
	metadata := &driver.IssueMetadata{
		Issuer: driver.AuditableIdentity{
			Identity:  issuerIdentity,
			AuditInfo: issuerIdentity,
		},
	}
	for i := range len(issue.Outputs) {
		marshalledinf, err := inf[i].Serialize()
		if err != nil {
			return nil, nil, err
		}
		metadata.Outputs = append(metadata.Outputs, &driver.IssueOutputMetadata{
			OutputMetadata: marshalledinf,
			Receivers: []*driver.AuditableIdentity{
				{
					Identity:  nil,
					AuditInfo: auditInfoRaw,
				},
			},
		})
	}

	// serialize token action
	raw, err := issue.Serialize()
	if err != nil {
		return nil, nil, err
	}

	// sign token request
	ir := &driver.TokenRequest{Issues: [][]byte{raw}}
	raw, err = ir.MarshalToMessageToSign([]byte("1"))
	if err != nil {
		return nil, nil, err
	}

	sig, err := issuer.SignTokenActions(raw)
	if err != nil {
		return nil, nil, err
	}
	ir.Signatures = append(ir.Signatures, sig)

	issueMetadata := &driver.TokenRequestMetadata{Issues: []*driver.IssueMetadata{metadata}}
	err = auditor.Check(context.Background(), ir, issueMetadata, nil, "1")
	if err != nil {
		return nil, nil, err
	}
	sigma, err := auditor.Endorse(ir, "1")
	if err != nil {
		return nil, nil, err
	}
	araw, err := auditor.Signer.Serialize()
	if err != nil {
		return nil, nil, err
	}
	ir.AuditorSignatures = append(ir.AuditorSignatures, &driver.AuditorSignature{
		Identity:  araw,
		Signature: sigma,
	})

	return ir, issueMetadata, nil
}

func prepareTransfer(pp *v1.PublicParams, signer driver.SigningIdentity, auditor *audit.Auditor, auditInfo *crypto.AuditInfo, id []byte, owners [][]byte, issuer *issue2.Issuer, issuerIdentity []byte) (*transfer.Sender, *driver.TokenRequest, *driver.TokenRequestMetadata, []*tokn.Token, error) {
	signers := make([]driver.Signer, 2)
	signers[0] = signer
	signers[1] = signer
	c := math.Curves[pp.Curve]

	invalues := make([]*math.Zr, 2)
	invalues[0] = c.NewZrFromInt(70)
	invalues[1] = c.NewZrFromInt(30)

	inBF := make([]*math.Zr, 2)
	rand, err := c.Rand()
	if err != nil {
		return nil, nil, nil, nil, err
	}
	for i := range 2 {
		inBF[i] = c.NewRandomZr(rand)
	}
	outvalues := make([]uint64, 2)
	outvalues[0] = 65
	outvalues[1] = 35

	ids := make([]*token2.ID, 2)
	ids[0] = &token2.ID{TxId: "0"}
	ids[1] = &token2.ID{TxId: "1"}

	inputs := prepareTokens(invalues, inBF, "ABC", pp.PedersenGenerators, c)
	tokens := make([]*tokn.Token, 2)
	tokens[0] = &tokn.Token{Data: inputs[0], Owner: id}
	tokens[1] = &tokn.Token{Data: inputs[1], Owner: id}

	inputInf := make([]*tokn.Metadata, 2)
	inputInf[0] = &tokn.Metadata{Type: "ABC", Value: invalues[0], BlindingFactor: inBF[0]}
	inputInf[1] = &tokn.Metadata{Type: "ABC", Value: invalues[1], BlindingFactor: inBF[1]}
	sender, err := transfer.NewSender(signers, tokens, ids, inputInf, pp)
	if err != nil {
		return nil, nil, nil, nil, err
	}

	transfer2, metas, err := sender.GenerateZKTransfer(context.Background(), outvalues, owners)
	if err != nil {
		return nil, nil, nil, nil, err
	}

	if issuerIdentity != nil {
		transfer2.Issuer = issuerIdentity
	}

	transferRaw, err := transfer2.Serialize()
	if err != nil {
		return nil, nil, nil, nil, err
	}

	tr := &driver.TokenRequest{Transfers: [][]byte{transferRaw}}
	raw, err := tr.MarshalToMessageToSign([]byte("1"))
	if err != nil {
		return nil, nil, nil, nil, err
	}

	marshalledInfo := make([][]byte, len(metas))
	for i := range metas {
		marshalledInfo[i], err = metas[i].Serialize()
		if err != nil {
			return nil, nil, nil, nil, err
		}
	}
	auditInfoRaw, err := auditInfo.Bytes()
	if err != nil {
		return nil, nil, nil, nil, err
	}
	metadata := &driver.TransferMetadata{}
	for range len(transfer2.Inputs) {
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

	for i := range len(transfer2.Outputs) {
		marshalledinf, err := metas[i].Serialize()
		if err != nil {
			return nil, nil, nil, nil, err
		}
		metadata.Outputs = append(metadata.Outputs, &driver.TransferOutputMetadata{
			OutputMetadata:  marshalledinf,
			OutputAuditInfo: auditInfoRaw,
			Receivers: []*driver.AuditableIdentity{
				{
					Identity:  nil,
					AuditInfo: auditInfoRaw,
				},
			},
		})
	}

	tokns := make([][]*tokn.Token, 1)
	tokns[0] = append(tokns[0], tokens...)

	if issuerIdentity != nil {
		metadata.Issuer = driver.Identity(issuerIdentity)
	}

	transferMetadata := &driver.TokenRequestMetadata{Transfers: []*driver.TransferMetadata{metadata}}
	err = auditor.Check(context.Background(), tr, transferMetadata, tokns, "1")
	if err != nil {
		return nil, nil, nil, nil, err
	}

	sigma, err := auditor.Endorse(tr, "1")
	if err != nil {
		return nil, nil, nil, nil, err
	}
	araw, err := auditor.Signer.Serialize()
	if err != nil {
		return nil, nil, nil, nil, err
	}
	tr.AuditorSignatures = append(tr.AuditorSignatures, &driver.AuditorSignature{
		Identity:  araw,
		Signature: sigma,
	})

	signatures, err := sender.SignTokenActions(raw)
	if err != nil {
		return nil, nil, nil, nil, err
	}
	tr.Signatures = append(tr.Signatures, signatures...)

	// Add issuer signature for redeem case
	if issuer != nil {
		issuerSignature, err := issuer.Signer.Sign(raw)
		if err != nil {
			return nil, nil, nil, nil, err
		}
		tr.Signatures = append(tr.Signatures, issuerSignature)
	}

	return sender, tr, transferMetadata, tokens, nil
}

func getState(id token2.ID) ([]byte, error) {
	return fakeLedger.GetState(id)
}

var (
	// curveHalfOrders contains the precomputed curve group orders halved.
	// It is used to ensure that signature' S value is lower or equal to the
	// curve group order halved. We accept only low-S signatures.
	// They are precomputed for efficiency reasons.
	curveHalfOrders = map[elliptic.Curve]*big.Int{
		elliptic.P224(): new(big.Int).Rsh(elliptic.P224().Params().N, 1),
		elliptic.P256(): new(big.Int).Rsh(elliptic.P256().Params().N, 1),
		elliptic.P384(): new(big.Int).Rsh(elliptic.P384().Params().N, 1),
		elliptic.P521(): new(big.Int).Rsh(elliptic.P521().Params().N, 1),
	}
)

type Signature struct {
	R, S *big.Int
}

type Signer struct {
	*Verifier
	SK *ecdsa.PrivateKey
}

func (d *Signer) Sign(message []byte) ([]byte, error) {
	dgst := sha256.Sum256(message)

	r, s, err := ecdsa.Sign(rand.Reader, d.SK, dgst[:])
	if err != nil {
		return nil, err
	}

	s, _, err = ToLowS(&d.SK.PublicKey, s)
	if err != nil {
		return nil, err
	}

	return utils.MarshalECDSASignature(r, s)
}

func (d *Signer) Serialize() ([]byte, error) {
	return d.Verifier.Serialize()
}

type Verifier struct {
	PK *ecdsa.PublicKey
}

func NewECDSASigner() (*Signer, error) {
	// Create ephemeral key and store it in the context
	sk, err := ecdsa.GenerateKey(elliptic.P256(), rand.Reader)
	if err != nil {
		return nil, err
	}
	return &Signer{SK: sk, Verifier: &Verifier{PK: &sk.PublicKey}}, nil
}

func (v *Verifier) Verify(message, sigma []byte) error {
	signature := &Signature{}
	_, err := asn1.Unmarshal(sigma, signature)
	if err != nil {
		return err
	}

	hash := sha256.New()
	n, err := hash.Write(message)
	if n != len(message) {
		return errors.Errorf("hash failure")
	}
	if err != nil {
		return err
	}
	digest := hash.Sum(nil)

	lowS, err := IsLowS(v.PK, signature.S)
	if err != nil {
		return err
	}
	if !lowS {
		return errors.New("signature is not in lowS")
	}

	valid := ecdsa.Verify(v.PK, digest, signature.R, signature.S)
	if !valid {
		return errors.Errorf("signature not valid")
	}

	return nil
}

func (v *Verifier) Serialize() ([]byte, error) {
	pkRaw, err := PemEncodeKey(v.PK)
	if err != nil {
		return nil, errors.Wrap(err, "failed marshalling public key")
	}

	wrap, err := identity.WrapWithType(ix509.IdentityType, pkRaw)
	if err != nil {
		return nil, errors.Wrap(err, "failed wrapping identity")
	}

	return wrap, nil
}

// PemEncodeKey takes a Go key and converts it to bytes
func PemEncodeKey(key interface{}) ([]byte, error) {
	var encoded []byte
	var err error
	var keyType string

	switch key.(type) {
	case *ecdsa.PrivateKey, *rsa.PrivateKey:
		keyType = "PRIVATE"
		encoded, err = x509.MarshalPKCS8PrivateKey(key)
	case *ecdsa.PublicKey, *rsa.PublicKey:
		keyType = "PUBLIC"
		encoded, err = x509.MarshalPKIXPublicKey(key)
	default:
		err = errors.Errorf("Programming error, unexpected key type %T", key)
	}
	if err != nil {
		return nil, err
	}

	return pem.EncodeToMemory(&pem.Block{Type: keyType + " KEY", Bytes: encoded}), nil
}

// IsLowS checks that s is a low-S
func IsLowS(k *ecdsa.PublicKey, s *big.Int) (bool, error) {
	halfOrder, ok := curveHalfOrders[k.Curve]
	if !ok {
		return false, fmt.Errorf("curve not recognized [%s]", k.Curve)
	}

	return s.Cmp(halfOrder) != 1, nil
}

func ToLowS(k *ecdsa.PublicKey, s *big.Int) (*big.Int, bool, error) {
	lowS, err := IsLowS(k, s)
	if err != nil {
		return nil, false, err
	}

	if !lowS {
		// Set s to N - s that will be then in the lower part of signature space
		// less or equal to half order
		s.Sub(k.Params().N, s)

		return s, true, nil
	}

	return s, false, nil
}

type Deserializer struct {
	auditInfo []byte
}

func (d *Deserializer) Match(_ context.Context, id []byte) error {
	identity, err := identity.WrapWithType(ix509.IdentityType, id)
	if err != nil {
		return errors.Wrapf(err, "failed to unmarshal identity [%s]", id)
	}
	if !bytes.Equal(d.auditInfo, identity) {
		return errors.Errorf("identity mismatch [%s][%s]", utils2.Hashable(identity), utils2.Hashable(d.auditInfo))
	}
	return nil
}

func (d *Deserializer) GetAuditInfoMatcher(_ context.Context, _ driver.Identity, auditInfo []byte) (driver.Matcher, error) {
	return &Deserializer{auditInfo: auditInfo}, nil
}

func (d *Deserializer) DeserializeVerifier(_ context.Context, _ driver.Identity) (driver.Verifier, error) {
	panic("implement me")
}
