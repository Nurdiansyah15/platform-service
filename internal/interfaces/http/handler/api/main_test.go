package api_test

import (
	"context"
	"fmt"
	"os"
	"testing"

	"platform-service/internal/config"
	"platform-service/internal/seeders"
	"platform-service/internal/testhelper"
)

const (
	testPassword = "Secret123!"
	// testConsoleSolution — solution aktif yang dipakai login platform admin di test.
	testConsoleSolution = testhelper.TestConsoleSolutionCode
)

var (
	// testSrv — server mode cloud (default).
	testSrv *testhelper.TestServer
	// testOnPremSrv — server mode on_prem di atas DB yang sama, untuk skenario LICENSE_MANAGED.
	testOnPremSrv *testhelper.TestServer
	// testAdminTok — access token platform admin hasil login sungguhan.
	testAdminTok string
)

func TestMain(m *testing.M) {
	var cleanup func()
	testSrv, cleanup = testhelper.MustStartTestServer()
	testOnPremSrv = testSrv.MustDerive(testhelper.WithDeploymentMode(config.DeploymentOnPrem))
	testAdminTok = mustSetupPlatformAdmin()

	code := m.Run()
	cleanup()
	os.Exit(code)
}

// mustSetupPlatformAdmin menjalankan seeder platform admin (sama dengan produksi),
// lalu login lewat API untuk mendapatkan access token.
func mustSetupPlatformAdmin() string {
	ctx := context.Background()
	// Solution console + katalog permission platform + role company_admin (sama dengan produksi).
	console := seeders.ConsoleSolutionSeeder{Code: testConsoleSolution, SolutionName: "Platform Console (test)"}
	if err := console.Run(ctx, testSrv.DB); err != nil {
		panic(fmt.Sprintf("mustSetupPlatformAdmin: seed console solution: %v", err))
	}

	email := testhelper.UniqueEmail("platform-admin")
	seeder := seeders.PlatformAdminSeeder{Email: email, Password: testPassword, Hasher: testhelper.TestPasswordHasher()}
	if err := seeder.Run(ctx, testSrv.DB); err != nil {
		panic(fmt.Sprintf("mustSetupPlatformAdmin: seeder: %v", err))
	}

	w := testSrv.POST("/api/v1/auth/login", map[string]any{
		"email": email, "password": testPassword, "solution": testConsoleSolution,
	})
	if w.Code != 200 {
		panic(fmt.Sprintf("mustSetupPlatformAdmin: login gagal %d — %s", w.Code, w.Body.String()))
	}
	return decodeTokens(w.Body.Bytes()).AccessToken
}
