/*
Copyright IBM Corp. All Rights Reserved.

SPDX-License-Identifier: Apache-2.0
*/

package auditdb

import "github.com/hashicorp/go-uuid"

func generateUUID() (string, error) {
	return uuid.GenerateUUID()
}
