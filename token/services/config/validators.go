/*
Copyright IBM Corp. All Rights Reserved.

SPDX-License-Identifier: Apache-2.0
*/

package config

type ValidateConfiguration interface {
}

type Validator interface {
	Validate(ValidateConfiguration) error
}
