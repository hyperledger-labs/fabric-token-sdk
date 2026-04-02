/*
Copyright IBM Corp. All Rights Reserved.

SPDX-License-Identifier: Apache-2.0
*/

package rest

type LoginRequest struct {
	Username string `json:"username"`
	//nolint:gosec // this is a login request struct for tests
	Password string `json:"password"`
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
