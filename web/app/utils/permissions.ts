import type { PermissionGroupRef } from '~/types/api'

/** Bentuk minimal permission yang bisa dikelompokkan. */
export interface GroupablePermission {
  code: string
  name: string
  group: PermissionGroupRef | null
}

export interface PermissionGroupView<T> {
  key: string
  label: string
  /** true = grup didefinisikan solution di katalog; false = hasil fallback prefix code. */
  defined: boolean
  items: T[]
}

const OTHER = 'lainnya'

/**
 * Kelompokkan permission: grup katalog (urut sort_order) lebih dulu; permission tanpa grup
 * dikelompokkan berdasarkan prefix code sebelum "." (perilaku lama), dan "Lainnya" terakhir.
 * Item di tiap grup diurutkan berdasarkan nama.
 */
export function groupPermissions<T extends GroupablePermission>(items: T[]): PermissionGroupView<T>[] {
  const defined = new Map<string, { order: number; view: PermissionGroupView<T> }>()
  const fallback = new Map<string, PermissionGroupView<T>>()
  for (const p of items) {
    if (p.group) {
      const entry = defined.get(p.group.code)
        ?? { order: p.group.sort_order, view: { key: `g:${p.group.code}`, label: p.group.name, defined: true, items: [] } }
      entry.view.items.push(p)
      defined.set(p.group.code, entry)
      continue
    }
    const prefix = /^[^.:]+[.:]/.test(p.code) ? p.code.split(/[.:]/)[0]! : OTHER
    const view = fallback.get(prefix)
      ?? { key: `p:${prefix}`, label: prefix.charAt(0).toUpperCase() + prefix.slice(1), defined: false, items: [] }
    view.items.push(p)
    fallback.set(prefix, view)
  }
  const byName = (a: T, b: T) => a.name.localeCompare(b.name)
  const first = [...defined.values()].sort((a, b) => a.order - b.order || a.view.label.localeCompare(b.view.label)).map(e => e.view)
  const rest = [...fallback.entries()]
    .sort(([a], [b]) => (a === OTHER ? 1 : b === OTHER ? -1 : a.localeCompare(b)))
    .map(([, v]) => v)
  return [...first, ...rest].map(g => ({ ...g, items: [...g.items].sort(byName) }))
}

const SHOW_CODE_KEY = 'permissions.showCode'

/** Preferensi per-viewer "tampilkan code permission" (disimpan di localStorage bila tersedia). */
export function useShowPermissionCode() {
  const show = useState<boolean>(SHOW_CODE_KEY, () => {
    if (!import.meta.client) return false
    try {
      return localStorage.getItem(SHOW_CODE_KEY) === '1'
    } catch {
      return false
    }
  })
  watch(show, (v) => {
    try {
      localStorage.setItem(SHOW_CODE_KEY, v ? '1' : '0')
    } catch {
      // storage tidak tersedia (private mode / diblokir) — preferensi hanya berlaku di sesi ini
    }
  })
  return show
}
