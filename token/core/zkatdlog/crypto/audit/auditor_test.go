/*
Copyright IBM Corp. All Rights Reserved.

SPDX-License-Identifier: Apache-2.0
*/
package audit_test

import (
	"encoding/json"
	"time"

	msp2 "github.com/hyperledger/fabric/msp"
	. "github.com/onsi/ginkgo"
	. "github.com/onsi/gomega"

	idemix2 "github.com/hyperledger-labs/fabric-smart-client/platform/fabric/core/generic/msp/idemix"
	sig2 "github.com/hyperledger-labs/fabric-smart-client/platform/view/core/sig"
	_ "github.com/hyperledger-labs/fabric-smart-client/platform/view/services/db/driver/memory"
	"github.com/hyperledger-labs/fabric-smart-client/platform/view/services/kvs"
	registry2 "github.com/hyperledger-labs/fabric-smart-client/platform/view/services/registry"
	"github.com/hyperledger-labs/fabric-smart-client/platform/view/view"

	"github.com/hyperledger-labs/fabric-token-sdk/token/core/math/gurvy/bn256"
	"github.com/hyperledger-labs/fabric-token-sdk/token/core/zkatdlog/crypto"
	"github.com/hyperledger-labs/fabric-token-sdk/token/core/zkatdlog/crypto/audit"
	"github.com/hyperledger-labs/fabric-token-sdk/token/core/zkatdlog/crypto/audit/mock"
	"github.com/hyperledger-labs/fabric-token-sdk/token/core/zkatdlog/crypto/common"
	"github.com/hyperledger-labs/fabric-token-sdk/token/core/zkatdlog/crypto/issue"
	"github.com/hyperledger-labs/fabric-token-sdk/token/core/zkatdlog/crypto/issue/anonym"
	"github.com/hyperledger-labs/fabric-token-sdk/token/core/zkatdlog/crypto/token"
	transfer2 "github.com/hyperledger-labs/fabric-token-sdk/token/core/zkatdlog/crypto/transfer"
	"github.com/hyperledger-labs/fabric-token-sdk/token/driver"
)

var _ = Describe("Auditor", func() {
	var (
		fakeSigningIdentity *mock.SigningIdentity
		pp                  *crypto.PublicParams
		auditor             *audit.Auditor
	)
	BeforeEach(func() {
		var err error
		fakeSigningIdentity = &mock.SigningIdentity{}
		pp, err = crypto.Setup(100, 2, nil)
		Expect(err).NotTo(HaveOccurred())
		auditor = audit.NewAuditor(pp.ZKATPedParams, nil, fakeSigningIdentity)
		fakeSigningIdentity.SignReturns([]byte("auditor-signature"), nil)

	})

	Describe("Audit an Issue", func() {
		When("audit information is computed correctly", func() {
			It("succeeds", func() {
				issue, metadata := createIssue(pp)
				raw, err := issue.Serialize()
				Expect(err).NotTo(HaveOccurred())
				err = auditor.Check(&driver.TokenRequest{Issues: [][]byte{raw}}, &driver.TokenRequestMetadata{Issues: []driver.IssueMetadata{metadata}}, nil, "1")
				Expect(err).NotTo(HaveOccurred())
				sig, err := auditor.Endorse(&driver.TokenRequest{Issues: [][]byte{raw}}, "1")
				Expect(err).NotTo(HaveOccurred())
				Expect(sig).To(Equal([]byte("auditor-signature")))
			})
		})
		When("when the token information does not match the outputs in issue ", func() {
			It("no signature is generated", func() {
				issue, metadata := createBogusIssue(pp)
				raw, err := issue.Serialize()
				Expect(err).NotTo(HaveOccurred())
				err = auditor.Check(&driver.TokenRequest{Issues: [][]byte{raw}}, &driver.TokenRequestMetadata{Issues: []driver.IssueMetadata{metadata}}, nil, "1")
				Expect(err).To(HaveOccurred())
				Expect(err.Error()).To(ContainSubstring("output at index [0] does not match the provided opening"))
			})
		})
	})
	Describe("Audit a transfer", func() {
		When("audit information is computed correctly", func() {
			It("succeeds", func() {
				transfer, metadata, tokens := createTransfer(pp)
				raw, err := transfer.Serialize()
				Expect(err).NotTo(HaveOccurred())
				err = auditor.Check(&driver.TokenRequest{Transfers: [][]byte{raw}}, &driver.TokenRequestMetadata{Transfers: []driver.TransferMetadata{metadata}}, tokens, "1")
				Expect(err).NotTo(HaveOccurred())
				sig, err := auditor.Endorse(&driver.TokenRequest{Transfers: [][]byte{raw}}, "1")
				Expect(err).NotTo(HaveOccurred())
				Expect(sig).To(Equal([]byte("auditor-signature")))
			})
		})
		When("token info does not match output", func() {
			It("fails", func() {
				transfer, metadata, tokens := createTransferWithBogusOutput(pp)
				raw, err := transfer.Serialize()
				Expect(err).NotTo(HaveOccurred())
				err = auditor.Check(&driver.TokenRequest{Transfers: [][]byte{raw}}, &driver.TokenRequestMetadata{Transfers: []driver.TransferMetadata{metadata}}, tokens, "1")
				Expect(err).To(HaveOccurred())
				Expect(fakeSigningIdentity.SignCallCount()).To(Equal(0))
			})
		})
		When("sender audit info does not match input", func() {
			It("fails", func() {
				transfer, metadata, tokens := createTransfer(pp)
				// test idemix info
				_, auditinfo := getIdemixInfo("./testdata/idemix")
				raw, err := auditinfo.Bytes()
				Expect(err).NotTo(HaveOccurred())
				metadata.SenderAuditInfos[0] = raw
				raw, err = transfer.Serialize()
				Expect(err).NotTo(HaveOccurred())
				err = auditor.Check(&driver.TokenRequest{Transfers: [][]byte{raw}}, &driver.TokenRequestMetadata{Transfers: []driver.TransferMetadata{metadata}}, tokens, "1")
				Expect(err).To(HaveOccurred())
				Expect(err.Error()).To(ContainSubstring("input at index [0] does not match the provided opening"))
				Expect(err.Error()).NotTo(ContainSubstring("attribute mistmatch"))
				Expect(fakeSigningIdentity.SignCallCount()).To(Equal(0))
			})
		})
		When("recipient audit info does not match output", func() {
			It("fails", func() {
				transfer, metadata, tokens := createTransfer(pp)
				// test idemix info
				_, auditinfo := getIdemixInfo("./testdata/idemix")
				raw, err := auditinfo.Bytes()
				Expect(err).NotTo(HaveOccurred())
				metadata.ReceiverAuditInfos[0] = raw
				raw, err = transfer.Serialize()
				Expect(err).NotTo(HaveOccurred())
				err = auditor.Check(&driver.TokenRequest{Transfers: [][]byte{raw}}, &driver.TokenRequestMetadata{Transfers: []driver.TransferMetadata{metadata}}, tokens, "1")
				Expect(err).To(HaveOccurred())
				Expect(err.Error()).To(ContainSubstring("output at index [0] does not match the provided opening"))
				Expect(err.Error()).To(ContainSubstring("attribute mistmatch"))
				Expect(fakeSigningIdentity.SignCallCount()).To(Equal(0))
			})
		})
	})
})

func createIssue(pp *crypto.PublicParams) (*issue.IssueAction, driver.IssueMetadata) {
	issuer := prepareIssuer(pp)
	id, auditInfo := getIdemixInfo("./testdata/idemix")

	issue, inf, err := issuer.GenerateZKIssue([]uint64{50, 20}, [][]byte{id, id})
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

	return issue, metadata
}

func createTransfer(pp *crypto.PublicParams) (*transfer2.TransferAction, driver.TransferMetadata, [][]*token.Token) {
	id, auditInfo := getIdemixInfo("./testdata/idemix")
	transfer, inf, inputs := prepareTransfer(pp, id)

	marshalledInfo := make([][]byte, len(inf))
	var err error
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
	tokns := make([][]*token.Token, 1)
	for i := 0; i < len(inputs); i++ {
		tokns[0] = append(tokns[0], inputs[i])
	}
	return transfer, metadata, tokns
}

func createBogusIssue(pp *crypto.PublicParams) (*issue.IssueAction, driver.IssueMetadata) {
	issuer := prepareIssuer(pp)
	id, auditInfo := getIdemixInfo("./testdata/idemix")

	issue, inf, err := issuer.GenerateZKIssue([]uint64{50, 20}, [][]byte{id, id})
	Expect(err).NotTo(HaveOccurred())

	// change value
	inf[0].Value = bn256.NewZrInt(15)
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
	inf[0].Value = bn256.NewZrInt(25)

	return issue, metadata
}

func createTransferWithBogusOutput(pp *crypto.PublicParams) (*transfer2.TransferAction, driver.TransferMetadata, [][]*token.Token) {
	id, auditInfo := getIdemixInfo("./testdata/idemix")
	transfer, inf, inputs := prepareTransfer(pp, id)

	inf[0].Value = bn256.NewZrInt(15)
	marshalledInfo := make([][]byte, len(inf))
	var err error
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

	tokns := make([][]*token.Token, 1)
	for i := 0; i < len(inputs); i++ {
		tokns[0] = append(tokns[0], inputs[i])
	}

	return transfer, metadata, tokns
}

func getIssuers(N, index int, pk *bn256.G1, pp []*bn256.G1) []*bn256.G1 {
	rand, err := bn256.GetRand()
	Expect(err).NotTo(HaveOccurred())
	issuers := make([]*bn256.G1, N)
	issuers[index] = pk
	for i := 0; i < N; i++ {
		if i != index {
			sk := bn256.RandModOrder(rand)
			t := bn256.RandModOrder(rand)
			issuers[i] = pp[0].Mul(sk)
			issuers[i].Add(pp[1].Mul(t))
		}
	}
	return issuers
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

func getIdemixInfo(dir string) (view.Identity, *idemix2.AuditInfo) {
	registry := registry2.New()
	registry.RegisterService(&fakeProv{typ: "memory"})

	kvss, err := kvs.New("memory", "", registry)
	Expect(err).NotTo(HaveOccurred())
	err = registry.RegisterService(kvss)
	Expect(err).NotTo(HaveOccurred())

	sigService := sig2.NewSignService(registry, nil)
	err = registry.RegisterService(sigService)
	Expect(err).NotTo(HaveOccurred())
	config, err := msp2.GetLocalMspConfigWithType(dir, nil, "idemix", "idemix")
	Expect(err).NotTo(HaveOccurred())

	p, err := idemix2.NewProvider(config, registry)
	Expect(err).NotTo(HaveOccurred())
	Expect(p).NotTo(BeNil())

	id, audit, err := p.Identity()
	Expect(err).NotTo(HaveOccurred())
	Expect(id).NotTo(BeNil())
	Expect(audit).NotTo(BeNil())

	auditInfo := &idemix2.AuditInfo{}
	err = auditInfo.FromBytes(audit)
	Expect(err).NotTo(HaveOccurred())
	err = auditInfo.Match(id)
	Expect(err).NotTo(HaveOccurred())

	return id, auditInfo
}

func prepareIssuer(pp *crypto.PublicParams) *anonym.Issuer {
	// prepare issuers' public keys
	sk, pk, err := anonym.GenerateKeyPair("ABC", pp)
	Expect(err).NotTo(HaveOccurred())

	// there are two issuers whereby issuers[1] has secret key sk and issues tokens of type ttype
	issuers := getIssuers(2, 1, pk, pp.ZKATPedParams)
	err = pp.AddIssuer(issuers[0])
	Expect(err).NotTo(HaveOccurred())
	err = pp.AddIssuer(issuers[1])
	Expect(err).NotTo(HaveOccurred())

	witness := anonym.NewWitness(sk, nil, nil, nil, nil, 1)
	signer := anonym.NewSigner(witness, nil, nil, 1, pp.ZKATPedParams)

	issuer := &anonym.Issuer{}
	issuer.New("ABC", signer, pp)

	return issuer
}

func createInputs(pp *crypto.PublicParams, id view.Identity) ([]*token.Token, []*token.TokenInformation) {
	inputs := make([]*token.Token, 2)
	infos := make([]*token.TokenInformation, 2)
	values := []*bn256.Zr{bn256.NewZrInt(25), bn256.NewZrInt(35)}
	rand, err := bn256.GetRand()
	Expect(err).NotTo(HaveOccurred())
	ttype := bn256.HashModOrder([]byte("ABC"))

	for i := 0; i < len(inputs); i++ {
		infos[i] = &token.TokenInformation{}
		infos[i].BlindingFactor = bn256.RandModOrder(rand)
		infos[i].Value = values[i]
		infos[i].Type = "ABC"
		inputs[i] = &token.Token{}
		inputs[i].Data, err = common.ComputePedersenCommitment([]*bn256.Zr{ttype, values[i], infos[i].BlindingFactor}, pp.ZKATPedParams)
		Expect(err).NotTo(HaveOccurred())
		inputs[i].Owner = id
	}

	return inputs, infos
}

func prepareTransfer(pp *crypto.PublicParams, id view.Identity) (*transfer2.TransferAction, []*token.TokenInformation, []*token.Token) {
	inputs, tokenInfos := createInputs(pp, id)

	fakeSigner := &mock.SigningIdentity{}
	sender, err := transfer2.NewSender([]driver.Signer{fakeSigner, fakeSigner}, inputs, []string{"0", "1"}, tokenInfos, pp)
	Expect(err).NotTo(HaveOccurred())
	transfer, inf, err := sender.GenerateZKTransfer([]uint64{40, 20}, [][]byte{id, id})
	Expect(err).NotTo(HaveOccurred())

	return transfer, inf, inputs
}
