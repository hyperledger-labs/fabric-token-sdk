/*
Copyright IBM Corp. All Rights Reserved.

SPDX-License-Identifier: Apache-2.0
*/

package testutils

import (
	"bytes"
	"context"
	"encoding/base64"
	"encoding/json"
	"os"
	"strconv"

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

type Env struct {
	Engine *validator.Validator

	TRWithTransferTxID string
	TRWithTransfer     *driver.TokenRequest
	TRWithTransferRaw  []byte

	TRWithRedeem     *driver.TokenRequest
	TRWithRedeemTxID string
	TRWithRedeemRaw  []byte

	TRWithIssue     *driver.TokenRequest
	TRWithIssueTxID string
	TRWithIssueRaw  []byte

	Sender         *transfer.Sender
	TRWithSwap     *driver.TokenRequest
	TRWithSwapTxID string
	TRWithSwapRaw  []byte
}

// SaveTransferToFile writes TRWithTransferTxID and TRWithTransferRaw (base64-encoded)
// into the provided path as JSON.
func (e *Env) SaveTransferToFile(path string) error {
	return e.saveToFile(path, e.TRWithTransferTxID, e.TRWithTransferRaw)
}

// SaveIssueToFile writes TRWithIssueTxID and TRWithIssueRaw (base64-encoded)
// into the provided path as JSON.
func (e *Env) SaveIssueToFile(path string) error {
	return e.saveToFile(path, e.TRWithIssueTxID, e.TRWithIssueRaw)
}

// SaveRedeemToFile writes TRWithRedeemTxID and TRWithRedeemRaw (base64-encoded)
// into the provided path as JSON.
func (e *Env) SaveRedeemToFile(path string) error {
	return e.saveToFile(path, e.TRWithRedeemTxID, e.TRWithRedeemRaw)
}

// SaveSwapToFile writes TRWithSwapTxID and TRWithSwapRaw (base64-encoded)
// into the provided path as JSON.
func (e *Env) SaveSwapToFile(path string) error {
	return e.saveToFile(path, e.TRWithSwapTxID, e.TRWithSwapRaw)
}

func (e *Env) saveToFile(path string, txID string, raw []byte) error {
	if e == nil {
		return errors.Errorf("nil Env")
	}

	payload := struct {
		TxID   string `json:"txid"`
		ReqRaw string `json:"req_raw"`
	}{
		TxID:   txID,
		ReqRaw: base64.StdEncoding.EncodeToString(raw),
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
	auditor = audit.NewAuditor(logging.MustGetLogger(), &noop.Tracer{}, multiplexer, pp.PedersenGenerators, c)

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
	irRaw, err := ir.Bytes()
	if err != nil {
		return nil, err
	}

	// prepare redeem
	_, rr, _, _, err = prepareRedeemRequest(benchCase, pp, auditor, setupConfiguration)
	if err != nil {
		return nil, err
	}
	rrRaw, err := rr.Bytes()
	if err != nil {
		return nil, err
	}

	// prepare transfer
	_, tr, _, _, err = prepareTransferRequest(benchCase, pp, auditor, setupConfiguration.AuditorSigner, oID)
	if err != nil {
		return nil, err
	}
	transferRaw, err := tr.Bytes()
	if err != nil {
		return nil, err
	}

	// atomic action request
	sender, ar, _, _, err = prepareSwapRequest(benchCase, pp, auditor, setupConfiguration.AuditorSigner, oID)
	if err != nil {
		return nil, err
	}
	arRaw, err := ar.Bytes()
	if err != nil {
		return nil, err
	}

	return &Env{
		Engine: engine,
		Sender: sender,

		TRWithTransferTxID: "1",
		TRWithTransfer:     tr,
		TRWithTransferRaw:  transferRaw,
		TRWithRedeem:       rr,
		TRWithRedeemTxID:   "1",
		TRWithRedeemRaw:    rrRaw,
		TRWithIssue:        ir,
		TRWithIssueTxID:    "1",
		TRWithIssueRaw:     irRaw,
		TRWithSwap:         ar,
		TRWithSwapTxID:     "2",
		TRWithSwapRaw:      arRaw,
	}, nil
}

func prepareNonAnonymousIssueRequest(pp *v1.PublicParams, auditor *audit.Auditor, setupConfiguration *benchmark.SetupConfiguration) (*issue2.Issuer, *driver.TokenRequest, *driver.TokenRequestMetadata, error) {
	issuer := issue2.NewIssuer("ABC", setupConfiguration.IssuerSigner, pp)
	issuerIdentity, err := setupConfiguration.IssuerSigner.Serialize()
	if err != nil {
		return nil, nil, nil, err
	}
	ir, metadata, err := prepareIssue(auditor, issuer, issuerIdentity, setupConfiguration.OwnerIdentity, setupConfiguration.AuditorSigner)
	if err != nil {
		return nil, nil, nil, err
	}

	return issuer, ir, metadata, nil
}

func prepareRedeemRequest(benchCase *benchmark2.Case, pp *v1.PublicParams, auditor *audit.Auditor, setupConfig *benchmark.SetupConfiguration) (*transfer.Sender, *driver.TokenRequest, *driver.TokenRequestMetadata, []*tokn.Token, error) {
	benchCaseRedeem := &benchmark2.Case{
		Workers:    benchCase.Workers,
		Bits:       benchCase.Bits,
		CurveID:    benchCase.CurveID,
		NumInputs:  benchCase.NumInputs,
		NumOutputs: 2,
	}
	owners := make([][]byte, 2)
	for i := range benchCase.NumInputs {
		owners[i] = setupConfig.OwnerIdentity.ID
	}

	issuer := issue2.NewIssuer("ABC", setupConfig.IssuerSigner, pp)
	issuerIdentity, err := setupConfig.IssuerSigner.Serialize()
	if err != nil {
		return nil, nil, nil, nil, err
	}

	return prepareTransfer(
		benchCaseRedeem,
		pp,
		setupConfig.OwnerIdentity.Signer,
		auditor,
		setupConfig.OwnerIdentity.AuditInfo,
		setupConfig.OwnerIdentity.ID,
		owners,
		issuer,
		issuerIdentity,
		setupConfig.AuditorSigner,
	)
}

func prepareTransferRequest(benchCase *benchmark2.Case, pp *v1.PublicParams, auditor *audit.Auditor, signer *benchmark.Signer, oID *benchmark.OwnerIdentity) (*transfer.Sender, *driver.TokenRequest, *driver.TokenRequestMetadata, []*tokn.Token, error) {
	owners := make([][]byte, benchCase.NumOutputs)
	for i := range benchCase.NumOutputs {
		owners[i] = oID.ID
	}

	return prepareTransfer(
		benchCase,
		pp,
		oID.Signer,
		auditor,
		oID.AuditInfo,
		oID.ID,
		owners,
		nil,
		nil,
		signer,
	)
}

func prepareSwapRequest(benchCase *benchmark2.Case, pp *v1.PublicParams, auditor *audit.Auditor, auditorSigner *benchmark.Signer, oID *benchmark.OwnerIdentity) (*transfer.Sender, *driver.TokenRequest, *driver.TokenRequestMetadata, []*tokn.Token, error) {
	sender1, tr1, trmetadata1, inputsForTransfer1, err := prepareTransferRequest(benchCase, pp, auditor, auditorSigner, oID)
	if err != nil {
		return nil, nil, nil, nil, err
	}
	sender2, tr2, trmetadata2, inputsForTransfer2, err := prepareTransferRequest(benchCase, pp, auditor, auditorSigner, oID)
	if err != nil {
		return nil, nil, nil, nil, err
	}
	//
	ar := &driver.TokenRequest{Transfers: append(tr1.Transfers, tr2.Transfers...)}
	raw, err := ar.MarshalToMessageToSign([]byte("2"))
	if err != nil {
		return nil, nil, nil, nil, err
	}

	// Sender signs request
	sender1Signatures, err := sender1.SignTokenActions(raw)
	if err != nil {
		return nil, nil, nil, nil, err
	}
	sender2Signatures, err := sender2.SignTokenActions(raw)
	if err != nil {
		return nil, nil, nil, nil, err
	}

	// auditor inspect token
	metadata := &driver.TokenRequestMetadata{}
	metadata.Transfers = []*driver.TransferMetadata{
		trmetadata1.Transfers[0],
		trmetadata2.Transfers[0],
	}

	tokns := make([][]*tokn.Token, 2)
	for i := range benchCase.NumInputs {
		tokns[0] = append(tokns[0], inputsForTransfer1[i])
	}
	for i := range benchCase.NumInputs {
		tokns[1] = append(tokns[1], inputsForTransfer2[i])
	}
	err = auditor.Check(context.Background(), ar, metadata, tokns, "2")
	if err != nil {
		return nil, nil, nil, nil, err
	}
	sigma, err := auditorEndorse(auditorSigner, ar, "2")
	if err != nil {
		return nil, nil, nil, nil, err
	}
	ar.AuditorSignatures = append(ar.AuditorSignatures, &driver.AuditorSignature{
		Identity:  pp.Auditors()[0],
		Signature: sigma,
	})

	ar.Signatures = append(ar.Signatures, sender1Signatures...)
	ar.Signatures = append(ar.Signatures, sender2Signatures...)

	return sender1, ar, metadata, nil, nil
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

func prepareIssue(auditor *audit.Auditor, issuer *issue2.Issuer, issuerIdentity []byte, oID *benchmark.OwnerIdentity, auditorSigner *benchmark.Signer) (*driver.TokenRequest, *driver.TokenRequestMetadata, error) {
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
	sigma, err := auditorEndorse(auditorSigner, ir, "1")
	if err != nil {
		return nil, nil, err
	}
	araw, err := auditorSigner.Serialize()
	if err != nil {
		return nil, nil, err
	}
	ir.AuditorSignatures = append(ir.AuditorSignatures, &driver.AuditorSignature{
		Identity:  araw,
		Signature: sigma,
	})

	return ir, issueMetadata, nil
}

func prepareTransfer(
	benchCase *benchmark2.Case,
	pp *v1.PublicParams,
	signer driver.SigningIdentity,
	auditor *audit.Auditor,
	auditInfo *crypto.AuditInfo,
	id []byte,
	owners [][]byte,
	issuer *issue2.Issuer,
	issuerIdentity []byte,
	auditorSigner *benchmark.Signer,
) (*transfer.Sender, *driver.TokenRequest, *driver.TokenRequestMetadata, []*tokn.Token, error) {
	signers := make([]driver.Signer, benchCase.NumInputs)
	for i := range benchCase.NumInputs {
		signers[i] = signer
	}
	c := math.Curves[pp.Curve]

	// prepare inputs
	inValues := make([]*math.Zr, benchCase.NumInputs)
	sumInputs := uint64(0)
	for i := range inValues {
		v := uint64(i*10 + 500)
		sumInputs += v
		inValues[i] = c.NewZrFromUint64(v)
	}

	if benchCase.NumOutputs <= 0 {
		return nil, nil, nil, nil, errors.Errorf("invalid number of outputs [%d]", benchCase.NumOutputs)
	}
	outputValue := sumInputs / uint64(benchCase.NumOutputs)
	sumOutputs := uint64(0)
	outValues := make([]uint64, benchCase.NumOutputs)
	for i := range benchCase.NumOutputs {
		outValues[i] = outputValue
		sumOutputs += outputValue
	}
	// add any adjustment to the last output
	delta := sumInputs - sumOutputs
	if delta > 0 {
		outValues[0] += delta
	}

	inBF := make([]*math.Zr, benchCase.NumInputs)
	rand, err := c.Rand()
	if err != nil {
		return nil, nil, nil, nil, err
	}
	for i := range benchCase.NumInputs {
		inBF[i] = c.NewRandomZr(rand)
	}

	ids := make([]*token2.ID, benchCase.NumInputs)
	for i := range benchCase.NumInputs {
		ids[i] = &token2.ID{TxId: strconv.Itoa(i)}
	}
	inputs := prepareTokens(inValues, inBF, "ABC", pp.PedersenGenerators, c)

	tokens := make([]*tokn.Token, benchCase.NumInputs)
	inputInf := make([]*tokn.Metadata, benchCase.NumInputs)
	for i := range benchCase.NumInputs {
		tokens[i] = &tokn.Token{Data: inputs[i], Owner: id}
		inputInf[i] = &tokn.Metadata{Type: "ABC", Value: inValues[i], BlindingFactor: inBF[i]}
	}

	sender, err := transfer.NewSender(signers, tokens, ids, inputInf, pp)
	if err != nil {
		return nil, nil, nil, nil, err
	}

	transfer2, metas, err := sender.GenerateZKTransfer(context.Background(), outValues, owners)
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
		metadata.Issuer = issuerIdentity
	}

	transferMetadata := &driver.TokenRequestMetadata{Transfers: []*driver.TransferMetadata{metadata}}
	err = auditor.Check(context.Background(), tr, transferMetadata, tokns, "1")
	if err != nil {
		return nil, nil, nil, nil, err
	}

	sigma, err := auditorEndorse(auditorSigner, tr, "1")
	if err != nil {
		return nil, nil, nil, nil, err
	}
	araw, err := auditorSigner.Serialize()
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

func auditorEndorse(signer driver.Signer, tokenRequest *driver.TokenRequest, txID string) ([]byte, error) {
	// Marshal tokenRequest
	bytes, err := tokenRequest.MarshalToMessageToSign([]byte(txID))
	if err != nil {
		return nil, errors.Wrapf(err, "failed marshalling token request [%s]", txID)
	}
	// Sign
	return signer.Sign(bytes)
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
