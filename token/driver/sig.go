/*
Copyright IBM Corp. All Rights Reserved.

SPDX-License-Identifier: Apache-2.0
*/
package driver

import "github.com/hyperledger-labs/fabric-smart-client/platform/view/view"

type SigService interface {
	GetVerifier(id view.Identity) (Verifier, error)
	GetSigner(id view.Identity) (Signer, error)
}
