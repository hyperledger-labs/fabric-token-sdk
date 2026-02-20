/*
Copyright IBM Corp. All Rights Reserved.

SPDX-License-Identifier: Apache-2.0
*/

package pp

import (
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestGenCmd(t *testing.T) {
	cmd := GenCmd()
	assert.NotNil(t, cmd)
	assert.Equal(t, "gen", cmd.Use)
}

func TestUpdateCmd(t *testing.T) {
	cmd := UpdateCmd()
	assert.NotNil(t, cmd)
	assert.Equal(t, "update", cmd.Use)
}

func TestUtilsCmd(t *testing.T) {
	cmd := UtilsCmd()
	assert.NotNil(t, cmd)
	assert.Equal(t, "pp", cmd.Use)
}
