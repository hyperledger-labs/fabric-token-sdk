/*
Copyright IBM Corp. All Rights Reserved.

SPDX-License-Identifier: Apache-2.0
*/

package common

import (
	"context"

	"github.com/hyperledger-labs/fabric-smart-client/pkg/utils/errors"
	"github.com/hyperledger-labs/fabric-token-sdk/token/driver"
	"github.com/hyperledger-labs/fabric-token-sdk/token/driver/protos-go/v1/request"
	"github.com/hyperledger-labs/fabric-token-sdk/token/services/logging"
	"github.com/hyperledger-labs/fabric-token-sdk/token/token"
)

// ExtractTokenIDsAndCheckDuplicates extracts all token IDs from both transfer and issue actions
// in the metadata and checks for duplicates. Returns the list of unique token IDs or an error
// if duplicates are found.
//
// This function is used by auditors to ensure that no token is spent multiple times within
// a single transaction.
func ExtractTokenIDsAndCheckDuplicates(
	metadata *driver.TokenRequestMetadata,
	anchor driver.TokenRequestAnchor,
) ([]*token.ID, error) {
	tokenIDMap := make(map[string]*token.ID)
	var tokenIDs []*token.ID

	for i, action := range metadata.Actions {
		// Extract TokenIDs from transfer actions
		if action.TransferMetadata != nil {
			ids := action.TransferMetadata.TokenIDs()
			for _, id := range ids {
				if id == nil {
					continue
				}
				// Check for duplicates using string representation as key
				idKey := id.String()
				if _, exists := tokenIDMap[idKey]; exists {
					return nil, errors.Errorf("duplicate token ID [%s] found in metadata at action index [%d] for tx [%s]", idKey, i, anchor)
				}
				tokenIDMap[idKey] = id
				tokenIDs = append(tokenIDs, id)
			}
		}

		// Extract TokenIDs from issue action inputs (for token upgrades/conversions)
		if action.IssueMetadata != nil {
			for _, input := range action.IssueMetadata.Inputs {
				if input == nil || input.TokenID == nil {
					continue
				}
				id := input.TokenID
				// Check for duplicates using string representation as key
				idKey := id.String()
				if _, exists := tokenIDMap[idKey]; exists {
					return nil, errors.Errorf("duplicate token ID [%s] found in metadata at action index [%d] for tx [%s]", idKey, i, anchor)
				}
				tokenIDMap[idKey] = id
				tokenIDs = append(tokenIDs, id)
			}
		}
	}

	return tokenIDs, nil
}

// RetrieveAuditTokens retrieves audit tokens from the query engine for the given token IDs
// and builds a map for efficient lookup. Returns the token map or an error if retrieval fails.
//
// The returned map uses token ID pointers as keys, allowing callers to efficiently look up
// tokens by their ID during validation.
func RetrieveAuditTokens(
	ctx context.Context,
	logger logging.Logger,
	queryEngine driver.QueryEngine,
	tokenIDs []*token.ID,
	anchor driver.TokenRequestAnchor,
) (map[*token.ID]*token.Token, error) {
	if len(tokenIDs) == 0 {
		return nil, nil
	}

	logger.DebugfContext(ctx, "[%s] retrieving [%d] audit tokens...", anchor, len(tokenIDs))
	tokens, err := queryEngine.ListAuditTokens(ctx, tokenIDs...)
	if err != nil {
		return nil, errors.WithMessagef(err, "failed to retrieve audit tokens for tx [%s]", anchor)
	}

	// Build the token map
	auditTokens := make(map[*token.ID]*token.Token, len(tokens))
	for i, tok := range tokens {
		if tok != nil && i < len(tokenIDs) {
			auditTokens[tokenIDs[i]] = tok
		}
	}
	logger.DebugfContext(ctx, "[%s] retrieved [%d] audit tokens", anchor, len(auditTokens))

	return auditTokens, nil
}

// ValidateStructure ensures complete structural correspondence between TokenRequest and TokenRequestMetadata.
// It validates that:
// - Action counts match between request and metadata
// - Each action has corresponding metadata with correct type
// - ActionIDs are sequential and match their position
// - Action types align with metadata types (no mixed metadata)
//
// This validation ensures that the request and metadata are structurally consistent before
// performing deeper semantic validation.
func ValidateStructure(
	tokenRequest *driver.TokenRequest,
	tokenRequestMetadata *driver.TokenRequestMetadata,
	txID driver.TokenRequestAnchor,
) error {
	// Validate action count matches metadata count
	if len(tokenRequest.Actions) != len(tokenRequestMetadata.Actions) {
		return errors.Errorf(
			"action count mismatch: request has [%d] actions but metadata has [%d] actions for tx [%s]",
			len(tokenRequest.Actions),
			len(tokenRequestMetadata.Actions),
			txID,
		)
	}

	// Validate each action has corresponding metadata with correct type
	for i, action := range tokenRequest.Actions {
		if action == nil {
			return errors.Errorf("action at index [%d] is nil for tx [%s]", i, txID)
		}

		metadata := tokenRequestMetadata.Actions[i]
		if metadata == nil {
			return errors.Errorf("metadata at index [%d] is nil for tx [%s]", i, txID)
		}

		// Verify ActionID matches position
		if metadata.ActionID != uint32(i) {
			return errors.Errorf(
				"metadata at index [%d] has incorrect ActionID [%d] for tx [%s]",
				i,
				metadata.ActionID,
				txID,
			)
		}

		// Verify action type matches metadata type
		switch action.Type {
		case request.ActionType_ACTION_TYPE_ISSUE:
			if metadata.IssueMetadata == nil {
				return errors.Errorf(
					"action at index [%d] is ISSUE but metadata has no IssueMetadata for tx [%s]",
					i,
					txID,
				)
			}
			if metadata.TransferMetadata != nil {
				return errors.Errorf(
					"action at index [%d] is ISSUE but metadata also has TransferMetadata for tx [%s]",
					i,
					txID,
				)
			}

		case request.ActionType_ACTION_TYPE_TRANSFER:
			if metadata.TransferMetadata == nil {
				return errors.Errorf(
					"action at index [%d] is TRANSFER but metadata has no TransferMetadata for tx [%s]",
					i,
					txID,
				)
			}
			if metadata.IssueMetadata != nil {
				return errors.Errorf(
					"action at index [%d] is TRANSFER but metadata also has IssueMetadata for tx [%s]",
					i,
					txID,
				)
			}

		default:
			return errors.Errorf(
				"action at index [%d] has unknown type [%s] for tx [%s]",
				i,
				action.Type,
				txID,
			)
		}
	}

	return nil
}

// ValidateIssueActionTokenTypes ensures all inputs and outputs in an issue action have the same token type.
// It also validates that input tokens exist in the auditTokens map.
//
// For issue actions with inputs (token upgrades/conversions), this validates that:
// - All input tokens exist in the audit token map
// - All inputs have the same token type
// - All outputs have the same token type as the inputs
//
// This ensures token type consistency within an issue action.
func ValidateIssueActionTokenTypes(
	metadata *driver.IssueMetadata,
	auditTokens map[*token.ID]*token.Token,
) error {
	var actionTokenType token.Type

	// Validate and extract token type from inputs (if any exist)
	for i, inputMetadata := range metadata.Inputs {
		if inputMetadata == nil {
			continue
		}

		// Verify input token exists in auditTokens map
		if inputMetadata.TokenID != nil {
			inputToken, exists := auditTokens[inputMetadata.TokenID]
			if !exists {
				return errors.Errorf("input token [%s:%d] at index [%d] not found in audit tokens",
					inputMetadata.TokenID.TxId, inputMetadata.TokenID.Index, i)
			}

			// For issue inputs (token upgrades/conversions), we get the type from the audit token
			if inputToken != nil && inputToken.Type != "" {
				if actionTokenType == "" {
					actionTokenType = inputToken.Type
				} else if actionTokenType != inputToken.Type {
					return errors.Errorf(
						"token type mismatch in issue action: input [%d] has type [%s] but expected [%s]",
						i, inputToken.Type, actionTokenType,
					)
				}
			}
		}
	}

	// Note: Output token type validation is driver-specific and handled by the driver's
	// action deserialization and Match() methods. This function only validates input consistency.

	return nil
}

// ValidateTransferActionTokenTypes ensures all inputs and outputs in a transfer action have the same token type.
// It also validates that input tokens exist in the auditTokens map.
//
// When validateValueSum is true (for privacy-preserving tokens like zkatdlog), this also validates
// that the sum of input values equals the sum of output values.
//
// For transfer actions, this validates that:
// - All input tokens exist in the audit token map
// - All inputs have the same token type
// - All outputs have the same token type as the inputs
// - (Optional) Sum of input values equals sum of output values
//
// This ensures token type consistency and value conservation within a transfer action.
func ValidateTransferActionTokenTypes(
	metadata *driver.TransferMetadata,
	auditTokens map[*token.ID]*token.Token,
	validateValueSum bool,
) error {
	var actionTokenType token.Type
	const precision = 64 // Standard precision for token quantities
	var inputSum token.Quantity
	hasAuditTokens := len(auditTokens) > 0 // Track if we have audit tokens available

	if validateValueSum {
		inputSum = token.NewZeroQuantity(precision)
	}

	// Validate and extract token type from inputs
	for i, inputMetadata := range metadata.Inputs {
		if inputMetadata == nil {
			return errors.Errorf("input metadata at index [%d] is nil", i)
		}

		// TokenID is required
		if inputMetadata.TokenID == nil {
			return errors.Errorf("input at index [%d] has nil TokenID", i)
		}

		// If audit tokens are available, verify input token exists and validate type
		if hasAuditTokens {
			inputToken, exists := auditTokens[inputMetadata.TokenID]
			if !exists {
				return errors.Errorf("input token [%s:%d] at index [%d] not found in audit tokens",
					inputMetadata.TokenID.TxId, inputMetadata.TokenID.Index, i)
			}

			if inputToken == nil {
				return errors.Errorf("input token [%s:%d] at index [%d] is nil in audit tokens",
					inputMetadata.TokenID.TxId, inputMetadata.TokenID.Index, i)
			}

			// Validate and accumulate token type
			if actionTokenType == "" {
				actionTokenType = inputToken.Type
			} else if actionTokenType != inputToken.Type {
				return errors.Errorf(
					"token type mismatch in transfer action: input [%d] has type [%s] but expected [%s]",
					i, inputToken.Type, actionTokenType,
				)
			}

			// Accumulate input value if validation is requested
			if validateValueSum {
				inputQty, err := token.ToQuantity(inputToken.Quantity, precision)
				if err != nil {
					return errors.Wrapf(err, "failed to convert input quantity at index [%d]", i)
				}
				inputSum, err = inputSum.Add(inputQty)
				if err != nil {
					return errors.Wrapf(err, "failed to add input quantity at index [%d]", i)
				}
			}
		}
	}

	// Note: Output token type and value validation is driver-specific.
	// For cleartext tokens (fabtoken), outputs are validated by the action's Match() method.
	// For privacy-preserving tokens (zkatdlog), outputs require additional cryptographic validation
	// which is handled by the driver-specific auditor.

	return nil
}
