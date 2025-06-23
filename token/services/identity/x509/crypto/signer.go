/*
Copyright IBM Corp. All Rights Reserved.

SPDX-License-Identifier: Apache-2.0
*/

package crypto

import (
	"crypto"
	"io"

	"github.com/hyperledger/fabric-lib-go/bccsp"
	"github.com/pkg/errors"
)

// SKIBasedSigner implements a crypto.Signer based on the bccsp
type SKIBasedSigner struct {
	csp bccsp.BCCSP
	SKI []byte
	pk  crypto.PublicKey
}

// NewSKIBasedSigner returns a new SKIBasedSigner
func NewSKIBasedSigner(csp bccsp.BCCSP, ski []byte, pk crypto.PublicKey) (crypto.Signer, error) {
	// Validate arguments
	if csp == nil {
		return nil, errors.New("bccsp instance must be different from nil")
	}
	if len(ski) == 0 {
		return nil, errors.New("SKI is empty")
	}
	if pk == nil {
		return nil, errors.New("PK is nil")
	}

	return &SKIBasedSigner{csp: csp, SKI: ski, pk: pk}, nil
}

// Public returns the public key corresponding to the opaque,
// private key.
func (s *SKIBasedSigner) Public() crypto.PublicKey {
	return s.pk
}

// Sign signs digest with the private key, possibly using entropy from rand.
// For an (EC)DSA key, it should be a DER-serialised, ASN.1 signature
// structure.
//
// Hash implements the SignerOpts interface and, in most cases, one can
// simply pass in the hash function used as opts. Sign may also attempt
// to type assert opts to other types in order to obtain algorithm
// specific values. See the documentation in each package for details.
//
// Note that when a signature of a hash of a larger message is needed,
// the caller is responsible for hashing the larger message and passing
// the hash (as digest) and the hash function (as opts) to Sign.
func (s *SKIBasedSigner) Sign(rand io.Reader, digest []byte, opts crypto.SignerOpts) ([]byte, error) {
	key, err := s.csp.GetKey(s.SKI)
	if err != nil {
		return nil, errors.WithMessagef(err, "failed to retrieve key for SKI [%s]", s.SKI)
	}
	return s.csp.Sign(key, digest, opts)
}
