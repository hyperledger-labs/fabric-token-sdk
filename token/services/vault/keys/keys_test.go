/*
Copyright IBM Corp. All Rights Reserved.

SPDX-License-Identifier: Apache-2.0
*/

package keys

import (
	"fmt"
	"github.com/stretchr/testify/assert"
	"testing"
)

func TestKey(t *testing.T) {
	assert.NoError(t, ValidateKey("seyJMaW5lYXJJRCI6ImM4ZWZkNGEzLWE4YTMtNDM4YS1iNGQ3LWJlOWVhN2NmZTNhMyIsIkFkZHJlc3MiOiIiLCJWYWx1YXRpb24iOjB9"))

	assert.NoError(t, ValidateKey(
		fmt.Sprintf(
			"%setoken%stestchannelzkatidemix%seyJMaW5lYXJJRCI6ImM4ZWZkNGEzLWE4YTMtNDM4YS1iNGQ3LWJlOWVhN2NmZTNhMyIsIkFkZHJlc3MiOiIiLCJWYWx1YXRpb24iOjB9%s04020c87bb3ac9074cea738585d81b1e07965fa74a8db453629d8ffa43529039%s0%s",
			string(rune(minUnicodeRuneValue)),
			string(rune(minUnicodeRuneValue)),
			string(rune(minUnicodeRuneValue)),
			string(rune(minUnicodeRuneValue)),
			string(rune(minUnicodeRuneValue)),
			string(rune(minUnicodeRuneValue)),
		)))

}
