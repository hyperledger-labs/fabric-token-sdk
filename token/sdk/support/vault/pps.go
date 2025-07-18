/*
Copyright IBM Corp. All Rights Reserved.

SPDX-License-Identifier: Apache-2.0
*/

package vault

import (
	"context"

	"github.com/pkg/errors"
)

type PublicParamsStorage struct {
	Provider *Provider
}

func (p *PublicParamsStorage) PublicParams(ctx context.Context, networkID string, channel string, namespace string) ([]byte, error) {
	vault, err := p.Provider.Vault(networkID, channel, namespace)
	if err != nil {
		return nil, errors.WithMessagef(err, "failed to get network for [%s:%s:%s]", networkID, channel, namespace)
	}
	return vault.QueryEngine().PublicParams(ctx)
}
