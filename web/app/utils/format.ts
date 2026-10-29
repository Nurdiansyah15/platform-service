const LOCALE = 'id-ID'

/** Tanggal pendek, mis. "29 Okt 2026". */
export function formatDate(value?: string | null): string {
  if (!value) return '—'
  return new Date(value).toLocaleDateString(LOCALE, { day: 'numeric', month: 'short', year: 'numeric' })
}

/** Tanggal + jam, mis. "29 Okt 2026, 09.15". */
export function formatDateTime(value?: string | null): string {
  if (!value) return '—'
  return new Date(value).toLocaleString(LOCALE, {
    day: 'numeric', month: 'short', year: 'numeric', hour: '2-digit', minute: '2-digit',
  })
}

/** Waktu relatif singkat, mis. "3 hari lalu". */
export function formatRelative(value?: string | null): string {
  if (!value) return '—'
  const diff = (new Date(value).getTime() - Date.now()) / 1000
  const rtf = new Intl.RelativeTimeFormat(LOCALE, { numeric: 'auto' })
  const units: [Intl.RelativeTimeFormatUnit, number][] = [
    ['year', 31536000], ['month', 2592000], ['day', 86400], ['hour', 3600], ['minute', 60],
  ]
  for (const [unit, sec] of units) {
    if (Math.abs(diff) >= sec) return rtf.format(Math.round(diff / sec), unit)
  }
  return 'baru saja'
}

/** Slug dari nama: huruf kecil, angka, dan tanda hubung. */
export function slugify(value: string): string {
  return value
    .toLowerCase()
    .normalize('NFKD')
    .replace(/[̀-ͯ]/g, '')
    .replace(/[^a-z0-9]+/g, '-')
    .replace(/^-+|-+$/g, '')
    .slice(0, 100)
}

/** Kode snake_case dari nama (untuk kode role). */
export function codify(value: string): string {
  return slugify(value).replace(/-/g, '_')
}

/** Inisial untuk avatar, mis. "Budi Santoso" → "BS". */
export function initials(name?: string | null): string {
  if (!name) return '?'
  return name.split(/\s+/).filter(Boolean).slice(0, 2).map(w => w[0]!.toUpperCase()).join('')
}
