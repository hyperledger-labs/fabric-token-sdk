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
	"time"

	"github.com/IBM/idemix/bccsp/types"
	math "github.com/IBM/mathlib"
	"github.com/hyperledger-labs/fabric-smart-client/pkg/utils/errors"
	registry2 "github.com/hyperledger-labs/fabric-smart-client/platform/view/services/registry"
	"github.com/hyperledger-labs/fabric-token-sdk/token"
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
	"github.com/hyperledger-labs/fabric-token-sdk/token/services/identity/sig"
	"github.com/hyperledger-labs/fabric-token-sdk/token/services/identity/storage/kvs"
	ix509 "github.com/hyperledger-labs/fabric-token-sdk/token/services/identity/x509"
	"github.com/hyperledger-labs/fabric-token-sdk/token/services/logging"
	utils2 "github.com/hyperledger-labs/fabric-token-sdk/token/services/utils"
	"github.com/hyperledger-labs/fabric-token-sdk/token/services/utils/slices"
	token2 "github.com/hyperledger-labs/fabric-token-sdk/token/token"
	"github.com/hyperledger/fabric/bccsp/utils"
	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"
	"go.opentelemetry.io/otel/trace/noop"
)

var fakeLedger *mock.Ledger

var _ = Describe("validator", func() {
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
	BeforeEach(func() {
		fakeLedger = &mock.Ledger{}
		var err error
		// prepare public parameters
		ipk, err = os.ReadFile("./testdata/idemix/msp/IssuerPublicKey")
		Expect(err).NotTo(HaveOccurred())
		pp, err = v1.Setup(32, ipk, math.FP256BN_AMCL)
		Expect(err).NotTo(HaveOccurred())

		c := math.Curves[pp.Curve]

		asigner, _ := prepareECDSASigner()
		idemixDes, err := idemix2.NewDeserializer(slices.GetUnique(pp.IdemixIssuerPublicKeys).PublicKey, math.FP256BN_AMCL)
		Expect(err).NotTo(HaveOccurred())
		des := deserializer.NewTypedVerifierDeserializerMultiplex()
		des.AddTypedVerifierDeserializer(idemix2.IdentityType, deserializer.NewTypedIdentityVerifierDeserializer(idemixDes, idemixDes))
		des.AddTypedVerifierDeserializer(ix509.IdentityType, deserializer.NewTypedIdentityVerifierDeserializer(&Deserializer{}, &Deserializer{}))
		auditor = audit.NewAuditor(logging.MustGetLogger("auditor"), &noop.Tracer{}, des, pp.PedersenGenerators, asigner, c)
		araw, err := asigner.Serialize()
		Expect(err).NotTo(HaveOccurred())
		pp.Auditor = araw

		// initialize enginw with pp
		deserializer, err := zkatdlog.NewDeserializer(pp)
		Expect(err).NotTo(HaveOccurred())
		engine = enginedlog.New(logging.MustGetLogger("validator"), pp, deserializer)

		// non-anonymous issue
		_, ir, _ = prepareNonAnonymousIssueRequest(pp, auditor)
		Expect(ir).NotTo(BeNil())

		// prepare redeem
		sender, rr, _, inputsForRedeem = prepareRedeemRequest(pp, auditor)
		Expect(sender).NotTo(BeNil())

		// prepare transfer
		var trmetadata *driver.TokenRequestMetadata
		sender, tr, trmetadata, inputsForTransfer = prepareTransferRequest(pp, auditor)
		Expect(sender).NotTo(BeNil())
		Expect(trmetadata).NotTo(BeNil())

		// atomic action request
		ar = &driver.TokenRequest{Transfers: tr.Transfers}
		raw, err := ar.MarshalToMessageToSign([]byte("2"))
		Expect(err).NotTo(HaveOccurred())

		// sender signs request
		signatures, err := sender.SignTokenActions(raw)
		Expect(err).NotTo(HaveOccurred())

		// auditor inspect token
		metadata := &driver.TokenRequestMetadata{}
		metadata.Transfers = []*driver.TransferMetadata{trmetadata.Transfers[0]}

		tokns := make([][]*tokn.Token, 1)
		for i := 0; i < 2; i++ {
			tokns[0] = append(tokns[0], inputsForTransfer[i])
		}
		err = auditor.Check(context.Background(), ar, metadata, tokns, "2")
		Expect(err).NotTo(HaveOccurred())
		sigma, err := auditor.Endorse(ar, "2")
		Expect(err).NotTo(HaveOccurred())
		ar.AuditorSignatures = append(ar.AuditorSignatures, sigma)

		ar.Signatures = append(ar.Signatures, signatures...)
	})
	Describe("Verify Token Requests", func() {
		Context("Validator is called correctly with a non-anonymous issue action", func() {
			var (
				err error
				raw []byte
			)
			BeforeEach(func() {
				raw, err = ir.Bytes()
				Expect(err).NotTo(HaveOccurred())
			})
			It("succeeds", func() {
				actions, _, err := engine.VerifyTokenRequestFromRaw(context.TODO(), fakeLedger.GetStateStub, "1", raw)
				Expect(err).NotTo(HaveOccurred())
				Expect(len(actions)).To(Equal(1))
			})
		})

		Context("validator is called correctly with a transfer action", func() {
			var (
				err error
				raw []byte
			)
			BeforeEach(func() {
				raw, err = inputsForTransfer[0].Serialize()
				Expect(err).NotTo(HaveOccurred())
				fakeLedger.GetStateReturnsOnCall(0, raw, nil)

				raw, err = inputsForTransfer[1].Serialize()
				Expect(err).NotTo(HaveOccurred())
				fakeLedger.GetStateReturnsOnCall(1, raw, nil)

				raw, err = inputsForTransfer[0].Serialize()
				Expect(err).NotTo(HaveOccurred())
				fakeLedger.GetStateReturnsOnCall(2, raw, nil)

				raw, err = inputsForTransfer[1].Serialize()
				Expect(err).NotTo(HaveOccurred())
				fakeLedger.GetStateReturnsOnCall(3, raw, nil)

				fakeLedger.GetStateReturnsOnCall(4, nil, nil)
				fakeLedger.GetStateReturnsOnCall(5, nil, nil)

				raw, err = tr.Bytes()
				Expect(err).NotTo(HaveOccurred())
			})
			It("succeeds", func() {
				actions, _, err := engine.VerifyTokenRequestFromRaw(context.TODO(), getState, "1", raw)
				Expect(err).NotTo(HaveOccurred())
				Expect(len(actions)).To(Equal(1))
			})
		})
		Context("validator is called correctly with a redeem action", func() {
			var (
				err error
				raw []byte
			)
			BeforeEach(func() {

				raw, err = inputsForRedeem[0].Serialize()
				Expect(err).NotTo(HaveOccurred())
				fakeLedger.GetStateReturnsOnCall(0, raw, nil)

				raw, err = inputsForRedeem[1].Serialize()
				Expect(err).NotTo(HaveOccurred())
				fakeLedger.GetStateReturnsOnCall(1, raw, nil)

				raw, err = inputsForRedeem[0].Serialize()
				Expect(err).NotTo(HaveOccurred())
				fakeLedger.GetStateReturnsOnCall(2, raw, nil)

				raw, err = inputsForRedeem[1].Serialize()
				Expect(err).NotTo(HaveOccurred())
				fakeLedger.GetStateReturnsOnCall(3, raw, nil)

				fakeLedger.GetStateReturnsOnCall(4, nil, nil)

				raw, err = rr.Bytes()
				Expect(err).NotTo(HaveOccurred())

			})
			It("succeeds", func() {
				actions, _, err := engine.VerifyTokenRequestFromRaw(context.TODO(), getState, "1", raw)
				Expect(err).NotTo(HaveOccurred())
				Expect(len(actions)).To(Equal(1))
			})
		})
		Context("enginve is called correctly with atomic swap", func() {
			var (
				err error
				raw []byte
			)
			BeforeEach(func() {
				raw, err = inputsForTransfer[0].Serialize()
				Expect(err).NotTo(HaveOccurred())
				fakeLedger.GetStateReturnsOnCall(0, raw, nil)

				raw, err = inputsForTransfer[1].Serialize()
				Expect(err).NotTo(HaveOccurred())
				fakeLedger.GetStateReturnsOnCall(1, raw, nil)

				fakeLedger.GetStateReturnsOnCall(2, nil, nil)

				raw, err = inputsForTransfer[0].Serialize()
				Expect(err).NotTo(HaveOccurred())
				fakeLedger.GetStateReturnsOnCall(3, raw, nil)

				raw, err = inputsForTransfer[1].Serialize()
				Expect(err).NotTo(HaveOccurred())
				fakeLedger.GetStateReturnsOnCall(4, raw, nil)

				fakeLedger.GetStateReturnsOnCall(5, nil, nil)
				fakeLedger.GetStateReturnsOnCall(6, nil, nil)

				raw, err = ar.Bytes()
				Expect(err).NotTo(HaveOccurred())

			})
			It("succeeds", func() {
				actions, _, err := engine.VerifyTokenRequestFromRaw(context.TODO(), getState, "2", raw)
				Expect(err).NotTo(HaveOccurred())
				Expect(len(actions)).To(Equal(1))
			})

			Context("when the sender's signature is not valid: wrong txID", func() {
				BeforeEach(func() {
					request := &driver.TokenRequest{Issues: ar.Issues, Transfers: ar.Transfers}
					raw, err = request.MarshalToMessageToSign([]byte("3"))
					Expect(err).NotTo(HaveOccurred())

					signatures, err := sender.SignTokenActions(raw)
					Expect(err).NotTo(HaveOccurred())
					ar.Signatures[1] = signatures[0]

					raw, err = ar.Bytes()
					Expect(err).NotTo(HaveOccurred())

				})
				It("fails", func() {
					_, _, err := engine.VerifyTokenRequestFromRaw(context.TODO(), getState, "2", raw)
					Expect(err.Error()).To(ContainSubstring("pseudonym signature invalid"))

				})
			})
		})
	})
})

func prepareECDSASigner() (*Signer, *Verifier) {
	signer, err := NewECDSASigner()
	Expect(err).NotTo(HaveOccurred())
	return signer, signer.Verifier
}

func prepareNonAnonymousIssueRequest(pp *v1.PublicParams, auditor *audit.Auditor) (*issue2.Issuer, *driver.TokenRequest, *driver.TokenRequestMetadata) {
	signer, err := NewECDSASigner()
	Expect(err).NotTo(HaveOccurred())

	issuer := &issue2.Issuer{}
	issuer.New("ABC", signer, pp)
	issuerIdentity, err := signer.Serialize()
	Expect(err).NotTo(HaveOccurred())
	ir, metadata := prepareIssue(auditor, issuer, issuerIdentity)

	return issuer, ir, metadata
}

func prepareRedeemRequest(pp *v1.PublicParams, auditor *audit.Auditor) (*transfer.Sender, *driver.TokenRequest, *driver.TokenRequestMetadata, []*tokn.Token) {
	id, auditInfo, signer := getIdemixInfo("./testdata/idemix")
	owners := make([][]byte, 2)
	owners[0] = id

	return prepareTransfer(pp, signer, auditor, auditInfo, id, owners)
}

func prepareTransferRequest(pp *v1.PublicParams, auditor *audit.Auditor) (*transfer.Sender, *driver.TokenRequest, *driver.TokenRequestMetadata, []*tokn.Token) {
	id, auditInfo, signer := getIdemixInfo("./testdata/idemix")
	owners := make([][]byte, 2)
	owners[0] = id
	owners[1] = id

	return prepareTransfer(pp, signer, auditor, auditInfo, id, owners)
}

func prepareTokens(values, bf []*math.Zr, ttype string, pp []*math.G1, curve *math.Curve) []*math.G1 {
	tokens := make([]*math.G1, len(values))
	for i := 0; i < len(values); i++ {
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

type fakeProv struct {
	typ string
}

func (f *fakeProv) GetString(key string) string {
	return f.typ
}

func (f *fakeProv) GetInt(key string) int {
	return 0
}

func (f *fakeProv) GetDuration(key string) time.Duration {
	return time.Duration(0)
}

func (f *fakeProv) GetBool(key string) bool {
	return false
}

func (f *fakeProv) GetStringSlice(key string) []string {
	return nil
}

func (f *fakeProv) IsSet(key string) bool {
	return false
}

func (f *fakeProv) UnmarshalKey(key string, rawVal interface{}) error {
	return nil
}

func (f *fakeProv) ConfigFileUsed() string {
	return ""
}

func (f *fakeProv) GetPath(key string) string {
	return ""
}

func (f *fakeProv) TranslatePath(path string) string {
	return ""
}

func getIdemixInfo(dir string) (driver.Identity, *crypto.AuditInfo, driver.SigningIdentity) {
	registry := registry2.New()
	configService := &fakeProv{typ: "memory"}
	Expect(registry.RegisterService(configService)).NotTo(HaveOccurred())

	backend, err := kvs.NewInMemory()
	Expect(err).NotTo(HaveOccurred())
	err = registry.RegisterService(backend)
	Expect(err).NotTo(HaveOccurred())

	sigService := sig.NewService(sig.NewMultiplexDeserializer(), kvs.NewIdentityDB(backend, token.TMSID{Network: "pineapple"}))
	err = registry.RegisterService(sigService)
	Expect(err).NotTo(HaveOccurred())
	config, err := crypto.NewConfig(dir)
	Expect(err).NotTo(HaveOccurred())

	keyStore, err := crypto.NewKeyStore(math.FP256BN_AMCL, backend)
	Expect(err).NotTo(HaveOccurred())
	cryptoProvider, err := crypto.NewBCCSP(keyStore, math.FP256BN_AMCL, false)
	Expect(err).NotTo(HaveOccurred())
	p, err := idemix2.NewKeyManager(config, sigService, types.EidNymRhNym, cryptoProvider)
	Expect(err).NotTo(HaveOccurred())
	Expect(p).NotTo(BeNil())

	id, audit, err := p.Identity(nil)
	Expect(err).NotTo(HaveOccurred())
	Expect(id).NotTo(BeNil())
	Expect(audit).NotTo(BeNil())

	auditInfo, err := p.DeserializeAuditInfo(audit)
	Expect(err).NotTo(HaveOccurred())
	err = auditInfo.Match(id)
	Expect(err).NotTo(HaveOccurred())

	signer, err := p.DeserializeSigningIdentity(id)
	Expect(err).NotTo(HaveOccurred())

	id, err = identity.WrapWithType(idemix2.IdentityType, id)
	Expect(err).NotTo(HaveOccurred())

	return id, auditInfo, signer
}

func prepareIssue(auditor *audit.Auditor, issuer *issue2.Issuer, issuerIdentity []byte) (*driver.TokenRequest, *driver.TokenRequestMetadata) {
	id, auditInfo, _ := getIdemixInfo("./testdata/idemix")
	ir := &driver.TokenRequest{}
	owners := make([][]byte, 1)
	owners[0] = id
	values := []uint64{40}

	issue, inf, err := issuer.GenerateZKIssue(values, owners)
	Expect(err).NotTo(HaveOccurred())

	auditInfoRaw, err := auditInfo.Bytes()
	Expect(err).NotTo(HaveOccurred())
	metadata := &driver.IssueMetadata{
		Issuer: driver.AuditableIdentity{
			Identity:  issuerIdentity,
			AuditInfo: issuerIdentity,
		},
	}
	for i := 0; i < len(issue.Outputs); i++ {
		marshalledinf, err := inf[i].Serialize()
		Expect(err).NotTo(HaveOccurred())
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
	Expect(err).NotTo(HaveOccurred())

	// sign token request
	ir = &driver.TokenRequest{Issues: [][]byte{raw}}
	raw, err = ir.MarshalToMessageToSign([]byte("1"))
	Expect(err).NotTo(HaveOccurred())

	sig, err := issuer.SignTokenActions(raw)
	Expect(err).NotTo(HaveOccurred())
	ir.Signatures = append(ir.Signatures, sig)

	issueMetadata := &driver.TokenRequestMetadata{Issues: []*driver.IssueMetadata{metadata}}
	err = auditor.Check(context.Background(), ir, issueMetadata, nil, "1")
	Expect(err).NotTo(HaveOccurred())
	sigma, err := auditor.Endorse(ir, "1")
	Expect(err).NotTo(HaveOccurred())
	ir.AuditorSignatures = append(ir.AuditorSignatures, sigma)

	return ir, issueMetadata
}

func prepareTransfer(pp *v1.PublicParams, signer driver.SigningIdentity, auditor *audit.Auditor, auditInfo *crypto.AuditInfo, id []byte, owners [][]byte) (*transfer.Sender, *driver.TokenRequest, *driver.TokenRequestMetadata, []*tokn.Token) {

	signers := make([]driver.Signer, 2)
	signers[0] = signer
	signers[1] = signer
	c := math.Curves[pp.Curve]

	invalues := make([]*math.Zr, 2)
	invalues[0] = c.NewZrFromInt(70)
	invalues[1] = c.NewZrFromInt(30)

	inBF := make([]*math.Zr, 2)
	rand, err := c.Rand()
	Expect(err).NotTo(HaveOccurred())
	for i := 0; i < 2; i++ {
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
	Expect(err).NotTo(HaveOccurred())

	transfer, metas, err := sender.GenerateZKTransfer(context.TODO(), outvalues, owners)
	Expect(err).NotTo(HaveOccurred())

	transferRaw, err := transfer.Serialize()
	Expect(err).NotTo(HaveOccurred())

	tr := &driver.TokenRequest{Transfers: [][]byte{transferRaw}}
	raw, err := tr.MarshalToMessageToSign([]byte("1"))
	Expect(err).NotTo(HaveOccurred())

	marshalledInfo := make([][]byte, len(metas))
	for i := 0; i < len(metas); i++ {
		marshalledInfo[i], err = metas[i].Serialize()
		Expect(err).NotTo(HaveOccurred())
	}
	auditInfoRaw, err := auditInfo.Bytes()
	Expect(err).NotTo(HaveOccurred())
	metadata := &driver.TransferMetadata{}
	for i := 0; i < len(transfer.Inputs); i++ {
		metadata.Inputs = append(metadata.Inputs, &driver.TransferInputMetadata{
			TokenID: nil,
			Senders: []*driver.AuditableIdentity{
				{
					Identity:  nil,
					AuditInfo: auditInfoRaw,
				},
			},
		})
		Expect(err).NotTo(HaveOccurred())
	}

	for i := 0; i < len(transfer.Outputs); i++ {
		marshalledinf, err := metas[i].Serialize()
		Expect(err).NotTo(HaveOccurred())
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
	for i := 0; i < len(tokens); i++ {
		tokns[0] = append(tokns[0], tokens[i])
	}
	transferMetadata := &driver.TokenRequestMetadata{Transfers: []*driver.TransferMetadata{metadata}}
	err = auditor.Check(context.Background(), tr, transferMetadata, tokns, "1")
	Expect(err).NotTo(HaveOccurred())

	sigma, err := auditor.Endorse(tr, "1")
	Expect(err).NotTo(HaveOccurred())
	tr.AuditorSignatures = append(tr.AuditorSignatures, sigma)

	signatures, err := sender.SignTokenActions(raw)
	Expect(err).NotTo(HaveOccurred())
	tr.Signatures = append(tr.Signatures, signatures...)

	return sender, tr, transferMetadata, tokens
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

func (d *Deserializer) Match(id []byte) error {
	identity, err := identity.WrapWithType(ix509.IdentityType, id)
	if err != nil {
		return errors.Wrapf(err, "failed to unmarshal identity [%s]", id)
	}
	if !bytes.Equal(d.auditInfo, identity) {
		return errors.Errorf("identity mismatch [%s][%s]", utils2.Hashable(identity), utils2.Hashable(d.auditInfo))
	}
	return nil
}

func (d *Deserializer) GetAuditInfoMatcher(owner driver.Identity, auditInfo []byte) (driver.Matcher, error) {
	return &Deserializer{auditInfo: auditInfo}, nil
}

func (d *Deserializer) DeserializeVerifier(id driver.Identity) (driver.Verifier, error) {
	panic("implement me")
}
