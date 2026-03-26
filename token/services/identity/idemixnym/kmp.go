/*
Copyright IBM Corp. All Rights Reserved.

SPDX-License-Identifier: Apache-2.0
*/

package idemixnym

import (
	bccsp "github.com/IBM/idemix/bccsp/types"
	math "github.com/IBM/mathlib"
	"github.com/hyperledger-labs/fabric-smart-client/platform/view/services/metrics"
	idriver "github.com/hyperledger-labs/fabric-token-sdk/token/services/identity/driver"
	"github.com/hyperledger-labs/fabric-token-sdk/token/services/identity/idemix"
	"github.com/hyperledger-labs/fabric-token-sdk/token/services/identity/idemix/crypto"
	"github.com/hyperledger-labs/fabric-token-sdk/token/services/identity/membership"
)

type KeyManagerProvider = idemix.KeyManagerProvider

func NewKeyManagerProvider(
	issuerPublicKey []byte,
	curveID math.CurveID,
	keyStore bccsp.KeyStore,
	config idriver.Config,
	cacheSize int,
	ignoreVerifyOnlyWallet bool,
	metricsProvider metrics.Provider,
	identityStoreService IdentityStoreService,
) *KeyManagerProvider {
	return idemix.NewKeyManagerProviderWithKeyManagerFactory(
		issuerPublicKey,
		curveID,
		keyStore,
		config,
		cacheSize,
		ignoreVerifyOnlyWallet,
		metricsProvider,
		func(conf *crypto.Config, _ bccsp.SignatureType, csp bccsp.BCCSP) (membership.KeyManager, error) {
			ikm, err := idemix.NewKeyManager(conf, bccsp.EidNymRhNym, csp)
			if err != nil {
				return nil, err
			}

			return NewKeyManager(ikm, identityStoreService), nil
		},
	)
}
