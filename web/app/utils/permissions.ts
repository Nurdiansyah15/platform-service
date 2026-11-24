import type { PermissionGroupRef, PermissionGroupResponse } from '~/types/api'

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
  /** Tingkat di pohon (0 = root). */
  depth: number
  /** Permission yang langsung berada di grup ini (bukan di subgrup). */
  items: T[]
  children: PermissionGroupView<T>[]
}

const OTHER = 'lainnya'

interface Node<T> { view: PermissionGroupView<T>; order: number }

/**
 * Kelompokkan permission menjadi pohon grup katalog (leluhur dari `group.parents`, urut
 * sort_order) lebih dulu; permission tanpa grup dikelompokkan berdasarkan prefix code sebelum
 * "." (perilaku lama), dan "Lainnya" terakhir. Item di tiap grup diurutkan berdasarkan nama.
 */
export function groupPermissions<T extends GroupablePermission>(items: T[]): PermissionGroupView<T>[] {
  const roots = new Map<string, Node<T>>()
  const childrenOf = new Map<PermissionGroupView<T>, Map<string, Node<T>>>()
  const fallback = new Map<string, PermissionGroupView<T>>()

  function nodeIn(level: Map<string, Node<T>>, g: { code: string; name: string; sort_order: number }, depth: number) {
    let n = level.get(g.code)
    if (!n) {
      n = { order: g.sort_order, view: { key: `g:${g.code}`, label: g.name, defined: true, depth, items: [], children: [] } }
      level.set(g.code, n)
    }
    return n.view
  }

  for (const p of items) {
    if (p.group) {
      const chain = [...(p.group.parents ?? []), p.group]
      let level = roots
      let view: PermissionGroupView<T> | undefined
      chain.forEach((g, depth) => {
        view = nodeIn(level, g, depth)
        level = childrenOf.get(view) ?? new Map()
        childrenOf.set(view, level)
      })
      view!.items.push(p)
      continue
    }
    const prefix = /^[^.:]+[.:]/.test(p.code) ? p.code.split(/[.:]/)[0]! : OTHER
    const view = fallback.get(prefix)
      ?? { key: `p:${prefix}`, label: prefix.charAt(0).toUpperCase() + prefix.slice(1), defined: false, depth: 0, items: [], children: [] }
    view.items.push(p)
    fallback.set(prefix, view)
  }

  const byName = (a: T, b: T) => a.name.localeCompare(b.name)
  const build = (level: Map<string, Node<T>>): PermissionGroupView<T>[] =>
    [...level.values()]
      .sort((a, b) => a.order - b.order || a.view.label.localeCompare(b.view.label))
      .map(({ view }) => ({ ...view, items: [...view.items].sort(byName), children: build(childrenOf.get(view) ?? new Map()) }))

  const rest = [...fallback.entries()]
    .sort(([a], [b]) => (a === OTHER ? 1 : b === OTHER ? -1 : a.localeCompare(b)))
    .map(([, v]) => ({ ...v, items: [...v.items].sort(byName) }))
  return [...build(roots), ...rest]
}

/** Seluruh permission di grup beserta subgrupnya. */
export function groupAllItems<T>(g: PermissionGroupView<T>): T[] {
  return [...g.items, ...g.children.flatMap(groupAllItems)]
}

/**
 * Ratakan pohon menjadi daftar section (hanya grup yang punya permission langsung) dengan
 * label berjenjang, mis. "Kepegawaian › Data Karyawan".
 */
export function flattenPermissionGroups<T>(tree: PermissionGroupView<T>[], prefix: string[] = []): { key: string; label: string; items: T[] }[] {
  return tree.flatMap((g) => {
    const path = [...prefix, g.label]
    const self = g.items.length ? [{ key: g.key, label: path.join(' › '), items: g.items }] : []
    return [...self, ...flattenPermissionGroups(g.children, path)]
  })
}

/** Kedalaman maksimal hierarki grup (sama dengan server). */
export const GROUP_MAX_DEPTH = 3

export interface OrderedGroup {
  group: PermissionGroupResponse
  /** 0 = root. */
  depth: number
  /** Nama berjenjang, mis. "Kepegawaian › Data Karyawan". */
  path: string
}

/**
 * Urutkan daftar grup (respons GET permission-groups) mengikuti pohon: induk lalu anaknya
 * (urut sort_order, lalu nama). Grup yang induknya tidak ada di daftar dianggap root.
 */
export function orderGroupTree(groups: PermissionGroupResponse[]): OrderedGroup[] {
  const ids = new Set(groups.map(g => g.id))
  const children = new Map<string | null, PermissionGroupResponse[]>()
  for (const g of groups) {
    const parent = g.parent_id && ids.has(g.parent_id) ? g.parent_id : null
    children.set(parent, [...(children.get(parent) ?? []), g])
  }
  const out: OrderedGroup[] = []
  const seen = new Set<string>()
  const walk = (parent: string | null, depth: number, prefix: string[]) => {
    const list = [...(children.get(parent) ?? [])].sort((a, b) => a.sort_order - b.sort_order || a.name.localeCompare(b.name))
    for (const g of list) {
      if (seen.has(g.id)) continue
      seen.add(g.id)
      const path = [...prefix, g.name]
      out.push({ group: g, depth, path: path.join(' › ') })
      walk(g.id, depth + 1, path)
    }
  }
  walk(null, 0, [])
  return out
}

/** ID grup id beserta seluruh turunannya. */
export function groupDescendantIds(groups: PermissionGroupResponse[], id: string): Set<string> {
  const out = new Set([id])
  let grew = true
  while (grew) {
    grew = false
    for (const g of groups) {
      if (g.parent_id && out.has(g.parent_id) && !out.has(g.id)) {
        out.add(g.id)
        grew = true
      }
    }
  }
  return out
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
