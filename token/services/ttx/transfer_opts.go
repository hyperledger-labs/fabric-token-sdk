/*
Copyright IBM Corp. All Rights Reserved.

SPDX-License-Identifier: Apache-2.0
*/

package ttx

import (
	"reflect"

	"github.com/hyperledger-labs/fabric-smart-client/pkg/utils/errors"
	"github.com/hyperledger-labs/fabric-smart-client/platform/view/view"
	"github.com/hyperledger-labs/fabric-token-sdk/token"
)

const (
	IssuerFSCIdentityKey = "IssuerFSCIdentityKey"
)

func WithFSCIssuerIdentity(issuerFSCIdentity view.Identity) token.TransferOption {
	return func(options *token.TransferOptions) error {
		options.Attributes[IssuerFSCIdentityKey] = issuerFSCIdentity
		return nil
	}
}

func GetFSCIssuerIdentityFromOpts(attributes map[interface{}]interface{}) (view.Identity, error) {
	if attributes == nil {
		return nil, nil
	}
	idBoxed, ok := attributes[IssuerFSCIdentityKey]
	if !ok {
		return nil, nil
	}
	id, ok := idBoxed.(view.Identity)
	if !ok {
		return nil, errors.Errorf("expected identity, found [%s]", reflect.TypeOf(idBoxed))
	}
	return id, nil
}
