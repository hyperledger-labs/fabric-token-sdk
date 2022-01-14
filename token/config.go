/*
Copyright IBM Corp. All Rights Reserved.

SPDX-License-Identifier: Apache-2.0
*/

package token

import "github.com/hyperledger-labs/fabric-token-sdk/token/driver"

type ConfigManager struct {
	cm driver.ConfigManager
}

func (m *ConfigManager) Certifiers() []string {
	return m.cm.TMS().Certification.Interactive.IDs
}
