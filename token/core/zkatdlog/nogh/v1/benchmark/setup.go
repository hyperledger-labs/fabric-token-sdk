/*
Copyright IBM Corp. All Rights Reserved.

SPDX-License-Identifier: Apache-2.0
*/

package benchmark

import (
	"context"
	"crypto/ecdsa"
	"crypto/elliptic"
	"crypto/rand"
	"fmt"
	"os"
	"path/filepath"

	"github.com/IBM/idemix/bccsp/types"
	math "github.com/IBM/mathlib"
	"github.com/hyperledger-labs/fabric-smart-client/pkg/utils/errors"
	"github.com/hyperledger-labs/fabric-smart-client/platform/fabric/core/generic/msp/x509"
	math2 "github.com/hyperledger-labs/fabric-token-sdk/token/core/common/crypto/math"
	"github.com/hyperledger-labs/fabric-token-sdk/token/core/zkatdlog/nogh/v1/setup"
	"github.com/hyperledger-labs/fabric-token-sdk/token/driver"
	"github.com/hyperledger-labs/fabric-token-sdk/token/services/identity"
	idemix2 "github.com/hyperledger-labs/fabric-token-sdk/token/services/identity/idemix"
	"github.com/hyperledger-labs/fabric-token-sdk/token/services/identity/idemix/crypto"
	"github.com/hyperledger-labs/fabric-token-sdk/token/services/identity/storage/kvs"
	ix509 "github.com/hyperledger-labs/fabric-token-sdk/token/services/identity/x509"
	crypto2 "github.com/hyperledger-labs/fabric-token-sdk/token/services/identity/x509/crypto"
)

type SetupConfiguration struct {
	PP            *setup.PublicParams
	OwnerIdentity *OwnerIdentity
	AuditorSigner *Signer
	IssuerSigner  *Signer
}

type SetupConfigurations struct {
	Configurations map[string]*SetupConfiguration
}

func NewSetupConfigurations(idemixTestdataPath string, bits []uint64, curveIDs []math.CurveID) (*SetupConfigurations, error) {
	configurations := map[string]*SetupConfiguration{}
	for _, curveID := range curveIDs {
		var ipk []byte
		var err error
		var oID *OwnerIdentity
		switch curveID {
		case math.BN254:
			idemixPath := filepath.Join(idemixTestdataPath, "bn254", "idemix")
			ipk, err = os.ReadFile(filepath.Join(idemixPath, "msp", "IssuerPublicKey"))
			if err != nil {
				return nil, err
			}
			oID, err = loadOwnerIdentity(context.Background(), idemixPath, curveID)
			if err != nil {
				return nil, err
			}
		case math.BLS12_381_BBS_GURVY:
			fallthrough
		case math2.BLS12_381_BBS_GURVY_FAST_RNG:
			idemixPath := filepath.Join(idemixTestdataPath, "bls12_381_bbs", "idemix")
			ipk, err = os.ReadFile(filepath.Join(idemixPath, "msp", "IssuerPublicKey"))
			if err != nil {
				return nil, err
			}
			oID, err = loadOwnerIdentity(context.Background(), idemixPath, curveID)
			if err != nil {
				return nil, err
			}
		default:
			return nil, errors.Errorf("curveID [%d] not found", curveID)
		}

		auditorSigner, err := PrepareECDSASigner()
		if err != nil {
			return nil, err
		}
		issuerSigner, err := NewECDSASigner()
		if err != nil {
			return nil, err
		}

		for _, bit := range bits {
			pp, err := setup.Setup(bit, ipk, curveID)
			if err != nil {
				return nil, err
			}
			issuerID, err := issuerSigner.Serialize()
			if err != nil {
				return nil, err
			}
			pp.AddIssuer(issuerID)
			auditorID, err := auditorSigner.Serialize()
			if err != nil {
				return nil, err
			}
			pp.AddAuditor(auditorID)
			configurations[key(bit, curveID)] = &SetupConfiguration{
				PP:            pp,
				OwnerIdentity: oID,
				AuditorSigner: auditorSigner,
				IssuerSigner:  issuerSigner,
			}
		}
	}
	return &SetupConfigurations{
		Configurations: configurations,
	}, nil
}

func (c *SetupConfigurations) GetPublicParams(bits uint64, curveID math.CurveID) (*setup.PublicParams, error) {
	configuration, ok := c.Configurations[key(bits, curveID)]
	if !ok {
		return nil, fmt.Errorf("configuration not found")
	}
	return configuration.PP, nil
}

func (c *SetupConfigurations) GetSetupConfiguration(bits uint64, curveID math.CurveID) (*SetupConfiguration, error) {
	configuration, ok := c.Configurations[key(bits, curveID)]
	if !ok {
		return nil, fmt.Errorf("configuration not found")
	}
	return configuration, nil
}

func key(bits uint64, curveID math.CurveID) string {
	return fmt.Sprintf("%d-%d", bits, curveID)
}

type OwnerIdentity struct {
	ID        driver.Identity
	AuditInfo *crypto.AuditInfo
	Signer    driver.SigningIdentity
}

func loadOwnerIdentity(ctx context.Context, dir string, curveID math.CurveID) (*OwnerIdentity, error) {
	backend, err := kvs.NewInMemory()
	if err != nil {
		return nil, err
	}
	config, err := crypto.NewConfig(dir)
	if err != nil {
		return nil, err
	}
	keyStore, err := crypto.NewKeyStore(curveID, kvs.Keystore(backend))
	if err != nil {
		return nil, err
	}
	cryptoProvider, err := crypto.NewBCCSP(keyStore, curveID)
	if err != nil {
		return nil, err
	}
	p, err := idemix2.NewKeyManager(config, types.EidNymRhNym, cryptoProvider)
	if err != nil {
		return nil, err
	}

	identityDescriptor, err := p.Identity(ctx, nil)
	if err != nil {
		return nil, err
	}
	id := identityDescriptor.Identity
	audit := identityDescriptor.AuditInfo

	auditInfo, err := p.DeserializeAuditInfo(ctx, audit)
	if err != nil {
		return nil, err
	}
	err = auditInfo.Match(ctx, id)
	if err != nil {
		return nil, err
	}

	signer, err := p.DeserializeSigningIdentity(ctx, id)
	if err != nil {
		return nil, err
	}

	id, err = identity.WrapWithType(idemix2.IdentityType, id)
	if err != nil {
		return nil, err
	}

	return &OwnerIdentity{
		ID:        id,
		AuditInfo: auditInfo,
		Signer:    signer,
	}, nil
}

func PrepareECDSASigner() (*Signer, error) {
	signer, err := NewECDSASigner()
	if err != nil {
		return nil, err
	}
	return signer, nil
}

type Signer struct {
	SK     *ecdsa.PrivateKey
	Signer driver.Signer
}

func NewECDSASigner() (*Signer, error) {
	// Create ephemeral key and store it in the context
	sk, err := ecdsa.GenerateKey(elliptic.P256(), rand.Reader)
	if err != nil {
		return nil, err
	}
	return &Signer{SK: sk, Signer: crypto2.NewEcdsaSigner(sk)}, nil
}

func (d *Signer) Sign(message []byte) ([]byte, error) {
	return d.Signer.Sign(message)
}

func (d *Signer) Serialize() ([]byte, error) {
	pkRaw, err := x509.PemEncodeKey(&d.SK.PublicKey)
	if err != nil {
		return nil, errors.Wrap(err, "failed marshalling public key")
	}

	wrap, err := identity.WrapWithType(ix509.IdentityType, pkRaw)
	if err != nil {
		return nil, errors.Wrap(err, "failed wrapping identity")
	}

	return wrap, nil
}
