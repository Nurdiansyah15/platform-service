package solution

import (
	"context"
	"log/slog"
	"sync"
	"time"

	"platform-service/internal/domain/solution/entity"
	solutionrepo "platform-service/internal/domain/solution/repository"
)

// DefaultOriginCacheTTL — perubahan solution dari instance lain berlaku paling lambat selama ini.
const DefaultOriginCacheTTL = 30 * time.Second

// OriginAllowlist mengimplementasikan port.OriginAllowlist: origin URL aplikasi & redirect URI
// semua solution active, di-cache in-memory selama TTL. Instance yang sama langsung
// memuat ulang setelah Invalidate (dipanggil SolutionUseCase saat solution berubah).
type OriginAllowlist struct {
	repo   solutionrepo.SolutionRepository
	ttl    time.Duration
	logger *slog.Logger

	mu        sync.RWMutex
	origins   map[string]struct{}
	expiresAt time.Time
	gen       uint64 // naik setiap Invalidate; hasil muat yang dimulai sebelumnya tidak dianggap segar

	loadMu sync.Mutex // satu query ke DB sekaligus
}

func NewOriginAllowlist(repo solutionrepo.SolutionRepository, ttl time.Duration, logger *slog.Logger) *OriginAllowlist {
	if logger == nil {
		logger = slog.Default()
	}
	return &OriginAllowlist{repo: repo, ttl: ttl, logger: logger}
}

func (a *OriginAllowlist) IsAllowedOrigin(ctx context.Context, origin string) bool {
	o, ok := entity.OriginOf(origin)
	if !ok {
		return false
	}
	_, ok = a.current(ctx)[o]
	return ok
}

// Invalidate memaksa daftar dimuat ulang pada pengecekan berikutnya.
func (a *OriginAllowlist) Invalidate() {
	a.mu.Lock()
	a.gen++
	a.expiresAt = time.Time{}
	a.mu.Unlock()
}

func (a *OriginAllowlist) current(ctx context.Context) map[string]struct{} {
	if set, fresh := a.snapshot(); fresh {
		return set
	}
	a.loadMu.Lock()
	defer a.loadMu.Unlock()
	if set, fresh := a.snapshot(); fresh {
		return set
	}

	a.mu.RLock()
	gen := a.gen
	a.mu.RUnlock()

	solutions, err := a.repo.ListActive(ctx)
	a.mu.Lock()
	defer a.mu.Unlock()
	if err != nil {
		// Pakai daftar terakhir (kosong bila belum pernah termuat); coba lagi setelah TTL.
		a.logger.WarnContext(ctx, "cors: gagal memuat origin solution", slog.Any("error", err))
		if a.origins == nil {
			a.origins = map[string]struct{}{}
		}
		if a.gen == gen {
			a.expiresAt = time.Now().Add(a.ttl)
		}
		return a.origins
	}
	set := make(map[string]struct{})
	for _, s := range solutions {
		for _, o := range s.Origins() {
			set[o] = struct{}{}
		}
	}
	a.origins = set
	if a.gen == gen {
		a.expiresAt = time.Now().Add(a.ttl)
	}
	return set
}

func (a *OriginAllowlist) snapshot() (map[string]struct{}, bool) {
	a.mu.RLock()
	defer a.mu.RUnlock()
	return a.origins, a.origins != nil && time.Now().Before(a.expiresAt)
}
