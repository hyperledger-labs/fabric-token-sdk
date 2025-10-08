/*
Copyright IBM Corp. All Rights Reserved.

SPDX-License-Identifier: Apache-2.0
*/

package pp

import (
	"encoding/json"
	"os"

	"github.com/hyperledger-labs/fabric-smart-client/pkg/utils/errors"
	"github.com/hyperledger-labs/fabric-smart-client/platform/view/view"
	"github.com/hyperledger-labs/fabric-token-sdk/token"
	"github.com/hyperledger-labs/fabric-token-sdk/token/services/logging"
	"github.com/hyperledger-labs/fabric-token-sdk/token/services/network/fabricx/pp"
)

var logger = logging.MustGetLogger()

type LoadPublicParameters struct {
	TMSID token.TMSID
	Path  string
	Raw   []byte
}

type LoadPublicParametersView struct {
	*LoadPublicParameters
}

func (f *LoadPublicParametersView) Call(ctx view.Context) (interface{}, error) {
	p, err := pp.GetPublicParametersService(ctx)
	if err != nil {
		return nil, err
	}

	if len(f.Raw) == 0 && len(f.Path) == 0 {
		return nil, errors.New("specify path or raw PPs")
	}
	ppRaw := f.Raw
	if len(ppRaw) == 0 {
		logger.Infof("Fetching PPs from [%s]", f.Path)
		if ppRaw, err = os.ReadFile(f.Path); err != nil || len(ppRaw) == 0 {
			return nil, errors.Wrapf(err, "failed to read PPs from [%s]", f.Path)
		}
	} else {
		logger.Infof("Raw PPs passed: %")
	}

	err = p.LoadPublicParams(f.TMSID, ppRaw)
	if err != nil {
		return nil, err
	}

	return nil, nil
}

type LoadPublicParametersViewFactory struct{}

func (p *LoadPublicParametersViewFactory) NewView(in []byte) (view.View, error) {
	f := &LoadPublicParametersView{LoadPublicParameters: &LoadPublicParameters{}}
	err := json.Unmarshal(in, f.LoadPublicParameters)
	if err != nil {
		return nil, err
	}
	return f, nil
}
