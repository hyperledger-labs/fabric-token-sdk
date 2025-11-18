/*
Copyright IBM Corp. All Rights Reserved.

SPDX-License-Identifier: Apache-2.0
*/

package cc

// DefaultParams defines the template for the public parameters burned into
// the token chaincode.
const DefaultParams = `
package tcc

var Params = "{{ Params }}"
`
