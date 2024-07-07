/*
Copyright IBM Corp. All Rights Reserved.

SPDX-License-Identifier: Apache-2.0
*/

package metrics

type Reporter interface {
	GetTotalRequests() string
	GetActiveRequests() string
	Summary() string
}
