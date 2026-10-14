package api_test

import (
	"os"
	"testing"

	"platform-service/internal/config"
	"platform-service/internal/testhelper"
)

var (
	// testSrv — server mode cloud (default).
	testSrv *testhelper.TestServer
	// testOnPremSrv — server mode on_prem di atas DB yang sama, untuk skenario LICENSE_MANAGED.
	testOnPremSrv *testhelper.TestServer
	// testAdminTok — token platform admin (fake sampai login nyata tersedia di Fase 3).
	testAdminTok string
)

func TestMain(m *testing.M) {
	var cleanup func()
	testSrv, cleanup = testhelper.MustStartTestServer()
	testOnPremSrv = testSrv.MustDerive(testhelper.WithDeploymentMode(config.DeploymentOnPrem))
	testAdminTok = testSrv.Auth.PlatformAdminToken()

	code := m.Run()
	cleanup()
	os.Exit(code)
}
