package persistence_test

import (
	"database/sql"
	"os"
	"testing"

	"platform-service/internal/testhelper"
)

var testDB *sql.DB

func TestMain(m *testing.M) {
	var cleanup func()
	testDB, cleanup = testhelper.MustStartTestDB()

	code := m.Run()
	cleanup()
	os.Exit(code)
}
