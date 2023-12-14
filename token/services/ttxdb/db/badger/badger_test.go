/*
Copyright IBM Corp. All Rights Reserved.

SPDX-License-Identifier: Apache-2.0
*/

package badger

import (
	"fmt"
	"os"
	"path/filepath"
	"sort"
	"testing"

	"github.com/hyperledger-labs/fabric-token-sdk/token/services/ttxdb/db/dbtest"
	"github.com/stretchr/testify/assert"
)

func TestDb(t *testing.T) {
	for _, c := range dbtest.Cases {
		db, err := OpenDB(filepath.Join(tempDir, c.Name))
		assert.NoError(t, err)
		assert.NotNil(t, db)
		t.Run(c.Name, func(xt *testing.T) { c.Fn(xt, db) })
		db.Close()
	}
}

func TestKThLexicographicString(t *testing.T) {
	var list []string
	for i := 0; i < 100; i++ {
		list = append(list, kThLexicographicString(26, i))
	}
	sort.Strings(list)
	for i := 0; i < 100; i++ {
		assert.Equal(t, list[i], kThLexicographicString(26, i))
	}
}

var tempDir string

func TestMain(m *testing.M) {
	var err error
	tempDir, err = os.MkdirTemp("", "badger-fsc-test")
	if err != nil {
		fmt.Fprintf(os.Stderr, "failed to create temporary directory: %v", err)
		os.Exit(-1)
	}
	defer os.RemoveAll(tempDir)

	m.Run()
}
