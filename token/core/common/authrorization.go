/*
Copyright IBM Corp. All Rights Reserved.

SPDX-License-Identifier: Apache-2.0
*/

package common

import (
	"context"

	"github.com/hyperledger-labs/fabric-token-sdk/token"
	"github.com/hyperledger-labs/fabric-token-sdk/token/driver"
	"github.com/hyperledger-labs/fabric-token-sdk/token/services/identity"
	"github.com/hyperledger-labs/fabric-token-sdk/token/services/logging"
	token2 "github.com/hyperledger-labs/fabric-token-sdk/token/token"
	"github.com/pkg/errors"
)

type Authorization interface {
	// IsMine returns true if the passed token is owned by an owner wallet.
	// It returns the ID of the owner wallet and any additional owner identifier, if supported.
	// It is possible that the wallet ID is empty and the additional owner identifier list is not.
	IsMine(ctx context.Context, tok *token2.Token) (string, []string, bool)
	// AmIAnAuditor return true if the passed TMS contains an auditor wallet for any of the auditor identities
	// defined in the public parameters of the passed TMS.
	AmIAnAuditor(ctx context.Context) bool
	// Issued returns true if the passed issuer issued the passed token
	Issued(ctx context.Context, issuer token.Identity, tok *token2.Token) bool
}

// WalletBasedAuthorization is a wallet-based authorization implementation
type WalletBasedAuthorization struct {
	Logger           logging.Logger
	PublicParameters driver.PublicParameters
	WalletService    driver.WalletService
	amIAnAuditor     bool
}

func NewTMSAuthorization(ctx context.Context, logger logging.Logger, publicParameters driver.PublicParameters, walletService driver.WalletService) *WalletBasedAuthorization {
	amIAnAuditor := false
	var errs []error
	for _, identity := range publicParameters.Auditors() {
		_, err := walletService.AuditorWallet(context.Background(), identity)
		if err == nil {
			amIAnAuditor = true
			break
		}
		errs = append(errs, errors.Wrapf(err, "I'm not this auditor identity [%s]", identity))
	}
	logger.DebugfContext(ctx, "am I an auditor? [%v], with errs [%v]", amIAnAuditor, errs)
	return &WalletBasedAuthorization{Logger: logger, PublicParameters: publicParameters, WalletService: walletService, amIAnAuditor: amIAnAuditor}
}

// IsMine returns true if the passed token is owned by an owner wallet.
// It returns the ID of the owner wallet and no additional owner identifiers.
func (w *WalletBasedAuthorization) IsMine(ctx context.Context, tok *token2.Token) (string, []string, bool) {
	wallet, err := w.WalletService.OwnerWallet(ctx, tok.Owner)
	if err != nil {
		return "", nil, false
	}
	return wallet.ID(), nil, true
}

// AmIAnAuditor return true if the passed TMS contains an auditor wallet for any of the auditor identities
// defined in the public parameters of the passed TMS.
func (w *WalletBasedAuthorization) AmIAnAuditor(ctx context.Context) bool {
	return w.amIAnAuditor
}

func (w *WalletBasedAuthorization) Issued(ctx context.Context, issuer token.Identity, tok *token2.Token) bool {
	_, err := w.WalletService.IssuerWallet(ctx, issuer)
	return err == nil
}

// AuthorizationMultiplexer iterates over multiple authorization checker
type AuthorizationMultiplexer struct {
	authorizations []Authorization
}

// NewAuthorizationMultiplexer returns a new AuthorizationMultiplexer for the passed ownership checkers
func NewAuthorizationMultiplexer(ownerships ...Authorization) *AuthorizationMultiplexer {
	return &AuthorizationMultiplexer{authorizations: ownerships}
}

// IsMine returns true it there exists an authorization checker that returns true
func (o *AuthorizationMultiplexer) IsMine(ctx context.Context, tok *token2.Token) (string, []string, bool) {
	for _, authorization := range o.authorizations {
		walletID, ids, mine := authorization.IsMine(ctx, tok)
		if mine {
			return walletID, ids, true
		}
	}
	return "", nil, false
}

// AmIAnAuditor returns true it there exists an authorization checker that returns true
func (o *AuthorizationMultiplexer) AmIAnAuditor(ctx context.Context) bool {
	for _, authorization := range o.authorizations {
		yes := authorization.AmIAnAuditor(ctx)
		if yes {
			return true
		}
	}
	return false
}

func (o *AuthorizationMultiplexer) Issued(ctx context.Context, issuer token.Identity, tok *token2.Token) bool {
	for _, authorization := range o.authorizations {
		yes := authorization.Issued(ctx, issuer, tok)
		if yes {
			return true
		}
	}
	return false
}

// OwnerType returns the type of owner (e.g. 'idemix' or 'htlc') and the identity bytes
func (o *AuthorizationMultiplexer) OwnerType(raw []byte) (string, []byte, error) {
	owner, err := identity.UnmarshalTypedIdentity(raw)
	if err != nil {
		return "", nil, err
	}
	return owner.Type, owner.Identity, nil
}
