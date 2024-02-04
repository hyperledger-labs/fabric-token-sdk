package sql

import (
	"testing"

	"github.com/test-go/testify/assert"
)

func TestParamsSqlite(t *testing.T) {
	tempDir := t.TempDir()

	initSqlite(t, tempDir, "PublicParams")
	t.Run("PublicParams", func(xt *testing.T) {
		defer Transactions.Close()
		TPublicParams(xt, PublicParams)
	})
}

func TestParamsSqliteMemory(t *testing.T) {

	t.Run("PublicParams", func(xt *testing.T) {
		initSqliteMemory(t, "PublicParams")
		defer Transactions.Close()
		TPublicParams(xt, PublicParams)
	})
}

func TestPostgres(t *testing.T) {
	// if os.Getenv("TESTCONTAINERS") != "true" {
	// 	t.Skip("set environment variable TESTCONTAINERS to true to include postgres test")
	// }
	if testing.Short() {
		t.Skip("skipping postgres test in short mode")
	}

	terminate, pgConnStr := startPostgresContainer(t)
	defer terminate()

	initPostgres(t, pgConnStr, "PublicParams")
	t.Run("PublicParams", func(xt *testing.T) {
		defer Transactions.Close()
		TPublicParams(xt, PublicParams)
	})
}

type params interface {
	GetRawPublicParams() ([]byte, error)
	StorePublicParams([]byte) error
}

func TPublicParams(t *testing.T, db params) {
	b := []byte("test bytes")
	b1 := []byte("test bytes1")

	_, err := db.GetRawPublicParams()
	assert.Error(t, err) // not found

	err = db.StorePublicParams(b)
	assert.NoError(t, err)

	res, err := db.GetRawPublicParams()
	assert.NoError(t, err) // not found
	assert.Equal(t, res, b)

	err = db.StorePublicParams(b1)
	assert.NoError(t, err)

	res, err = db.GetRawPublicParams()
	assert.NoError(t, err) // not found
	assert.Equal(t, res, b1)
}
