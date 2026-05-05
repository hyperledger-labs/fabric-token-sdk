/*
Copyright IBM Corp. All Rights Reserved.

SPDX-License-Identifier: Apache-2.0
*/

package ttx

import "github.com/hyperledger-labs/fabric-token-sdk/token"

// EndorsementsOpts is used to configure the CollectEndorsementsView
type EndorsementsOpts struct {
	// SkipAuditing set it to true to skip the auditing phase
	SkipAuditing bool
	// SkipAuditorSignatureVerification set it to true to skip the verification of the auditor signature
	SkipAuditorSignatureVerification bool
	// SkipApproval set it to true to skip the approval phase
	SkipApproval bool
	// SkipDistributeEnv set it to true to skip the distribution phase
	SkipDistributeEnv bool
	// External Signers
	ExternalWalletSigners map[string]ExternalWalletSigner
	// PolicySigners, when non-nil, restricts signature collection for policy
	// identities to the listed component identities.  Component identities not
	// in the list produce a nil slot in the PolicySignature, which is valid for
	// OR branches.  When nil, all component identities are contacted (default).
	PolicySigners []token.Identity
}

func (o *EndorsementsOpts) ExternalWalletSigner(id string) ExternalWalletSigner {
	if o.ExternalWalletSigners == nil {
		return nil
	}

	return o.ExternalWalletSigners[id]
}

// EndorsementsOpt is a function that configures a EndorsementsOpts
type EndorsementsOpt func(*EndorsementsOpts) error

// CompileCollectEndorsementsOpts compiles the given list of ServiceOption
func CompileCollectEndorsementsOpts(opts ...EndorsementsOpt) (*EndorsementsOpts, error) {
	txOptions := &EndorsementsOpts{}
	for _, opt := range opts {
		if err := opt(txOptions); err != nil {
			return nil, err
		}
	}

	return txOptions, nil
}

// WithSkipAuditing to skip auditing
func WithSkipAuditing() EndorsementsOpt {
	return func(o *EndorsementsOpts) error {
		o.SkipAuditing = true

		return nil
	}
}

// WithSkipAuditorSignatureVerification to skip auditor signature verification
func WithSkipAuditorSignatureVerification() EndorsementsOpt {
	return func(o *EndorsementsOpts) error {
		o.SkipAuditorSignatureVerification = true

		return nil
	}
}

// WithSkipApproval to skip approval
func WithSkipApproval() EndorsementsOpt {
	return func(o *EndorsementsOpts) error {
		o.SkipApproval = true

		return nil
	}
}

// WithSkipDistributeEnv to skip approval
func WithSkipDistributeEnv() EndorsementsOpt {
	return func(o *EndorsementsOpts) error {
		o.SkipDistributeEnv = true

		return nil
	}
}

// WithPolicySigners restricts signature collection for PolicyIdentity owners to
// the given component identities.  Unlisted components produce nil slots in the
// PolicySignature, satisfying OR branches without contacting the other parties.
func WithPolicySigners(signers ...token.Identity) EndorsementsOpt {
	return func(o *EndorsementsOpts) error {
		o.PolicySigners = append(o.PolicySigners, signers...)

		return nil
	}
}

func WithExternalWalletSigner(walletID string, ews ExternalWalletSigner) EndorsementsOpt {
	return func(o *EndorsementsOpts) error {
		if o.ExternalWalletSigners == nil {
			o.ExternalWalletSigners = map[string]ExternalWalletSigner{}
		}
		o.ExternalWalletSigners[walletID] = ews

		return nil
	}
}
