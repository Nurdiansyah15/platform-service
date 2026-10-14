package middleware

import (
	"strings"

	i18n "platform-service/internal/infrastructure/i18n"

	"github.com/gin-gonic/gin"
)

// Locale mendeteksi bahasa dari request dan menyimpannya ke Gin context.
// Urutan prioritas: ?lang=xx → Accept-Language header → default i18n.DefaultLocale().
func Locale() gin.HandlerFunc {
	return func(c *gin.Context) {
		lang := c.Query("lang")
		if lang == "" {
			lang = parseAcceptLanguage(c.GetHeader("Accept-Language"))
		}
		if !i18n.IsSupportedLocale(lang) {
			lang = i18n.DefaultLocale()
		}
		c.Set("locale", lang)
		c.Next()
	}
}

// parseAcceptLanguage mengambil tag bahasa pertama dari Accept-Language header.
// Contoh: "en-US,en;q=0.9,id;q=0.8" → "en"
func parseAcceptLanguage(header string) string {
	if header == "" {
		return ""
	}
	parts := strings.Split(header, ",")
	if len(parts) == 0 {
		return ""
	}
	first := strings.TrimSpace(parts[0])
	// strip priority weight (e.g. "en;q=0.9" → "en")
	if idx := strings.Index(first, ";"); idx != -1 {
		first = strings.TrimSpace(first[:idx])
	}
	// ambil primary subtag saja (e.g. "en-US" → "en")
	if idx := strings.Index(first, "-"); idx != -1 {
		first = first[:idx]
	}
	return strings.ToLower(first)
}
