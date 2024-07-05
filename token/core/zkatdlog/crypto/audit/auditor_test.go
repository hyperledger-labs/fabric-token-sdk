/*
Copyright IBM Corp. All Rights Reserved.

SPDX-License-Identifier: Apache-2.0
*/

package audit_test

import (
	"encoding/json"
	"os"
	"time"

	"github.com/IBM/idemix/bccsp/types"
	math "github.com/IBM/mathlib"
	_ "github.com/hyperledger-labs/fabric-smart-client/platform/view/services/db/driver/memory"
	mem "github.com/hyperledger-labs/fabric-smart-client/platform/view/services/db/driver/memory"
	"github.com/hyperledger-labs/fabric-smart-client/platform/view/services/flogging"
	"github.com/hyperledger-labs/fabric-smart-client/platform/view/services/kvs"
	registry2 "github.com/hyperledger-labs/fabric-smart-client/platform/view/services/registry"
	token2 "github.com/hyperledger-labs/fabric-token-sdk/token"
	"github.com/hyperledger-labs/fabric-token-sdk/token/core/zkatdlog/crypto"
	"github.com/hyperledger-labs/fabric-token-sdk/token/core/zkatdlog/crypto/audit"
	"github.com/hyperledger-labs/fabric-token-sdk/token/core/zkatdlog/crypto/audit/mock"
	"github.com/hyperledger-labs/fabric-token-sdk/token/core/zkatdlog/crypto/token"
	transfer2 "github.com/hyperledger-labs/fabric-token-sdk/token/core/zkatdlog/crypto/transfer"
	"github.com/hyperledger-labs/fabric-token-sdk/token/driver"
	"github.com/hyperledger-labs/fabric-token-sdk/token/services/identity"
	kvs2 "github.com/hyperledger-labs/fabric-token-sdk/token/services/identity/kvs"
	"github.com/hyperledger-labs/fabric-token-sdk/token/services/identity/msp"
	"github.com/hyperledger-labs/fabric-token-sdk/token/services/identity/msp/idemix"
	msp3 "github.com/hyperledger-labs/fabric-token-sdk/token/services/identity/msp/idemix/msp"
	"github.com/hyperledger-labs/fabric-token-sdk/token/services/identity/sig"
	msp2 "github.com/hyperledger/fabric/msp"
	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"
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
		ipk, err := os.ReadFile("./testdata/idemix/msp/IssuerPublicKey")
		Expect(err).NotTo(HaveOccurred())
		pp, err = crypto.Setup(32, ipk, math.FP256BN_AMCL)
		Expect(err).NotTo(HaveOccurred())
		des, err := idemix.NewDeserializer(pp.IdemixIssuerPK, math.FP256BN_AMCL)
		Expect(err).NotTo(HaveOccurred())
		auditor = audit.NewAuditor(flogging.MustGetLogger("auditor"), des, pp.PedersenGenerators, nil, fakeSigningIdentity, math.Curves[pp.Curve])
		fakeSigningIdentity.SignReturns([]byte("auditor-signature"), nil)

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
				Expect(err.Error()).To(ContainSubstring("owner at index [0] does not match the provided opening"))
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
				metadata.OutputAuditInfos[0] = raw
				raw, err = transfer.Serialize()
				Expect(err).NotTo(HaveOccurred())
				err = auditor.Check(&driver.TokenRequest{Transfers: [][]byte{raw}}, &driver.TokenRequestMetadata{Transfers: []driver.TransferMetadata{metadata}}, tokens, "1")
				Expect(err).To(HaveOccurred())
				Expect(err.Error()).To(ContainSubstring("owner at index [0] does not match the provided opening"))
				Expect(err.Error()).To(ContainSubstring("does not match the provided opening"))
				Expect(fakeSigningIdentity.SignCallCount()).To(Equal(0))
			})
		})
	})
})

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

	metadata.OutputsMetadata = marshalledInfo
	metadata.Outputs = make([][]byte, len(transfer.OutputTokens))
	metadata.ReceiverAuditInfos = make([][]byte, len(transfer.OutputTokens))
	metadata.OutputAuditInfos = make([][]byte, len(transfer.OutputTokens))
	for i := 0; i < len(transfer.OutputTokens); i++ {
		metadata.Outputs[i], err = json.Marshal(transfer.OutputTokens[i].Data)
		Expect(err).NotTo(HaveOccurred())
		metadata.ReceiverAuditInfos[i], err = auditInfo.Bytes()
		Expect(err).NotTo(HaveOccurred())
		metadata.OutputAuditInfos[i] = metadata.ReceiverAuditInfos[i]
	}
	tokns := make([][]*token.Token, 1)
	for i := 0; i < len(inputs); i++ {
		tokns[0] = append(tokns[0], inputs[i])
	}
	return transfer, metadata, tokns
}

func createTransferWithBogusOutput(pp *crypto.PublicParams) (*transfer2.TransferAction, driver.TransferMetadata, [][]*token.Token) {
	id, auditInfo := getIdemixInfo("./testdata/idemix")
	transfer, inf, inputs := prepareTransfer(pp, id)

	c := math.Curves[pp.Curve]
	inf[0].Value = c.NewZrFromInt(15)
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

	metadata.OutputsMetadata = marshalledInfo
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

func getIdemixInfo(dir string) (driver.Identity, *msp3.AuditInfo) {
	registry := registry2.New()
	Expect(registry.RegisterService(&fakeProv{typ: "memory"})).NotTo(HaveOccurred())

	backend, err := kvs.New(registry, &mem.Driver{}, "")
	Expect(err).NotTo(HaveOccurred())
	err = registry.RegisterService(backend)
	Expect(err).NotTo(HaveOccurred())

	sigService := sig.NewService(sig.NewMultiplexDeserializer(), kvs2.NewIdentityDB(backend, token2.TMSID{Network: "pineapple"}))
	err = registry.RegisterService(sigService)
	Expect(err).NotTo(HaveOccurred())
	config, err := msp2.GetLocalMspConfigWithType(dir, nil, "idemix", "idemix")
	Expect(err).NotTo(HaveOccurred())

	keyStore, err := msp3.NewKeyStore(math.FP256BN_AMCL, backend)
	Expect(err).NotTo(HaveOccurred())
	cryptoProvider, err := msp3.NewBCCSP(keyStore, math.FP256BN_AMCL, false)
	Expect(err).NotTo(HaveOccurred())
	p, err := idemix.NewProvider(config, sigService, types.EidNymRhNym, cryptoProvider)
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

	id, err = identity.WrapWithType(msp.IdemixIdentity, id)
	Expect(err).NotTo(HaveOccurred())

	return id, auditInfo
}

func createInputs(pp *crypto.PublicParams, id driver.Identity) ([]*token.Token, []*token.Metadata) {
	c := math.Curves[pp.Curve]
	inputs := make([]*token.Token, 2)
	infos := make([]*token.Metadata, 2)
	values := []*math.Zr{c.NewZrFromInt(25), c.NewZrFromInt(35)}
	rand, err := c.Rand()
	Expect(err).NotTo(HaveOccurred())
	ttype := c.HashToZr([]byte("ABC"))

	for i := 0; i < len(inputs); i++ {
		infos[i] = &token.Metadata{}
		infos[i].BlindingFactor = c.NewRandomZr(rand)
		infos[i].Value = values[i]
		infos[i].Type = "ABC"
		inputs[i] = &token.Token{}
		inputs[i].Data = commit([]*math.Zr{ttype, values[i], infos[i].BlindingFactor}, pp.PedersenGenerators, c)
		Expect(err).NotTo(HaveOccurred())
		inputs[i].Owner = id
	}

	return inputs, infos
}

func prepareTransfer(pp *crypto.PublicParams, id driver.Identity) (*transfer2.TransferAction, []*token.Metadata, []*token.Token) {
	inputs, tokenInfos := createInputs(pp, id)

	fakeSigner := &mock.SigningIdentity{}
	sender, err := transfer2.NewSender([]driver.Signer{fakeSigner, fakeSigner}, inputs, []string{"0", "1"}, tokenInfos, pp)
	Expect(err).NotTo(HaveOccurred())
	transfer, inf, err := sender.GenerateZKTransfer([]uint64{40, 20}, [][]byte{id, id})
	Expect(err).NotTo(HaveOccurred())

	return transfer, inf, inputs
}

func commit(vector []*math.Zr, generators []*math.G1, c *math.Curve) *math.G1 {
	com := c.NewG1()
	for i := range vector {
		com.Add(generators[i].Mul(vector[i]))
	}
	return com
}
