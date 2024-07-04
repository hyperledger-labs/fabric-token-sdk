/*
Copyright IBM Corp. All Rights Reserved.

SPDX-License-Identifier: Apache-2.0
*/

package fodlog

import (
	"errors"

	"github.com/hyperledger-labs/fabric-smart-client/pkg/node"
	fabricsdk "github.com/hyperledger-labs/fabric-smart-client/platform/fabric/sdk/dig"
	orionsdk "github.com/hyperledger-labs/fabric-smart-client/platform/orion/sdk/dig"
	viewsdk "github.com/hyperledger-labs/fabric-smart-client/platform/view/sdk/dig"
	_ "github.com/hyperledger-labs/fabric-token-sdk/token/core/zkatdlog/nogh/driver"
	tokensdk "github.com/hyperledger-labs/fabric-token-sdk/token/sdk/dig"
	auditdb "github.com/hyperledger-labs/fabric-token-sdk/token/services/auditdb/db/sql"
	identitydb "github.com/hyperledger-labs/fabric-token-sdk/token/services/identitydb/db/sql"
	"github.com/hyperledger-labs/fabric-token-sdk/token/services/network/fabric"
	"github.com/hyperledger-labs/fabric-token-sdk/token/services/network/orion"
	tokendb "github.com/hyperledger-labs/fabric-token-sdk/token/services/tokendb/db/sql"
	tokenlockdb "github.com/hyperledger-labs/fabric-token-sdk/token/services/tokenlockdb/db/sql"
	ttxdb "github.com/hyperledger-labs/fabric-token-sdk/token/services/ttxdb/db/sql"
	"go.uber.org/dig"
)

type SDK struct {
	*tokensdk.SDK
}

func NewSDK(registry node.Registry) *SDK {
	return &SDK{SDK: tokensdk.NewFrom(orionsdk.NewFrom(fabricsdk.NewFrom(viewsdk.NewSDK(registry))))}
}

func (p *SDK) Install() error {
	err := errors.Join(
		p.Container().Provide(fabric.NewDriver, dig.Group("network-drivers")),
		p.Container().Provide(orion.NewDriver, dig.Group("network-drivers")),
		p.Container().Provide(tokenlockdb.NewDriver, dig.Group("tokenlockdb-drivers")),
		p.Container().Provide(auditdb.NewDriver, dig.Group("auditdb-drivers")),
		p.Container().Provide(tokendb.NewDriver, dig.Group("tokendb-drivers")),
		p.Container().Provide(ttxdb.NewDriver, dig.Group("ttxdb-drivers")),
		p.Container().Provide(identitydb.NewDriver, dig.Group("identitydb-drivers")),
		p.Container().Provide(tokensdk.NewDBDrivers),
	)
	if err != nil {
		return err
	}

	return p.SDK.Install()
}
