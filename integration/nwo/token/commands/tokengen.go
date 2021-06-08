/*
Copyright IBM Corp. All Rights Reserved.

SPDX-License-Identifier: Apache-2.0
*/
package commands

type CertifierKeygen struct {
	Driver string
	PPPath string
	Output string
}

func (c CertifierKeygen) SessionName() string {
	return "tokengen-certifier-keygen"
}

func (c CertifierKeygen) Args() []string {
	return []string{
		"certifier-keygen",
		"--driver", c.Driver,
		"--pppath", c.PPPath,
		"--output", c.Output,
	}
}
