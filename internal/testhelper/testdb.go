package testhelper

import (
	"context"
	"database/sql"
	"fmt"
	"log"
	"time"

	_ "github.com/jackc/pgx/v5/stdlib"
	"github.com/testcontainers/testcontainers-go"
	"github.com/testcontainers/testcontainers-go/wait"

	"platform-service/internal/migrations"
	"platform-service/internal/migrator"
)

const (
	testDBUser     = "test"
	testDBPassword = "test"
	testDBName     = "testdb"
)

// MustStartTestDB menjalankan container postgres:16-alpine, menjalankan semua
// migrasi, dan mengembalikan *sql.DB siap pakai. Panggil sekali dari TestMain.
func MustStartTestDB() (*sql.DB, func()) {
	ctx := context.Background()

	container, err := testcontainers.GenericContainer(ctx, testcontainers.GenericContainerRequest{
		ContainerRequest: testcontainers.ContainerRequest{
			Image: "postgres:16-alpine",
			Env: map[string]string{
				"POSTGRES_USER":     testDBUser,
				"POSTGRES_PASSWORD": testDBPassword,
				"POSTGRES_DB":       testDBName,
			},
			ExposedPorts: []string{"5432/tcp"},
			WaitingFor: wait.ForLog("database system is ready to accept connections").
				WithOccurrence(2).
				WithStartupTimeout(60 * time.Second),
		},
		Started: true,
	})
	if err != nil {
		log.Fatalf("testhelper: start postgres container: %v", err)
	}

	host, err := container.Host(ctx)
	if err != nil {
		log.Fatalf("testhelper: get container host: %v", err)
	}
	port, err := container.MappedPort(ctx, "5432")
	if err != nil {
		log.Fatalf("testhelper: get mapped port: %v", err)
	}

	hostPort := fmt.Sprintf("%s:%s", host, port.Port())
	db := mustOpenAndMigrate(ctx, hostPort)

	cleanup := func() {
		_ = db.Close()
		if err := container.Terminate(ctx); err != nil {
			log.Printf("testhelper: terminate container: %v", err)
		}
	}
	return db, cleanup
}

func mustOpenAndMigrate(ctx context.Context, hostPort string) *sql.DB {
	dsn := fmt.Sprintf("postgres://%s:%s@%s/%s?sslmode=disable", testDBUser, testDBPassword, hostPort, testDBName)

	db, err := sql.Open("pgx", dsn)
	if err != nil {
		log.Fatalf("testhelper: open db: %v", err)
	}
	db.SetMaxOpenConns(10)
	db.SetConnMaxLifetime(5 * time.Minute)

	pingCtx, cancel := context.WithTimeout(ctx, 30*time.Second)
	defer cancel()
	for {
		if err := db.PingContext(pingCtx); err == nil {
			break
		}
		select {
		case <-pingCtx.Done():
			log.Fatalf("testhelper: db never became ready: %v", pingCtx.Err())
		case <-time.After(200 * time.Millisecond):
		}
	}

	m, err := migrator.New(db, migrations.FS)
	if err != nil {
		log.Fatalf("testhelper: load migrations: %v", err)
	}
	if _, err := m.Up(ctx); err != nil {
		log.Fatalf("testhelper: run migrations: %v", err)
	}
	return db
}
