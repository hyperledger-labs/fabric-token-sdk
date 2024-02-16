/*
Copyright IBM Corp. All Rights Reserved.

SPDX-License-Identifier: Apache-2.0
*/

package common

import (
	"fmt"
	"os"
	"path/filepath"
	"time"

	api2 "github.com/hyperledger-labs/fabric-smart-client/integration/nwo/api"
	"github.com/hyperledger-labs/fabric-smart-client/integration/nwo/common"
	sfcnode "github.com/hyperledger-labs/fabric-smart-client/integration/nwo/fsc/node"
	"github.com/hyperledger-labs/fabric-token-sdk/integration/nwo/token/generators"
	topology2 "github.com/hyperledger-labs/fabric-token-sdk/integration/nwo/token/topology"
	. "github.com/onsi/gomega"
	"github.com/onsi/gomega/gexec"
)

type TokenPlatform interface {
	TokenGen(keygen common.Command) (*gexec.Session, error)
	PublicParametersFile(tms *topology2.TMS) string
	GetContext() api2.Context
	PublicParameters(tms *topology2.TMS) []byte
	GetPublicParamsGenerators(driver string) generators.PublicParamsGenerator
	PublicParametersDir() string
	GetBuilder() api2.Builder
	TokenDir() string
	UpdatePublicParams(tms *topology2.TMS, pp []byte)
}

type NetworkHandler struct {
	TokenPlatform            TokenPlatform
	CryptoMaterialGenerators map[string]generators.CryptoMaterialGenerator
	CASupports               map[string]CAFactory

	EventuallyTimeout time.Duration
	ColorIndex        int
}

func (p *NetworkHandler) TTXDBSQLDataSourceDir(peer *sfcnode.Node) string {
	return filepath.Join(p.TokenPlatform.GetContext().RootDir(), "fsc", "nodes", peer.ID(), "ttxdb")
}

func (p *NetworkHandler) TokensDBSQLDataSourceDir(peer *sfcnode.Node) string {
	return filepath.Join(p.TokenPlatform.GetContext().RootDir(), "fsc", "nodes", peer.ID(), "tokensdb")
}

func (p *NetworkHandler) AuditDBSQLDataSourceDir(peer *sfcnode.Node) string {
	return filepath.Join(p.TokenPlatform.GetContext().RootDir(), "fsc", "nodes", peer.ID(), "auditdb")
}

func (p *NetworkHandler) DBPath(root string, tms *topology2.TMS) string {
	return "file:" +
		filepath.Join(
			root,
			fmt.Sprintf("%s_%s_%s", tms.Network, tms.Channel, tms.Namespace)+"_db.sqlite",
		) + "?_pragma=journal_mode(WAL)&_pragma=busy_timeout(5000)"
}

func (p *NetworkHandler) FSCNodeKVSDir(peer *sfcnode.Node) string {
	return filepath.Join(p.TokenPlatform.GetContext().RootDir(), "fsc", "nodes", peer.ID(), "kvs")
}

func (p *NetworkHandler) DeleteDBs(node *sfcnode.Node) {
	for _, path := range []string{p.TokensDBSQLDataSourceDir(node)} {
		logger.Infof("remove all [%s]", path)
		Expect(os.RemoveAll(path)).ToNot(HaveOccurred())
		Expect(os.MkdirAll(path, 0775)).ToNot(HaveOccurred(), "failed to create [%s]", path)
	}
}
