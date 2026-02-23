/*
Copyright IBM Corp. All Rights Reserved.

SPDX-License-Identifier: Apache-2.0
*/

package vault

import (
	"context"

	"github.com/hyperledger-labs/fabric-smart-client/pkg/utils/errors"
)

// PublicParamsStorage retrieves public parameters from vault storage.
type PublicParamsStorage struct {
	Provider *Provider
}

// PublicParams returns the serialized public parameters for the given TMS coordinates.
func (p *PublicParamsStorage) PublicParams(ctx context.Context, networkID string, channel string, namespace string) ([]byte, error) {
	vault, err := p.Provider.Vault(networkID, channel, namespace)
	if err != nil {
		return nil, errors.WithMessagef(err, "failed to get network for [%s:%s:%s]", networkID, channel, namespace)
	}

	return vault.QueryEngine().PublicParams(ctx)
}
