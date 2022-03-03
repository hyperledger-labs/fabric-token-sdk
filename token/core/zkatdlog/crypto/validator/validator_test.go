/*
Copyright IBM Corp. All Rights Reserved.

SPDX-License-Identifier: Apache-2.0
*/

package validator_test

import (
	"encoding/asn1"
	"encoding/json"
	"io/ioutil"
	"time"

	math "github.com/IBM/mathlib"
	idemix2 "github.com/hyperledger-labs/fabric-smart-client/platform/fabric/core/generic/msp/idemix"
	"github.com/hyperledger-labs/fabric-smart-client/platform/view/core/sig"
	_ "github.com/hyperledger-labs/fabric-smart-client/platform/view/services/db/driver/memory"
	"github.com/hyperledger-labs/fabric-smart-client/platform/view/services/kvs"
	registry2 "github.com/hyperledger-labs/fabric-smart-client/platform/view/services/registry"
	"github.com/hyperledger-labs/fabric-smart-client/platform/view/view"
	msp2 "github.com/hyperledger/fabric/msp"
	. "github.com/onsi/ginkgo"
	. "github.com/onsi/gomega"

	"github.com/hyperledger-labs/fabric-token-sdk/token/core/identity"
	"github.com/hyperledger-labs/fabric-token-sdk/token/core/zkatdlog/crypto"
	"github.com/hyperledger-labs/fabric-token-sdk/token/core/zkatdlog/crypto/audit"
	"github.com/hyperledger-labs/fabric-token-sdk/token/core/zkatdlog/crypto/ecdsa"
	issue2 "github.com/hyperledger-labs/fabric-token-sdk/token/core/zkatdlog/crypto/issue"
	"github.com/hyperledger-labs/fabric-token-sdk/token/core/zkatdlog/crypto/issue/nonanonym"
	tokn "github.com/hyperledger-labs/fabric-token-sdk/token/core/zkatdlog/crypto/token"
	"github.com/hyperledger-labs/fabric-token-sdk/token/core/zkatdlog/crypto/transfer"
	enginedlog "github.com/hyperledger-labs/fabric-token-sdk/token/core/zkatdlog/crypto/validator"
	"github.com/hyperledger-labs/fabric-token-sdk/token/core/zkatdlog/crypto/validator/mock"
	zkatdlog "github.com/hyperledger-labs/fabric-token-sdk/token/core/zkatdlog/nogh"
	"github.com/hyperledger-labs/fabric-token-sdk/token/driver"
)

var fakeldger *mock.Ledger

type idemix interface {
	DeserializeAuditInfo(raw []byte) (*idemix2.AuditInfo, error)
}

type deserializer struct {
	idemix idemix
}

func (d *deserializer) GetOwnerMatcher(raw []byte) (driver.Matcher, error) {
	return d.idemix.DeserializeAuditInfo(raw)
}

var _ = Describe("validator", func() {
	var (
		engine *enginedlog.Validator
		pp     *crypto.PublicParams

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
		fakeldger = &mock.Ledger{}
		var err error
		// prepare public parameters
		ipk, err = ioutil.ReadFile("./testdata/idemix/msp/IssuerPublicKey")
		Expect(err).NotTo(HaveOccurred())
		pp, err = crypto.Setup(100, 2, ipk, math.FP256BN_AMCL)
		Expect(err).NotTo(HaveOccurred())

		c := math.Curves[pp.Curve]

		asigner, _ := prepareECDSASigner()
		des, err := idemix2.NewDeserializer(pp.IdemixPK)
		Expect(err).NotTo(HaveOccurred())
		auditor = audit.NewAuditor(&deserializer{idemix: des}, pp.ZKATPedParams, pp.IdemixPK, asigner, c)
		araw, err := asigner.Serialize()
		Expect(err).NotTo(HaveOccurred())
		pp.Auditor = araw

		// initialize enginw with pp
		deserializer, err := zkatdlog.NewDeserializer(pp)
		Expect(err).NotTo(HaveOccurred())
		engine = enginedlog.New(pp, deserializer)

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
		raw, err := asn1.Marshal(*ar)
		Expect(err).NotTo(HaveOccurred())

		// sender signs request
		signatures, err := sender.SignTokenActions(raw, "2")
		Expect(err).NotTo(HaveOccurred())

		// auditor inspect token
		metadata := &driver.TokenRequestMetadata{}
		metadata.Transfers = []driver.TransferMetadata{trmetadata.Transfers[0]}

		tokns := make([][]*tokn.Token, 1)
		for i := 0; i < 2; i++ {
			tokns[0] = append(tokns[0], inputsForTransfer[i])
		}
		err = auditor.Check(ar, metadata, tokns, "2")
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
				raw, err = asn1.Marshal(*ir)
				Expect(err).NotTo(HaveOccurred())
			})
			It("succeeds", func() {
				actions, err := engine.VerifyTokenRequestFromRaw(fakeldger.GetStateStub, "1", raw)
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
				fakeldger.GetStateReturnsOnCall(0, raw, nil)

				raw, err = inputsForTransfer[1].Serialize()
				Expect(err).NotTo(HaveOccurred())
				fakeldger.GetStateReturnsOnCall(1, raw, nil)

				raw, err = inputsForTransfer[0].Serialize()
				Expect(err).NotTo(HaveOccurred())
				fakeldger.GetStateReturnsOnCall(2, raw, nil)

				raw, err = inputsForTransfer[1].Serialize()
				Expect(err).NotTo(HaveOccurred())
				fakeldger.GetStateReturnsOnCall(3, raw, nil)

				fakeldger.GetStateReturnsOnCall(4, nil, nil)
				fakeldger.GetStateReturnsOnCall(5, nil, nil)

				raw, err = asn1.Marshal(*tr)
				Expect(err).NotTo(HaveOccurred())
			})
			It("succeeds", func() {
				actions, err := engine.VerifyTokenRequestFromRaw(getState, "1", raw)
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
				fakeldger.GetStateReturnsOnCall(0, raw, nil)

				raw, err = inputsForRedeem[1].Serialize()
				Expect(err).NotTo(HaveOccurred())
				fakeldger.GetStateReturnsOnCall(1, raw, nil)

				raw, err = inputsForRedeem[0].Serialize()
				Expect(err).NotTo(HaveOccurred())
				fakeldger.GetStateReturnsOnCall(2, raw, nil)

				raw, err = inputsForRedeem[1].Serialize()
				Expect(err).NotTo(HaveOccurred())
				fakeldger.GetStateReturnsOnCall(3, raw, nil)

				fakeldger.GetStateReturnsOnCall(4, nil, nil)

				raw, err = asn1.Marshal(*rr)
				Expect(err).NotTo(HaveOccurred())

			})
			It("succeeds", func() {
				actions, err := engine.VerifyTokenRequestFromRaw(getState, "1", raw)
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
				fakeldger.GetStateReturnsOnCall(0, raw, nil)

				raw, err = inputsForTransfer[1].Serialize()
				Expect(err).NotTo(HaveOccurred())
				fakeldger.GetStateReturnsOnCall(1, raw, nil)

				fakeldger.GetStateReturnsOnCall(2, nil, nil)

				raw, err = inputsForTransfer[0].Serialize()
				Expect(err).NotTo(HaveOccurred())
				fakeldger.GetStateReturnsOnCall(3, raw, nil)

				raw, err = inputsForTransfer[1].Serialize()
				Expect(err).NotTo(HaveOccurred())
				fakeldger.GetStateReturnsOnCall(4, raw, nil)

				fakeldger.GetStateReturnsOnCall(5, nil, nil)
				fakeldger.GetStateReturnsOnCall(6, nil, nil)

				raw, err = asn1.Marshal(*ar)
				Expect(err).NotTo(HaveOccurred())

			})
			It("succeeds", func() {
				actions, err := engine.VerifyTokenRequestFromRaw(getState, "2", raw)
				Expect(err).NotTo(HaveOccurred())
				Expect(len(actions)).To(Equal(1))
			})

			Context("when the sender's signature is not valid: wrong txID", func() {
				BeforeEach(func() {
					request := &driver.TokenRequest{Issues: ar.Issues, Transfers: ar.Transfers}
					raw, err = asn1.Marshal(*request)
					Expect(err).NotTo(HaveOccurred())

					signatures, err := sender.SignTokenActions(raw, "3")
					Expect(err).NotTo(HaveOccurred())
					ar.Signatures[1] = signatures[0]

					raw, err = asn1.Marshal(*ar)
					Expect(err).NotTo(HaveOccurred())

				})
				It("fails", func() {
					_, err := engine.VerifyTokenRequestFromRaw(getState, "2", raw)
					Expect(err.Error()).To(ContainSubstring("pseudonym signature invalid"))

				})
			})
		})
	})
})

func prepareECDSASigner() (*ecdsa.ECDSASigner, *ecdsa.ECDSAVerifier) {
	signer, err := ecdsa.NewECDSASigner()
	Expect(err).NotTo(HaveOccurred())
	return signer, signer.ECDSAVerifier
}

func prepareNonAnonymousIssueRequest(pp *crypto.PublicParams, auditor *audit.Auditor) (*nonanonym.Issuer, *driver.TokenRequest, *driver.TokenRequestMetadata) {
	signer, err := ecdsa.NewECDSASigner()
	Expect(err).NotTo(HaveOccurred())

	issuer := &nonanonym.Issuer{}
	issuer.New("ABC", signer, pp)
	ir, metadata := prepareIssue(auditor, issuer)

	return issuer, ir, metadata
}

func prepareRedeemRequest(pp *crypto.PublicParams, auditor *audit.Auditor) (*transfer.Sender, *driver.TokenRequest, *driver.TokenRequestMetadata, []*tokn.Token) {
	id, auditInfo, signer := getIdemixInfo("./testdata/idemix")
	owners := make([][]byte, 2)
	owners[0] = id

	return prepareTransfer(pp, signer, auditor, auditInfo, id, owners)
}

func prepareTransferRequest(pp *crypto.PublicParams, auditor *audit.Auditor) (*transfer.Sender, *driver.TokenRequest, *driver.TokenRequestMetadata, []*tokn.Token) {
	id, auditInfo, signer := getIdemixInfo("./testdata/idemix")
	owners := make([][]byte, 2)
	owners[0] = id
	owners[1] = id

	return prepareTransfer(pp, signer, auditor, auditInfo, id, owners)
}

func getIssuers(N, index int, pk *math.G1, pp []*math.G1, curve *math.Curve) []*math.G1 {
	rand, err := curve.Rand()
	Expect(err).NotTo(HaveOccurred())
	issuers := make([]*math.G1, N)
	issuers[index] = pk
	for i := 0; i < N; i++ {
		if i != index {
			sk := curve.NewRandomZr(rand)
			t := curve.NewRandomZr(rand)
			issuers[i] = pp[0].Mul(sk)
			issuers[i].Add(pp[1].Mul(t))
		}
	}

	return issuers

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
	typ  string
	path string
}

func (f *fakeProv) GetString(key string) string {
	return f.typ
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
	*(rawVal.(*kvs.Opts)) = kvs.Opts{
		Path: f.path,
	}

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

func getIdemixInfo(dir string) (view.Identity, *idemix2.AuditInfo, driver.SigningIdentity) {
	registry := registry2.New()
	registry.RegisterService(&fakeProv{typ: "memory"})

	kvss, err := kvs.New("memory", "", registry)
	Expect(err).NotTo(HaveOccurred())
	err = registry.RegisterService(kvss)
	Expect(err).NotTo(HaveOccurred())

	sigService := sig.NewSignService(registry, nil, kvss)
	err = registry.RegisterService(sigService)
	Expect(err).NotTo(HaveOccurred())
	config, err := msp2.GetLocalMspConfigWithType(dir, nil, "idemix", "idemix")
	Expect(err).NotTo(HaveOccurred())

	p, err := idemix2.NewEIDNymProvider(config, registry)
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

	id, err = identity.MarshallRawOwner(&identity.RawOwner{Identity: id, Type: identity.SerializedIdentityType})
	Expect(err).NotTo(HaveOccurred())

	return id, auditInfo, signer
}

func prepareIssue(auditor *audit.Auditor, issuer issue2.Issuer) (*driver.TokenRequest, *driver.TokenRequestMetadata) {
	id, auditInfo, _ := getIdemixInfo("./testdata/idemix")
	ir := &driver.TokenRequest{}
	owners := make([][]byte, 1)
	owners[0] = id
	values := []uint64{40}

	issue, inf, err := issuer.GenerateZKIssue(values, owners)
	Expect(err).NotTo(HaveOccurred())

	marshalledinf := make([][]byte, len(inf))
	for i := 0; i < len(inf); i++ {
		marshalledinf[i], err = inf[i].Serialize()
		Expect(err).NotTo(HaveOccurred())
	}

	metadata := driver.IssueMetadata{}
	metadata.TokenInfo = marshalledinf
	metadata.Outputs = make([][]byte, len(issue.OutputTokens))
	metadata.AuditInfos = make([][]byte, len(issue.OutputTokens))
	for i := 0; i < len(issue.OutputTokens); i++ {
		metadata.Outputs[i], err = json.Marshal(issue.OutputTokens[i].Data)
		Expect(err).NotTo(HaveOccurred())
		metadata.AuditInfos[i], err = auditInfo.Bytes()
		Expect(err).NotTo(HaveOccurred())
	}

	// serialize token action
	raw, err := issue.Serialize()
	Expect(err).NotTo(HaveOccurred())

	// sign token request
	ir = &driver.TokenRequest{Issues: [][]byte{raw}}
	raw, err = asn1.Marshal(*ir)
	Expect(err).NotTo(HaveOccurred())

	sig, err := issuer.SignTokenActions(raw, "1")
	Expect(err).NotTo(HaveOccurred())
	ir.Signatures = append(ir.Signatures, sig)

	issueMetadata := &driver.TokenRequestMetadata{Issues: []driver.IssueMetadata{metadata}}
	err = auditor.Check(ir, issueMetadata, nil, "1")
	Expect(err).NotTo(HaveOccurred())
	sigma, err := auditor.Endorse(ir, "1")
	Expect(err).NotTo(HaveOccurred())
	ir.AuditorSignatures = append(ir.AuditorSignatures, sigma)

	return ir, issueMetadata
}

func prepareTransfer(pp *crypto.PublicParams, signer driver.SigningIdentity, auditor *audit.Auditor, auditInfo *idemix2.AuditInfo, id []byte, owners [][]byte) (*transfer.Sender, *driver.TokenRequest, *driver.TokenRequestMetadata, []*tokn.Token) {

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

	ids := make([]string, 2)
	ids[0] = "0"
	ids[1] = "1"

	inputs := prepareTokens(invalues, inBF, "ABC", pp.ZKATPedParams, c)
	tokens := make([]*tokn.Token, 2)
	tokens[0] = &tokn.Token{Data: inputs[0], Owner: id}
	tokens[1] = &tokn.Token{Data: inputs[1], Owner: id}

	inputInf := make([]*tokn.TokenInformation, 2)
	inputInf[0] = &tokn.TokenInformation{Type: "ABC", Value: invalues[0], BlindingFactor: inBF[0]}
	inputInf[1] = &tokn.TokenInformation{Type: "ABC", Value: invalues[1], BlindingFactor: inBF[1]}
	sender, err := transfer.NewSender(signers, tokens, ids, inputInf, pp)
	Expect(err).NotTo(HaveOccurred())

	transfer, inf, err := sender.GenerateZKTransfer(outvalues, owners)
	Expect(err).NotTo(HaveOccurred())

	raw, err := transfer.Serialize()
	Expect(err).NotTo(HaveOccurred())

	tr := &driver.TokenRequest{Transfers: [][]byte{raw}}
	raw, err = asn1.Marshal(*tr)
	Expect(err).NotTo(HaveOccurred())

	marshalledInfo := make([][]byte, len(inf))
	for i := 0; i < len(inf); i++ {
		marshalledInfo[i], err = json.Marshal(inf[i])
		Expect(err).NotTo(HaveOccurred())
	}
	metadata := driver.TransferMetadata{}
	metadata.SenderAuditInfos = make([][]byte, len(transfer.Inputs))
	for i := 0; i < len(transfer.Inputs); i++ {
		metadata.SenderAuditInfos[i], err = auditInfo.Bytes()
		Expect(err).NotTo(HaveOccurred())
	}

	metadata.TokenInfo = marshalledInfo
	metadata.Outputs = make([][]byte, len(transfer.OutputTokens))
	metadata.ReceiverAuditInfos = make([][]byte, len(transfer.OutputTokens))
	for i := 0; i < len(transfer.OutputTokens); i++ {
		metadata.Outputs[i], err = json.Marshal(transfer.OutputTokens[i].Data)
		Expect(err).NotTo(HaveOccurred())
		metadata.ReceiverAuditInfos[i], err = auditInfo.Bytes()
		Expect(err).NotTo(HaveOccurred())
	}

	tokns := make([][]*tokn.Token, 1)
	for i := 0; i < len(tokens); i++ {
		tokns[0] = append(tokns[0], tokens[i])
	}
	transferMetadata := &driver.TokenRequestMetadata{Transfers: []driver.TransferMetadata{metadata}}
	err = auditor.Check(tr, transferMetadata, tokns, "1")
	Expect(err).NotTo(HaveOccurred())

	sigma, err := auditor.Endorse(tr, "1")
	Expect(err).NotTo(HaveOccurred())
	tr.AuditorSignatures = append(tr.AuditorSignatures, sigma)

	signatures, err := sender.SignTokenActions(raw, "1")
	Expect(err).NotTo(HaveOccurred())
	tr.Signatures = append(tr.Signatures, signatures...)

	return sender, tr, transferMetadata, tokens
}

func getState(key string) ([]byte, error) {
	return fakeldger.GetState(key)
}
