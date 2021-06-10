/*
Copyright IBM Corp. All Rights Reserved.

SPDX-License-Identifier: Apache-2.0
*/
package token

import (
	"bytes"
	"encoding/base64"
	"encoding/json"
	"fmt"
	"io"
	"io/ioutil"
	"os"
	"path/filepath"
	"strings"
	"text/template"

	. "github.com/onsi/gomega"

	"github.com/hyperledger-labs/fabric-smart-client/integration/nwo/fabric/topology"

	pp2 "github.com/hyperledger-labs/fabric-token-sdk/token/core/cmd/pp"
	packager2 "github.com/hyperledger-labs/fabric-token-sdk/token/core/cmd/pp/packager"
)

const DefaultTokenChaincode = "github.com/hyperledger-labs/fabric-token-sdk/token/services/tcc/main"

func (p *Platform) tccSetup(tms *TMS, cc *topology.ChannelChaincode) (*topology.ChannelChaincode, uint16) {
	// Load public parameters
	ppRaw, err := ioutil.ReadFile(p.PublicParametersFile(tms))
	Expect(err).ToNot(HaveOccurred())

	// produce chaincode package
	packageDir := filepath.Join(
		p.Registry.RootDir,
		"chaincodes",
		"tcc",
		tms.Channel,
		tms.Namespace,
	)
	packageFile := filepath.Join(
		packageDir,
		cc.Chaincode.Name+".tar.gz",
	)
	Expect(os.MkdirAll(packageDir, 0766)).ToNot(HaveOccurred())

	t, err := template.New("node").Funcs(template.FuncMap{
		"Params": func() string { return base64.StdEncoding.EncodeToString(ppRaw) },
	}).Parse(pp2.DefaultParams)
	Expect(err).ToNot(HaveOccurred())
	paramsFile := bytes.NewBuffer(nil)
	err = t.Execute(io.MultiWriter(paramsFile), nil)
	Expect(err).ToNot(HaveOccurred())

	port := p.Registry.ReservePort()
	err = packager2.New().PackageChaincode(
		cc.Chaincode.Path,
		cc.Chaincode.Lang,
		cc.Chaincode.Label,
		packageFile,
		func(s string, s2 string) []byte {
			// Is the public params?
			if strings.HasSuffix(s, "github.com/hyperledger-labs/fabric-token-sdk/token/services/tcc/params.go") {
				return paramsFile.Bytes()
			}

			// Is connection.json?
			if s == s2 && s == "connection.json" {
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
				return raw
			}
			return nil
		},
	)
	Expect(err).ToNot(HaveOccurred())

	cc.Chaincode.Ctor = fmt.Sprintf(`{"Args":["init"]}`)
	cc.Chaincode.PackageFile = packageFile

	return cc, port
}

func (p *Platform) PrepareTCC(tms *TMS) (*topology.ChannelChaincode, uint16) {
	orgs := tms.TokenChaincode.Orgs

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
		for _, peer := range p.FabricNetwork.Topology().Peers {
			if peer.Organization == org {
				peers = append(peers, peer.Name)
			}
		}
	}

	return p.tccSetup(tms, &topology.ChannelChaincode{
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
}
