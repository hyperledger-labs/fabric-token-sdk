/*
Copyright IBM Corp. All Rights Reserved.

SPDX-License-Identifier: Apache-2.0
*/

package cc

import (
	"bytes"
	"encoding/base64"
	"encoding/json"
	"fmt"
	"io"
	"os"
	"path/filepath"
	"strings"
	"text/template"

	"github.com/hyperledger-labs/fabric-smart-client/integration/nwo/fabric/packager"
	"github.com/hyperledger-labs/fabric-smart-client/integration/nwo/fabric/topology"
	"github.com/hyperledger-labs/fabric-smart-client/platform/view/services/flogging"
	pp2 "github.com/hyperledger-labs/fabric-token-sdk/cmd/tokengen/cobra/pp/cc"
	common2 "github.com/hyperledger-labs/fabric-token-sdk/integration/nwo/token/common"
	"github.com/hyperledger-labs/fabric-token-sdk/integration/nwo/token/fabric"
	topology3 "github.com/hyperledger-labs/fabric-token-sdk/integration/nwo/token/topology"
	. "github.com/onsi/gomega"
)

const (
	DefaultTokenChaincode                    = "github.com/hyperledger-labs/fabric-token-sdk/token/services/network/fabric/tcc/main"
	DefaultTokenChaincodeParamsReplaceSuffix = "/token/services/network/fabric/tcc/params.go"
)

var logger = flogging.MustGetLogger("token-sdk.integration.token.fabric.cc")

type fabricPlatform interface {
	Topology() *topology.Topology
	UpdateChaincode(name string, version string, path string, file string)
}

type TCC struct {
	Chaincode *topology.ChannelChaincode
}

type GenericBackend struct {
	TokenChaincodePath                string
	TokenChaincodeParamsReplaceSuffix string
	TokenPlatform                     common2.TokenPlatform
}

func NewDefaultGenericBackend(tokenPlatform common2.TokenPlatform) *GenericBackend {
	return NewGenericBackend(
		DefaultTokenChaincode,
		DefaultTokenChaincodeParamsReplaceSuffix,
		tokenPlatform,
	)
}

func NewGenericBackend(tokenChaincodePath string, tokenChaincodeParamsReplaceSuffix string, tokenPlatform common2.TokenPlatform) *GenericBackend {
	return &GenericBackend{TokenChaincodePath: tokenChaincodePath, TokenChaincodeParamsReplaceSuffix: tokenChaincodeParamsReplaceSuffix, TokenPlatform: tokenPlatform}
}

func (p *GenericBackend) PrepareNamespace(entry *fabric.Entry, tms *topology3.TMS) {
	orgs := tms.BackendParams["fabric.orgs"].([]string)

	// Standard Chaincode
	policy := "AND ("
	for i, org := range orgs {
		if i > 0 {
			policy += ","
		}
		policy += "'" + org + "MSP.member'"
	}
	policy += ")"

	var peers []string
	for _, org := range orgs {
		for _, peer := range p.Fabric(tms).Topology().Peers {
			if peer.Organization == org {
				peers = append(peers, peer.Name)
			}
		}
	}

	cc, _ := p.tccSetup(tms, &topology.ChannelChaincode{
		Chaincode: topology.Chaincode{
			Name:            tms.Namespace,
			Version:         "Version-0.0",
			Sequence:        "1",
			InitRequired:    true,
			Path:            p.TokenChaincodePath,
			Lang:            "golang",
			Label:           tms.Namespace,
			Policy:          policy,
			SignaturePolicy: policy,
		},
		Channel: tms.Channel,
		Peers:   peers,
	})
	p.Fabric(tms).Topology().AddChaincode(cc)
}

func (p *GenericBackend) UpdatePublicParams(tms *topology3.TMS, ppRaw []byte) {
	var cc *topology.ChannelChaincode
	for _, chaincode := range p.Fabric(tms).Topology().Chaincodes {
		if chaincode.Chaincode.Name == tms.Namespace {
			cc = chaincode
			break
		}
	}
	Expect(cc).NotTo(BeNil(), "failed to find chaincode [%s]", tms.Namespace)

	packageDir := filepath.Join(
		p.TokenPlatform.GetContext().RootDir(),
		"token",
		"chaincodes",
		"tcc",
		tms.Network,
		tms.Channel,
		tms.Namespace,
	)
	newChaincodeVersion := cc.Chaincode.Version + ".1"
	packageFile := filepath.Join(
		packageDir,
		cc.Chaincode.Name+newChaincodeVersion+".tar.gz",
	)
	Expect(os.MkdirAll(packageDir, 0766)).ToNot(HaveOccurred())

	paramsFile := PublicParamsTemplate(ppRaw)

	err := packager.New().PackageChaincode(
		cc.Chaincode.Path,
		cc.Chaincode.Lang,
		cc.Chaincode.Label,
		packageFile,
		func(filePath string, fileName string) (string, []byte) {
			if strings.HasSuffix(filePath, p.TokenChaincodeParamsReplaceSuffix) {
				logger.Debugf("replace [%s:%s]? Yes, this is tcc params", filePath, fileName)
				return "", paramsFile.Bytes()
			}
			return "", nil
		},
	)
	Expect(err).ToNot(HaveOccurred())
	cc.Chaincode.PackageFile = packageFile
	p.Fabric(tms).UpdateChaincode(cc.Chaincode.Name,
		newChaincodeVersion,
		cc.Chaincode.Path, cc.Chaincode.PackageFile)
}

func (p *GenericBackend) tccSetup(tms *topology3.TMS, cc *topology.ChannelChaincode) (*topology.ChannelChaincode, uint16) {
	// Load public parameters
	logger.Debugf("tcc setup, reading public parameters from [%s]", p.TokenPlatform.PublicParametersFile(tms))
	ppRaw, err := os.ReadFile(p.TokenPlatform.PublicParametersFile(tms))
	Expect(err).ToNot(HaveOccurred())

	// produce chaincode package
	packageDir := filepath.Join(
		p.TokenPlatform.GetContext().RootDir(),
		"token",
		"chaincodes",
		"tcc",
		tms.Network,
		tms.Channel,
		tms.Namespace,
	)
	packageFile := filepath.Join(
		packageDir,
		cc.Chaincode.Name+".tar.gz",
	)
	Expect(os.MkdirAll(packageDir, 0766)).ToNot(HaveOccurred())
	paramsFile := PublicParamsTemplate(ppRaw)
	port := p.TokenPlatform.GetContext().ReservePort()
	err = packager.New().PackageChaincode(
		cc.Chaincode.Path,
		cc.Chaincode.Lang,
		cc.Chaincode.Label,
		packageFile,
		func(filePath string, fileName string) (string, []byte) {
			// logger.Infof("replace [%s:%s]?", s, s2)
			// Is the public params?
			if strings.HasSuffix(filePath, p.TokenChaincodeParamsReplaceSuffix) {
				logger.Debugf("replace [%s:%s]? Yes, this is tcc params", filePath, fileName)
				return "", paramsFile.Bytes()
			}

			// Is connection.json?
			if filePath == fileName && filePath == "connection.json" {
				// Connection holds the path and type for a chaincode package
				type Connection struct {
					Address     string `json:"address"`
					DialTimeout string `json:"dial_timeout"`
					TLSRequired bool   `json:"tls_required"`
				}
				connection := &Connection{
					Address:     p.TokenChaincodeServerAddr(port),
					DialTimeout: "10s",
					TLSRequired: false,
				}
				raw, err := json.Marshal(connection)
				if err != nil {
					panic("failed to marshal chaincode package connection into JSON")
				}
				return "", raw
			}
			return "", nil
		},
	)
	Expect(err).ToNot(HaveOccurred())

	cc.Chaincode.Ctor = `{"Args":["init"]}`
	cc.Chaincode.PackageFile = packageFile

	return cc, port
}

func (p *GenericBackend) TCCCtor(tms *topology3.TMS) string {
	logger.Debugf("tcc setup, reading public parameters for setting up CTOR [%s]", p.TokenPlatform.PublicParametersFile(tms))
	ppRaw, err := os.ReadFile(p.TokenPlatform.PublicParametersFile(tms))
	Expect(err).ToNot(HaveOccurred())

	return fmt.Sprintf(`{"Args":["init", "%s"]}`, base64.StdEncoding.EncodeToString(ppRaw))
}

func (p *GenericBackend) TokenChaincodeServerAddr(port uint16) string {
	return fmt.Sprintf("127.0.0.1:%d", port)
}

func (p *GenericBackend) Fabric(tms *topology3.TMS) fabricPlatform {
	return p.TokenPlatform.GetContext().PlatformByName(tms.Network).(fabricPlatform)
}

func PublicParamsTemplate(ppRaw []byte) *bytes.Buffer {
	t, err := template.New("node").Funcs(template.FuncMap{
		"Params": func() string { return base64.StdEncoding.EncodeToString(ppRaw) },
	}).Parse(pp2.DefaultParams)
	Expect(err).ToNot(HaveOccurred())
	paramsFile := bytes.NewBuffer(nil)
	err = t.Execute(io.MultiWriter(paramsFile), nil)
	Expect(err).ToNot(HaveOccurred())
	return paramsFile
}
