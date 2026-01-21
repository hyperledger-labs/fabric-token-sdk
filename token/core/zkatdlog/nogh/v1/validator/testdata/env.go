/*
Copyright IBM Corp. All Rights Reserved.

SPDX-License-Identifier: Apache-2.0
*/

package testdata

import (
	"bytes"
	"context"
	"encoding/base64"
	"encoding/json"
	"os"

	math "github.com/IBM/mathlib"
	"github.com/hyperledger-labs/fabric-smart-client/pkg/utils/errors"
	"github.com/hyperledger-labs/fabric-token-sdk/token/core/zkatdlog/nogh/v1/audit"
	"github.com/hyperledger-labs/fabric-token-sdk/token/core/zkatdlog/nogh/v1/benchmark"
	zkatdlog "github.com/hyperledger-labs/fabric-token-sdk/token/core/zkatdlog/nogh/v1/driver"
	issue2 "github.com/hyperledger-labs/fabric-token-sdk/token/core/zkatdlog/nogh/v1/issue"
	v1 "github.com/hyperledger-labs/fabric-token-sdk/token/core/zkatdlog/nogh/v1/setup"
	tokn "github.com/hyperledger-labs/fabric-token-sdk/token/core/zkatdlog/nogh/v1/token"
	"github.com/hyperledger-labs/fabric-token-sdk/token/core/zkatdlog/nogh/v1/transfer"
	"github.com/hyperledger-labs/fabric-token-sdk/token/core/zkatdlog/nogh/v1/validator"
	"github.com/hyperledger-labs/fabric-token-sdk/token/driver"
	benchmark2 "github.com/hyperledger-labs/fabric-token-sdk/token/services/benchmark"
	"github.com/hyperledger-labs/fabric-token-sdk/token/services/identity"
	"github.com/hyperledger-labs/fabric-token-sdk/token/services/identity/deserializer"
	idemix2 "github.com/hyperledger-labs/fabric-token-sdk/token/services/identity/idemix"
	"github.com/hyperledger-labs/fabric-token-sdk/token/services/identity/idemix/crypto"
	ix509 "github.com/hyperledger-labs/fabric-token-sdk/token/services/identity/x509"
	"github.com/hyperledger-labs/fabric-token-sdk/token/services/logging"
	utils2 "github.com/hyperledger-labs/fabric-token-sdk/token/services/utils"
	"github.com/hyperledger-labs/fabric-token-sdk/token/services/utils/slices"
	token2 "github.com/hyperledger-labs/fabric-token-sdk/token/token"
	"go.opentelemetry.io/otel/trace/noop"
)

var (
	testUseCase = &benchmark2.Case{
		Bits:       32,
		CurveID:    math.BLS12_381_BBS_GURVY,
		NumInputs:  2,
		NumOutputs: 2,
	}
)

type actionType int

const (
	TransferAction actionType = iota
	RedeemAction
	IssueAction
)

type Env struct {
	Engine            *validator.Validator
	inputsForTransfer []*tokn.Token
	inputsForRedeem   []*tokn.Token
	Sender            *transfer.Sender

	TRWithTransferTxID string
	TRWithTransfer     *driver.TokenRequest
	TRWithTransferRaw  []byte

	TRWithRedeem *driver.TokenRequest
	TRWithIssue  *driver.TokenRequest
	TRWithSwap   *driver.TokenRequest
}

// SaveTransferToFile writes TRWithTransferTxID and TRWithTransferRaw (base64-encoded)
// into the provided path as JSON.
func (e *Env) SaveTransferToFile(path string) error {
	if e == nil {
		return errors.Errorf("nil Env")
	}

	payload := struct {
		TxID   string `json:"txid"`
		ReqRaw string `json:"req_raw"`
	}{
		TxID:   e.TRWithTransferTxID,
		ReqRaw: base64.StdEncoding.EncodeToString(e.TRWithTransferRaw),
	}

	b, err := json.MarshalIndent(payload, "", "  ")
	if err != nil {
		return err
	}

	if err := os.WriteFile(path, b, 0o600); err != nil {
		return err
	}
	return nil
}

func NewEnv(benchCase *benchmark2.Case, configurations *benchmark.SetupConfigurations) (*Env, error) {
	var (
		engine *validator.Validator

		inputsForRedeem   []*tokn.Token
		inputsForTransfer []*tokn.Token

		sender  *transfer.Sender
		auditor *audit.Auditor

		ir *driver.TokenRequest // regular issue request
		rr *driver.TokenRequest // redeem request
		tr *driver.TokenRequest // transfer request
		ar *driver.TokenRequest // atomic action request
	)

	// prepare public parameters
	setupConfiguration, err := configurations.GetSetupConfiguration(benchCase.Bits, benchCase.CurveID)
	if err != nil {
		return nil, err
	}
	pp := setupConfiguration.PP
	oID := setupConfiguration.OwnerIdentity

	c := math.Curves[pp.Curve]

	idemixDes, err := idemix2.NewDeserializer(slices.GetUnique(pp.IdemixIssuerPublicKeys).PublicKey, benchCase.CurveID)
	if err != nil {
		return nil, err
	}
	multiplexer := deserializer.NewTypedVerifierDeserializerMultiplex()
	multiplexer.AddTypedVerifierDeserializer(idemix2.IdentityType, deserializer.NewTypedIdentityVerifierDeserializer(idemixDes, idemixDes))
	multiplexer.AddTypedVerifierDeserializer(ix509.IdentityType, deserializer.NewTypedIdentityVerifierDeserializer(&Deserializer{}, &Deserializer{}))
	auditor = audit.NewAuditor(
		logging.MustGetLogger(),
		&noop.Tracer{},
		multiplexer,
		pp.PedersenGenerators,
		setupConfiguration.AuditorSigner,
		c,
	)

	// initialize enginw with pp
	des, err := zkatdlog.NewDeserializer(pp)
	if err != nil {
		return nil, err
	}
	engine = validator.New(
		logging.MustGetLogger(),
		pp,
		des,
		nil,
		nil,
		nil,
	)

	// non-anonymous issue
	_, ir, _, err = prepareNonAnonymousIssueRequest(pp, auditor, setupConfiguration)
	if err != nil {
		return nil, err
	}

	// prepare redeem
	_, rr, _, inputsForRedeem, err = prepareRedeemRequest(pp, auditor, setupConfiguration)
	if err != nil {
		return nil, err
	}

	// prepare transfer
	var trmetadata *driver.TokenRequestMetadata
	sender, tr, trmetadata, inputsForTransfer, err = prepareTransferRequest(pp, auditor, oID)
	if err != nil {
		return nil, err
	}
	transferRaw, err := tr.Bytes()
	if err != nil {
		return nil, err
	}

	// atomic action request
	ar = &driver.TokenRequest{Transfers: tr.Transfers}
	raw, err := ar.MarshalToMessageToSign([]byte("2"))
	if err != nil {
		return nil, err
	}

	// Sender signs request
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
		Identity:  pp.Auditors()[0],
		Signature: sigma,
	})

	ar.Signatures = append(ar.Signatures, signatures...)

	return &Env{
		TRWithIssue:        ir,
		TRWithTransfer:     tr,
		Engine:             engine,
		inputsForTransfer:  inputsForTransfer,
		inputsForRedeem:    inputsForRedeem,
		TRWithRedeem:       rr,
		TRWithSwap:         ar,
		Sender:             sender,
		TRWithTransferRaw:  transferRaw,
		TRWithTransferTxID: "1",
	}, nil
}

func prepareNonAnonymousIssueRequest(pp *v1.PublicParams, auditor *audit.Auditor, setupConfiguration *benchmark.SetupConfiguration) (*issue2.Issuer, *driver.TokenRequest, *driver.TokenRequestMetadata, error) {
	issuer := issue2.NewIssuer("ABC", setupConfiguration.IssuerSigner, pp)
	issuerIdentity, err := setupConfiguration.IssuerSigner.Serialize()
	if err != nil {
		return nil, nil, nil, err
	}
	ir, metadata, err := prepareIssue(auditor, issuer, issuerIdentity, setupConfiguration.OwnerIdentity)
	if err != nil {
		return nil, nil, nil, err
	}

	return issuer, ir, metadata, nil
}

func prepareRedeemRequest(pp *v1.PublicParams, auditor *audit.Auditor, setupConfig *benchmark.SetupConfiguration) (*transfer.Sender, *driver.TokenRequest, *driver.TokenRequestMetadata, []*tokn.Token, error) {
	owners := make([][]byte, 2)
	owners[0] = setupConfig.OwnerIdentity.ID

	issuer := issue2.NewIssuer("ABC", setupConfig.IssuerSigner, pp)
	issuerIdentity, err := setupConfig.IssuerSigner.Serialize()
	if err != nil {
		return nil, nil, nil, nil, err
	}

	return prepareTransfer(
		pp,
		setupConfig.OwnerIdentity.Signer,
		auditor,
		setupConfig.OwnerIdentity.AuditInfo,
		setupConfig.OwnerIdentity.ID,
		owners,
		issuer,
		issuerIdentity,
	)
}

func prepareTransferRequest(pp *v1.PublicParams, auditor *audit.Auditor, oID *benchmark.OwnerIdentity) (*transfer.Sender, *driver.TokenRequest, *driver.TokenRequestMetadata, []*tokn.Token, error) {
	owners := make([][]byte, 2)
	owners[0] = oID.ID
	owners[1] = oID.ID

	return prepareTransfer(pp, oID.Signer, auditor, oID.AuditInfo, oID.ID, owners, nil, nil)
}

func prepareTokens(values, bf []*math.Zr, tokenType string, pp []*math.G1, curve *math.Curve) []*math.G1 {
	tokens := make([]*math.G1, len(values))
	for i := range values {
		tokens[i] = prepareToken(values[i], bf[i], tokenType, pp, curve)
	}
	return tokens
}

func prepareToken(value *math.Zr, rand *math.Zr, tokenType string, pp []*math.G1, curve *math.Curve) *math.G1 {
	token := curve.NewG1()
	token.Add(pp[0].Mul(curve.HashToZr([]byte(tokenType))))
	token.Add(pp[1].Mul(value))
	token.Add(pp[2].Mul(rand))
	return token
}

func prepareIssue(auditor *audit.Auditor, issuer *issue2.Issuer, issuerIdentity []byte, oID *benchmark.OwnerIdentity) (*driver.TokenRequest, *driver.TokenRequestMetadata, error) {
	owners := make([][]byte, 1)
	owners[0] = oID.ID
	values := []uint64{40}

	issue, inf, err := issuer.GenerateZKIssue(values, owners)
	if err != nil {
		return nil, nil, err
	}

	auditInfoRaw, err := oID.AuditInfo.Bytes()
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
