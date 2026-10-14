package i18n

import (
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"sync"
)

var supportedLocales = []string{"en", "id"}
var defaultLocale = "en"

// Bundle menyimpan semua terjemahan yang sudah di-load.
// Thread-safe untuk concurrent read.
type Bundle struct {
	mu      sync.RWMutex
	locales map[string]map[string]string
}

// Load membaca semua file JSON dari localesDir dan me-load ke dalam Bundle.
// Fallback graceful: locale yang gagal di-load dilewati tanpa error fatal.
func Load(localesDir string) (*Bundle, error) {
	b := &Bundle{locales: make(map[string]map[string]string)}
	for _, lang := range supportedLocales {
		path := filepath.Join(localesDir, lang+".json")
		data, err := os.ReadFile(path)
		if err != nil {
			if os.IsNotExist(err) {
				continue
			}
			return nil, fmt.Errorf("i18n: gagal baca file %s: %w", path, err)
		}
		messages := make(map[string]string)
		if err := json.Unmarshal(data, &messages); err != nil {
			return nil, fmt.Errorf("i18n: gagal parse %s: %w", path, err)
		}
		b.locales[lang] = messages
	}
	if len(b.locales) == 0 {
		return nil, fmt.Errorf("i18n: tidak ada file locale ditemukan di %q", localesDir)
	}
	return b, nil
}

// Translate mengembalikan terjemahan untuk key di locale yang diminta.
// Fallback chain: lang → defaultLocale → key itu sendiri.
func (b *Bundle) Translate(lang, key string) string {
	b.mu.RLock()
	defer b.mu.RUnlock()
	if m, ok := b.locales[lang]; ok {
		if msg, ok := m[key]; ok {
			return msg
		}
	}
	if lang != defaultLocale {
		if m, ok := b.locales[defaultLocale]; ok {
			if msg, ok := m[key]; ok {
				return msg
			}
		}
	}
	return key
}

// TranslateF mengembalikan terjemahan dengan format args (untuk validation messages).
func (b *Bundle) TranslateF(lang, key string, args ...any) string {
	tmpl := b.Translate(lang, key)
	if len(args) == 0 {
		return tmpl
	}
	return fmt.Sprintf(tmpl, args...)
}

// IsSupportedLocale memeriksa apakah lang didukung.
func IsSupportedLocale(lang string) bool {
	for _, l := range supportedLocales {
		if l == lang {
			return true
		}
	}
	return false
}

// DefaultLocale mengembalikan locale default.
func DefaultLocale() string {
	return defaultLocale
}
