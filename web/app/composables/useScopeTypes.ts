import type { ScopeTypeResponse } from '~/types/api'

/**
 * Scope type solution (company & branch built-in + scope custom dari katalog), di-cache per
 * solution selama sesi. `nameOf` memberi nama tampilan untuk code scope type.
 */
export function useScopeTypes(solutionId: MaybeRefOrGetter<string | null | undefined>) {
  const api = useApi()
  const cache = useState<Record<string, ScopeTypeResponse[]>>('scope-types', () => ({}))
  const loading = ref(false)

  const items = computed<ScopeTypeResponse[]>(() => {
    const id = toValue(solutionId)
    return id ? cache.value[id] ?? [] : []
  })

  async function load(force = false) {
    const id = toValue(solutionId)
    if (!id || (!force && cache.value[id])) return
    loading.value = true
    try {
      const res = await api.get<ScopeTypeResponse[]>(`/api/v1/solutions/${id}/scope-types`)
      cache.value = { ...cache.value, [id]: res.data }
    } catch {
      // nama scope jatuh ke code; halaman tetap bisa dipakai
    } finally {
      loading.value = false
    }
  }

  watch(() => toValue(solutionId), () => load(), { immediate: true })

  const BUILTIN: Record<string, string> = { company: 'Company', branch: 'Cabang' }
  function nameOf(code: string) {
    return BUILTIN[code] ?? items.value.find(s => s.code === code)?.name ?? code
  }

  return { items, loading, load, nameOf }
}
