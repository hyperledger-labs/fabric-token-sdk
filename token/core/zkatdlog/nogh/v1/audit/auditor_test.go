/*
Copyright IBM Corp. All Rights Reserved.

SPDX-License-Identifier: Apache-2.0
*/

package audit_test

import (
	"context"
	"os"
	"time"

	"github.com/IBM/idemix/bccsp/types"
	math "github.com/IBM/mathlib"
	"github.com/hyperledger-labs/fabric-smart-client/platform/view/services/view"
	"github.com/hyperledger-labs/fabric-token-sdk/token/core/zkatdlog/nogh/v1/audit"
	"github.com/hyperledger-labs/fabric-token-sdk/token/core/zkatdlog/nogh/v1/audit/mock"
	v1 "github.com/hyperledger-labs/fabric-token-sdk/token/core/zkatdlog/nogh/v1/setup"
	"github.com/hyperledger-labs/fabric-token-sdk/token/core/zkatdlog/nogh/v1/token"
	"github.com/hyperledger-labs/fabric-token-sdk/token/core/zkatdlog/nogh/v1/transfer"
	"github.com/hyperledger-labs/fabric-token-sdk/token/driver"
	"github.com/hyperledger-labs/fabric-token-sdk/token/services/identity"
	"github.com/hyperledger-labs/fabric-token-sdk/token/services/identity/deserializer"
	idemix2 "github.com/hyperledger-labs/fabric-token-sdk/token/services/identity/idemix"
	"github.com/hyperledger-labs/fabric-token-sdk/token/services/identity/idemix/crypto"
	kvs2 "github.com/hyperledger-labs/fabric-token-sdk/token/services/identity/storage/kvs"
	"github.com/hyperledger-labs/fabric-token-sdk/token/services/logging"
	"github.com/hyperledger-labs/fabric-token-sdk/token/services/utils/slices"
	token3 "github.com/hyperledger-labs/fabric-token-sdk/token/token"
	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"
	"go.opentelemetry.io/otel/trace/noop"
)

var _ = Describe("Auditor", func() {
	var (
		fakeSigningIdentity *mock.SigningIdentity
		pp                  *v1.PublicParams
		auditor             *audit.Auditor
	)
	BeforeEach(func() {
		var err error
		fakeSigningIdentity = &mock.SigningIdentity{}
		ipk, err := os.ReadFile("./testdata/idemix/msp/IssuerPublicKey")
		Expect(err).NotTo(HaveOccurred())
		pp, err = v1.Setup(32, ipk, math.FP256BN_AMCL)
		Expect(err).NotTo(HaveOccurred())
		idemixDes, err := idemix2.NewDeserializer(slices.GetUnique(pp.IdemixIssuerPublicKeys).PublicKey, math.FP256BN_AMCL)
		Expect(err).NotTo(HaveOccurred())
		des := deserializer.NewTypedVerifierDeserializerMultiplex()
		des.AddTypedVerifierDeserializer(idemix2.IdentityType, deserializer.NewTypedIdentityVerifierDeserializer(idemixDes, idemixDes))
		auditor = audit.NewAuditor(
			logging.MustGetLogger(),
			&noop.Tracer{},
			des,
			pp.PedersenGenerators,
			fakeSigningIdentity,
			math.Curves[pp.Curve],
		)
		fakeSigningIdentity.SignReturns([]byte("auditor-signature"), nil)

	})

	Describe("Audit a transfer", func() {
		When("audit information is computed correctly", func() {
			It("succeeds", func() {
				transfer, metadata, tokens := createTransfer(pp)
				raw, err := transfer.Serialize()
				Expect(err).NotTo(HaveOccurred())
				err = auditor.Check(context.Background(), &driver.TokenRequest{Transfers: [][]byte{raw}}, &driver.TokenRequestMetadata{Transfers: []*driver.TransferMetadata{metadata}}, tokens, "1")
				Expect(err).NotTo(HaveOccurred())
			})
		})
		When("token info does not match output", func() {
			It("fails", func() {
				transfer, metadata, tokens := createTransferWithBogusOutput(pp)
				raw, err := transfer.Serialize()
				Expect(err).NotTo(HaveOccurred())
				err = auditor.Check(
					context.Background(),
					&driver.TokenRequest{Transfers: [][]byte{raw}},
					&driver.TokenRequestMetadata{Transfers: []*driver.TransferMetadata{metadata}},
					tokens,
					"1",
				)
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
				metadata.Inputs[0].Senders[0].AuditInfo = raw
				raw, err = transfer.Serialize()
				Expect(err).NotTo(HaveOccurred())
				err = auditor.Check(context.Background(), &driver.TokenRequest{Transfers: [][]byte{raw}}, &driver.TokenRequestMetadata{Transfers: []*driver.TransferMetadata{metadata}}, tokens, "1")
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
				metadata.Outputs[0].OutputAuditInfo = raw
				raw, err = transfer.Serialize()
				Expect(err).NotTo(HaveOccurred())
				err = auditor.Check(context.Background(), &driver.TokenRequest{Transfers: [][]byte{raw}}, &driver.TokenRequestMetadata{Transfers: []*driver.TransferMetadata{metadata}}, tokens, "1")
				Expect(err).To(HaveOccurred())
				Expect(err.Error()).To(ContainSubstring("owner at index [0] does not match the provided opening"))
				Expect(err.Error()).To(ContainSubstring("does not match the provided opening"))
				Expect(fakeSigningIdentity.SignCallCount()).To(Equal(0))
			})
		})
	})
})

func createTransfer(pp *v1.PublicParams) (*transfer.Action, *driver.TransferMetadata, [][]*token.Token) {
	id, auditInfo := getIdemixInfo("./testdata/idemix")
	transfer, meta, inputs := prepareTransfer(pp, id)

	auditInfoRaw, err := auditInfo.Bytes()
	Expect(err).NotTo(HaveOccurred())

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
		Expect(err).NotTo(HaveOccurred())
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

func createTransferWithBogusOutput(pp *v1.PublicParams) (*transfer.Action, *driver.TransferMetadata, [][]*token.Token) {
	id, auditInfo := getIdemixInfo("./testdata/idemix")
	transfer, inf, inputs := prepareTransfer(pp, id)

	c := math.Curves[pp.Curve]
	inf[0].Value = c.NewZrFromInt(15)
	marshalledInfo := make([][]byte, len(inf))
	var err error
	for i := range inf {
		marshalledInfo[i], err = inf[i].Serialize()
		Expect(err).NotTo(HaveOccurred())
	}
	auditInfoRaw, err := auditInfo.Bytes()
	Expect(err).NotTo(HaveOccurred())

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
		Expect(err).NotTo(HaveOccurred())
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

func getIdemixInfo(dir string) (driver.Identity, *crypto.AuditInfo) {
	sp := view.NewServiceProvider()
	configService := &fakeProv{typ: "memory"}
	Expect(sp.RegisterService(configService)).NotTo(HaveOccurred())

	backend, err := kvs2.NewInMemory()
	Expect(err).NotTo(HaveOccurred())
	err = sp.RegisterService(backend)
	Expect(err).NotTo(HaveOccurred())

	Expect(err).NotTo(HaveOccurred())
	config, err := crypto.NewConfig(dir)
	Expect(err).NotTo(HaveOccurred())

	keyStore, err := crypto.NewKeyStore(math.FP256BN_AMCL, kvs2.Keystore(backend))
	Expect(err).NotTo(HaveOccurred())
	cryptoProvider, err := crypto.NewBCCSP(keyStore, math.FP256BN_AMCL, false)
	Expect(err).NotTo(HaveOccurred())
	p, err := idemix2.NewKeyManager(config, types.EidNymRhNym, cryptoProvider)
	Expect(err).NotTo(HaveOccurred())
	Expect(p).NotTo(BeNil())

	identityDescriptor, err := p.Identity(context.Background(), nil)
	Expect(err).NotTo(HaveOccurred())
	id := identityDescriptor.Identity
	audit := identityDescriptor.AuditInfo
	Expect(id).NotTo(BeNil())
	Expect(audit).NotTo(BeNil())

	auditInfo, err := p.DeserializeAuditInfo(context.Background(), audit)
	Expect(err).NotTo(HaveOccurred())
	err = auditInfo.Match(context.Background(), id)
	Expect(err).NotTo(HaveOccurred())

	id, err = identity.WrapWithType(idemix2.IdentityType, id)
	Expect(err).NotTo(HaveOccurred())

	return id, auditInfo
}

func createInputs(pp *v1.PublicParams, id driver.Identity) ([]*token.Token, []*token.Metadata) {
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

func prepareTransfer(pp *v1.PublicParams, id driver.Identity) (*transfer.Action, []*token.Metadata, []*token.Token) {
	inputs, tokenInfos := createInputs(pp, id)

	fakeSigner := &mock.SigningIdentity{}

	sender, err := transfer.NewSender([]driver.Signer{fakeSigner, fakeSigner}, inputs, []*token3.ID{{TxId: "0"}, {TxId: "1"}}, tokenInfos, pp)
	Expect(err).NotTo(HaveOccurred())
	transfer, inf, err := sender.GenerateZKTransfer(context.TODO(), []uint64{40, 20}, [][]byte{id, id})
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
