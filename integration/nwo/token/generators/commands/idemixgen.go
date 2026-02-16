/*
Copyright IBM Corp. All Rights Reserved.

SPDX-License-Identifier: Apache-2.0
*/

package commands

type CAKeyGen struct {
	NetworkPrefix string
	Output        string
	Curve         string
	Aries         bool
}

func (c CAKeyGen) SessionName() string {
	return c.NetworkPrefix + "-idemixgen-ca-key-gen"
}

func (c CAKeyGen) Args() []string {
	args := []string{
		"ca-keygen",
		"--output", c.Output,
		"--curve", c.Curve,
	}
	if c.Aries {
		args = append(args, "--aries")
	}

	return args
}

type SignerConfig struct {
	NetworkPrefix    string
	CAInput          string
	Output           string
	OrgUnit          string
	Admin            bool
	EnrollmentID     string
	RevocationHandle string
	Curve            string
	Aries            bool
}

func (c SignerConfig) SessionName() string {
	return c.NetworkPrefix + "idemixgen-signerconfig"
}

func (c SignerConfig) Args() []string {
	args := []string{
		"signerconfig",
		"--ca-input", c.CAInput,
		"--output", c.Output,
		"--admin",
		"-u", c.OrgUnit,
		"-e", c.EnrollmentID,
		"-r", c.RevocationHandle,
		"--curve", c.Curve,
	}
	if c.Aries {
		args = append(args, "--aries")
	}

	return args
}
