import type { CompanyResponse } from '~/types/api'

/**
 * Konteks company dari route `/companies/[companyId]/**`. Platform admin bisa membuka
 * company mana pun; user lain hanya company aktif di sesinya (cid).
 */
export function useCompanyContext() {
  const route = useRoute()
  const auth = useAuthStore()
  const api = useApi()
  const cache = useState<Record<string, CompanyResponse>>('company-cache', () => ({}))

  const companyId = computed(() => String(route.params.companyId ?? ''))
  const company = computed(() => cache.value[companyId.value] ?? null)
  const canAccess = computed(() => auth.isPlatformAdmin || auth.activeCompanyId === companyId.value)
  /** User member company ini (bisa diaktifkan lewat switch). */
  const isMemberOf = computed(() => auth.companies.some(c => c.id === companyId.value))
  const error = ref('')

  async function load(force = false) {
    const id = companyId.value
    if (!id || !canAccess.value || (!force && cache.value[id])) return
    error.value = ''
    try {
      const res = await api.get<CompanyResponse>(`/api/v1/companies/${id}`)
      cache.value = { ...cache.value, [id]: res.data }
    } catch (err) {
      error.value = (err as { message?: string }).message || 'Gagal memuat company'
    }
  }

  function setCompany(c: CompanyResponse) {
    cache.value = { ...cache.value, [c.id]: c }
  }

  return { companyId, company, canAccess, isMemberOf, error, load, setCompany }
}
