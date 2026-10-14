// Command seeder menjalankan seeder di internal/seeders.
// Pemakaian: seeder            → jalankan semua
//
//	seeder <name> ...   → jalankan seeder tertentu
package main

import (
	"context"
	"log"
	"os"
	"slices"

	"platform-service/internal/config"
	"platform-service/internal/infrastructure/database"
	"platform-service/internal/seeders"
)

func main() {
	cfg, err := config.Load()
	if err != nil {
		log.Fatalf("gagal load config: %v", err)
	}

	ctx := context.Background()
	db, err := database.Open(ctx, cfg.Database)
	if err != nil {
		log.Fatalf("gagal koneksi database: %v", err)
	}
	defer db.Close()

	only := os.Args[1:]
	ran := 0
	seedCfg := seeders.Config{
		PlatformAdminEmail:    os.Getenv("PLATFORM_ADMIN_EMAIL"),
		PlatformAdminName:     os.Getenv("PLATFORM_ADMIN_NAME"),
		PlatformAdminPassword: os.Getenv("PLATFORM_ADMIN_PASSWORD"),
	}
	for _, s := range seeders.All(seedCfg) {
		if len(only) > 0 && !slices.Contains(only, s.Name()) {
			continue
		}
		log.Printf("seeding %s...", s.Name())
		if err := s.Run(ctx, db); err != nil {
			log.Fatalf("seeder %s gagal: %v", s.Name(), err)
		}
		ran++
	}
	log.Printf("selesai: %d seeder dijalankan", ran)
}
