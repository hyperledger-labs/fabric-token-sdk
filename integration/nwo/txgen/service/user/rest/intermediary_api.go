/*
Copyright IBM Corp. All Rights Reserved.

SPDX-License-Identifier: Apache-2.0
*/

package rest

type LoginRequest struct {
	Username string `json:"username"`
	Password string `json:"password"` //nolint:gosec
}

type LoginResponse struct {
	Token string `json:"token"`
}

type BalanceResponse struct {
	Balance struct {
		Type     string `json:"type"`
		Quantity string `json:"quantity"`
	} `json:"balance"`
}
