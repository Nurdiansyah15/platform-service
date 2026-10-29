import type { SolutionResponse } from '~/types/api'

/** Konteks solution dari route `/admin/solutions/[solutionId]/**` (cache per id). */
export function useSolutionContext() {
  const route = useRoute()
  const api = useApi()
  const cache = useState<Record<string, SolutionResponse>>('solution-cache', () => ({}))

  const solutionId = computed(() => String(route.params.solutionId ?? ''))
  const solution = computed(() => cache.value[solutionId.value] ?? null)
  const error = ref('')

  async function load(force = false) {
    const id = solutionId.value
    if (!id || (!force && cache.value[id])) return
    error.value = ''
    try {
      const res = await api.get<SolutionResponse>(`/api/v1/solutions/${id}`)
      cache.value = { ...cache.value, [id]: res.data }
    } catch (err) {
      error.value = (err as { message?: string }).message || 'Gagal memuat solution'
    }
  }

  function setSolution(s: SolutionResponse) {
    cache.value = { ...cache.value, [s.id]: s }
  }

  return { solutionId, solution, error, load, setSolution }
}
