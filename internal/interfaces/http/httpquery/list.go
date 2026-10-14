// Package httpquery berisi helper parsing query string yang dipakai bersama
// semua handler.
package httpquery

import (
	"strings"

	"github.com/gin-gonic/gin"
)

// List membaca query param bertipe array. Menerima bentuk repeatable
// (`?k=a&k=b`), bracket (`?k[]=a&k[]=b`), dan comma-separated (`?k=a,b`),
// boleh dicampur. Nilai kosong dibuang, duplikat dipertahankan urutan pertama.
func List(c *gin.Context, key string) []string {
	raw := append(c.QueryArray(key), c.QueryArray(key+"[]")...)
	seen := make(map[string]struct{}, len(raw))
	var out []string
	for _, v := range raw {
		for _, part := range strings.Split(v, ",") {
			part = strings.TrimSpace(part)
			if part == "" {
				continue
			}
			if _, ok := seen[part]; ok {
				continue
			}
			seen[part] = struct{}{}
			out = append(out, part)
		}
	}
	return out
}
