/*
Copyright IBM Corp. All Rights Reserved.

SPDX-License-Identifier: Apache-2.0
*/

package benchmark

import (
	"github.com/hyperledger-labs/fabric-smart-client/pkg/utils/errors"
	"github.com/hyperledger-labs/fabric-smart-client/platform/view/services/metrics/disabled"
	tokcommon "github.com/hyperledger-labs/fabric-token-sdk/token/core/common"
	v1 "github.com/hyperledger-labs/fabric-token-sdk/token/core/zkatdlog/nogh/v1"
	zkcommon "github.com/hyperledger-labs/fabric-token-sdk/token/core/zkatdlog/nogh/v1/crypto/common"
	issue_pkg "github.com/hyperledger-labs/fabric-token-sdk/token/core/zkatdlog/nogh/v1/issue"
	"github.com/hyperledger-labs/fabric-token-sdk/token/driver"
	drivermock "github.com/hyperledger-labs/fabric-token-sdk/token/driver/mock"
	"github.com/hyperledger-labs/fabric-token-sdk/token/services/logging"
	token2 "github.com/hyperledger-labs/fabric-token-sdk/token/token"
	"go.opentelemetry.io/otel/trace/noop"
)

// IssueSetup holds a ready-to-call IssueService together with the invocation
// parameters for a single benchmark iteration. All fields are pre-computed at
// setup time so that the benchmark loop itself contains only the hot path.
type IssueSetup struct {
	Service   *v1.IssueService
	IssuerID  driver.Identity
	TokenType token2.Type
	Values    []uint64
	Owners    [][]byte
}

// NewIssueSetup constructs an IssueSetup from a SetupConfiguration.
// It stubs WalletService and Deserializer with counterfeiter mocks, wiring the
// real IssuerSigner from the configuration so that GenerateZKIssue exercises
// the actual ECDSA signing path.
func NewIssueSetup(conf *SetupConfiguration) (*IssueSetup, error) {
	ppm, err := tokcommon.NewPublicParamsManagerFromParams(conf.PP)
	if err != nil {
		return nil, errors.Wrap(err, "failed creating public params manager for issue setup")
	}

	issuerWallet := &drivermock.IssuerWallet{}
	issuerWallet.GetSignerReturns(conf.IssuerSigner.Signer, nil)

	walletService := &drivermock.WalletService{}
	walletService.IssuerWalletReturns(issuerWallet, nil)

	// GetAuditInfo is called twice inside Issue(): once per recipient and once
	// for the issuer identity. Returning fixed bytes is sufficient because the
	// audit info is stored in the metadata but not validated within Issue().
	deserializer := &drivermock.Deserializer{}
	deserializer.GetAuditInfoReturns([]byte("auditInfo"), nil)

	issuerID, err := conf.IssuerSigner.Serialize()
	if err != nil {
		return nil, errors.Wrap(err, "failed serializing issuer identity for issue setup")
	}

	service := v1.NewIssueService(
		logging.MustGetLogger(),
		ppm,
		walletService,
		deserializer,
		v1.NewMetrics(&disabled.Provider{}),
		nil, // TokensUpgradeService is only used when issuerIdentity.IsNone()
	)

	return &IssueSetup{
		Service:   service,
		IssuerID:  issuerID,
		TokenType: "ABC",
		Values:    []uint64{100},
		Owners:    [][]byte{conf.OwnerIdentity.ID},
	}, nil
}

// AuditCheckSetup holds a ready-to-call AuditorService together with a
// pre-built TokenRequest and TokenRequestMetadata for a single issue action.
// The request is generated once at setup time from real ZK proof material so
// that AuditorCheck exercises the actual Pedersen commitment arithmetic on
// every benchmark iteration.
type AuditCheckSetup struct {
	Service  *v1.AuditorService
	Request  *driver.TokenRequest
	Metadata *driver.TokenRequestMetadata
	Anchor   driver.TokenRequestAnchor
}

// NewAuditCheckSetup constructs an AuditCheckSetup from a SetupConfiguration.
// It generates a real ZK issue action using the crypto layer directly, then
// wires an AuditorService with a mock Deserializer (MatchIdentity always
// succeeds) and a mock TokenCommitmentLoader (unused for issue-only requests).
func NewAuditCheckSetup(conf *SetupConfiguration) (*AuditCheckSetup, error) {
	// Build a real issue action using the crypto layer so that the Pedersen
	// commitment in the action is genuine and AuditorCheck performs real arithmetic.
	issuerID := conf.IssuerSigner.ID
	signer := &zkcommon.WrappedSigningIdentity{
		Identity: issuerID,
		Signer:   conf.IssuerSigner.Signer,
	}

	values := []uint64{100}
	owners := [][]byte{conf.OwnerIdentity.ID}

	issuer := issue_pkg.NewIssuer("ABC", signer, conf.PP)
	issueAction, zkMeta, err := issuer.GenerateZKIssue(values, owners)
	if err != nil {
		return nil, errors.Wrap(err, "failed generating ZK issue for audit check setup")
	}

	issueActionRaw, err := issueAction.Serialize()
	if err != nil {
		return nil, errors.Wrap(err, "failed serializing issue action for audit check setup")
	}

	metaBytes, err := zkMeta[0].Serialize()
	if err != nil {
		return nil, errors.Wrap(err, "failed serializing token metadata for audit check setup")
	}

	request := &driver.TokenRequest{
		Issues: [][]byte{issueActionRaw},
	}

	// The issuer identity stored in the action (ia.Issuer) equals issuerID.
	// Setting IssueMetadata.Issuer.Identity to the same value satisfies the
	// identity-equality check inside audit.GetAuditInfoForIssues.
	metadata := &driver.TokenRequestMetadata{
		Issues: []*driver.IssueMetadata{
			{
				Issuer: driver.AuditableIdentity{
					Identity:  issuerID,
					AuditInfo: conf.IssuerSigner.AuditInfo,
				},
				Outputs: []*driver.IssueOutputMetadata{
					{
						OutputMetadata: metaBytes,
						Receivers: []*driver.AuditableIdentity{
							{
								Identity:  conf.OwnerIdentity.ID,
								AuditInfo: conf.OwnerIdentity.AuditInfo,
							},
						},
					},
				},
			},
		},
	}

	ppm, err := tokcommon.NewPublicParamsManagerFromParams(conf.PP)
	if err != nil {
		return nil, errors.Wrap(err, "failed creating public params manager for audit check setup")
	}

	// MatchIdentity always returns nil so identity verification is a no-op.
	// The benchmark focuses on the Pedersen commitment arithmetic in InspectOutput,
	// not on identity-matching overhead.
	deserializer := &drivermock.Deserializer{}
	deserializer.MatchIdentityReturns(nil)

	service := v1.NewAuditorService(
		logging.MustGetLogger(),
		ppm,
		deserializer,
		v1.NewMetrics(&disabled.Provider{}),
		noop.NewTracerProvider(),
	)

	return &AuditCheckSetup{
		Service:  service,
		Request:  request,
		Metadata: metadata,
		Anchor:   "benchmark-anchor",
	}, nil
}
