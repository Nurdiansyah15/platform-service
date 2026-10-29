import type { Meta } from '~/types/api'

type FilterValue = string | number | boolean | undefined

interface ListQueryOptions {
  /** Filter tambahan (mis. status); perubahan mereset ke halaman 1. */
  filters?: Record<string, Ref<FilterValue>>
  limit?: number
  /** Simpan page/search/filter di query URL agar bisa di-refresh & dibagikan. */
  syncUrl?: boolean
}

/**
 * State daftar ber-pagination: items, meta, loading, error, page, search (debounce),
 * dan filter. `path` null = belum siap (tidak memuat).
 */
export function useListQuery<T>(path: MaybeRefOrGetter<string | null>, opts: ListQueryOptions = {}) {
  const api = useApi()
  const route = useRoute()
  const router = useRouter()
  const filters = opts.filters ?? {}
  const limit = opts.limit ?? 20

  const fromQuery = (key: string) => (typeof route.query[key] === 'string' ? (route.query[key] as string) : undefined)
  if (opts.syncUrl) {
    for (const [key, r] of Object.entries(filters)) {
      const v = fromQuery(key)
      if (v !== undefined) r.value = v === 'true' ? true : v === 'false' ? false : v
    }
  }

  const items = ref<T[]>([]) as Ref<T[]>
  const meta = ref<Meta | null>(null)
  const loading = ref(false)
  const error = ref('')
  const page = ref(opts.syncUrl ? Number(fromQuery('page') ?? 1) || 1 : 1)
  const search = ref(opts.syncUrl ? (fromQuery('q') ?? '') : '')

  let seq = 0
  async function load() {
    const p = toValue(path)
    if (!p) return
    const mySeq = ++seq
    loading.value = true
    error.value = ''
    const query: Record<string, unknown> = { page: page.value, limit, search: search.value.trim() || undefined }
    for (const [key, r] of Object.entries(filters)) query[key] = r.value
    try {
      const res = await api.get<T[]>(p, { query })
      if (mySeq !== seq) return // respons lama (filter sudah berubah)
      items.value = res.data
      meta.value = res.meta ?? null
    } catch (err) {
      if (mySeq !== seq) return
      error.value = (err as { message?: string }).message || 'Gagal memuat data'
    } finally {
      if (mySeq === seq) loading.value = false
    }
    if (opts.syncUrl) {
      const q: Record<string, string | undefined> = { ...(route.query as Record<string, string>) }
      q.page = page.value > 1 ? String(page.value) : undefined
      q.q = search.value.trim() || undefined
      for (const [key, r] of Object.entries(filters)) q[key] = r.value === undefined ? undefined : String(r.value)
      router.replace({ query: q })
    }
  }

  function resetAndLoad() {
    page.value = 1
    load()
  }

  let debounce: ReturnType<typeof setTimeout> | undefined
  watch(search, () => {
    clearTimeout(debounce)
    debounce = setTimeout(resetAndLoad, 300)
  })
  watch(Object.values(filters), resetAndLoad)
  watch(page, load)
  watch(() => toValue(path), resetAndLoad)
  onMounted(load)

  return { items, meta, loading, error, page, search, reload: load }
}
