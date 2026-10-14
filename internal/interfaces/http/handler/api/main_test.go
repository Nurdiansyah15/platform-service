package api_test

import (
	"os"
	"testing"

	"platform-service/internal/testhelper"
)

var testSrv *testhelper.TestServer

func TestMain(m *testing.M) {
	var cleanup func()
	testSrv, cleanup = testhelper.MustStartTestServer()

	code := m.Run()
	cleanup()
	os.Exit(code)
}
