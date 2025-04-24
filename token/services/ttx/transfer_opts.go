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

// WithFSCIssuerIdentity takes an issuer's node Identity
// and sets the appropriate attribute in a TransferOptions struct
func WithFSCIssuerIdentity(issuerFSCIdentity view.Identity) token.TransferOption {
	return func(options *token.TransferOptions) error {
		if options.Attributes == nil {
			options.Attributes = make(map[interface{}]interface{})
		}
		options.Attributes[IssuerFSCIdentityKey] = issuerFSCIdentity
		return nil
	}
}

// GetFSCIssuerIdentityFromOpts extracts an issuer's node identity
// from the appropriate attribute in a given attribute map.
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
