/*
Copyright IBM Corp. All Rights Reserved.

SPDX-License-Identifier: Apache-2.0
*/

package nfttx

//go:generate counterfeiter -o mock/query_service.go -fake-name QueryService . QueryService
//go:generate counterfeiter -o mock/vault.go -fake-name Vault . vault
//go:generate counterfeiter -o mock/selector.go -fake-name Selector . selector
//go:generate counterfeiter -o mock/view_context.go -fake-name Context github.com/hyperledger-labs/fabric-smart-client/platform/view/view.Context
