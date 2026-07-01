/*
Copyright IBM Corp. All Rights Reserved.

SPDX-License-Identifier: Apache-2.0
*/

package signers

import (
	"context"
	"fmt"
	"strings"

	"github.com/LFDT-Panurus/panurus/token/services/identity"
	identitydriver "github.com/LFDT-Panurus/panurus/token/services/identity/driver"
	"github.com/LFDT-Panurus/panurus/token/services/identity/idemix"
	"github.com/LFDT-Panurus/panurus/token/services/identity/idemixnym"
	"github.com/LFDT-Panurus/panurus/token/services/identity/x509"
	"github.com/LFDT-Panurus/panurus/token/services/storage/services/cleanup"
)

// Run iterates all entries in the Signers table in batches of batchSize.
// For each signer entry whose identity is NOT found in the Tokens table,
// it derives the SKIs and prints them to stdout.
func Run(ctx context.Context, stores *Stores, batchSize int) error {
	extractor := cleanup.NewSKIExtractor()
	extractor.RegisterProvider(idemix.IdentityTypeString, idemix.NewSKIProvider())
	extractor.RegisterProvider(idemixnym.IdentityTypeString, idemixnym.NewSKIProvider(stores.Identity))
	extractor.RegisterProvider(x509.IdentityTypeString, cleanup.NewNoopSKIProvider())

	offset := 0
	for {
		entries, err := stores.Identity.IterateSigners(ctx, offset, batchSize)
		if err != nil {
			return fmt.Errorf("iterate signers at offset %d: %w", offset, err)
		}

		for _, entry := range entries {
			if err := processEntry(ctx, entry, stores, extractor); err != nil {
				// Log and continue — a single bad entry should not abort the whole scan.
				fmt.Printf("WARN: identity %s: %v\n", entry.IdentityHash, err)
			}
		}

		if len(entries) < batchSize {
			// Last page reached.
			break
		}

		offset += batchSize
	}

	return nil
}

func processEntry(
	ctx context.Context,
	entry identitydriver.SignerEntry,
	stores *Stores,
	extractor *cleanup.SKIExtractor,
) error {
	typed, err := identity.UnmarshalTypedIdentity(entry.Identity)
	if err != nil {
		return fmt.Errorf("unmarshal identity: %w", err)
	}

	ownerType := identity.TypeToString(typed.Type)

	found, err := stores.Token.HasTokenForIdentity(ctx, typed.Identity, ownerType)
	if err != nil {
		return fmt.Errorf("token lookup: %w", err)
	}

	if found {
		// Signer has tokens — not orphaned.
		return nil
	}

	skis, err := extractor.GetSKIsFromIdentity(ctx, typed.Identity, ownerType)
	if err != nil {
		return fmt.Errorf("extract SKIs: %w", err)
	}

	fmt.Printf("%s: [%s]\n", entry.IdentityHash, strings.Join(skis, ", "))

	return nil
}
