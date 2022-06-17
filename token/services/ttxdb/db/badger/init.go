/*
Copyright IBM Corp. All Rights Reserved.

SPDX-License-Identifier: Apache-2.0
*/

package badger

import (
	"os"
	"path/filepath"

	view2 "github.com/hyperledger-labs/fabric-smart-client/platform/view"
	"github.com/hyperledger-labs/fabric-smart-client/platform/view/services/flogging"
	"github.com/hyperledger-labs/fabric-token-sdk/token/services/ttxdb"
	"github.com/hyperledger-labs/fabric-token-sdk/token/services/ttxdb/driver"
	"github.com/pkg/errors"
)

var logger = flogging.MustGetLogger("token-sdk.ttxdb.badger")

const (
	// OptsKey is the key for the opts in the config
	OptsKey = "token.ttxdb.persistence.opts"
)

type Opts struct {
	Path string
}

type Driver struct {
}

func (d Driver) Open(sp view2.ServiceProvider, name string) (driver.TokenTransactionDB, error) {
	opts := &Opts{}
	err := view2.GetConfigService(sp).UnmarshalKey(OptsKey, opts)
	if err != nil {
		return nil, errors.Wrapf(err, "failed getting opts for vault")
	}
	opts.Path = filepath.Join(opts.Path, name)
	logger.Debugf("init kvs with badger at [%s]", opts.Path)

	err = os.MkdirAll(opts.Path, 0755)
	if err != nil {
		return nil, errors.Wrapf(err, "failed creating folders for vault [%s]", opts.Path)
	}
	persistence, err := OpenDB(opts.Path)
	if err != nil {
		return nil, errors.Wrapf(err, "failed opening vault [%s]", opts.Path)
	}
	return persistence, nil
}

func init() {
	ttxdb.Register("badger", &Driver{})
}
