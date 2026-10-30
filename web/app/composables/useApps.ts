import type { AppResponse } from '~/types/api'

/**
 * Daftar solution yang bisa dibuka (GET /auth/apps) untuk company aktif. Dipakai launcher
 * dan menu apps; dimuat ulang saat company aktif berganti.
 */
export function useApps() {
  const auth = useAuthStore()
  const api = useApi()
  const apps = useState<AppResponse[]>('apps', () => [])
  const loadedFor = useState<string | null>('apps-loaded-for', () => null)
  const loading = ref(false)

  async function load(force = false) {
    const key = `${auth.sessionId}:${auth.activeCompanyId ?? ''}`
    if (!auth.isAuthenticated || (!force && loadedFor.value === key)) return
    loading.value = true
    try {
      apps.value = (await api.get<AppResponse[]>('/api/v1/auth/apps')).data
      loadedFor.value = key
    } catch {
      apps.value = []
    } finally {
      loading.value = false
    }
  }

  /**
   * URL untuk membuka solution. Company aktif ikut dikirim agar solution meminta sesi
   * untuk company yang sama lewat alur SSO (/authorize).
   */
  function launchHref(app: AppResponse) {
    try {
      const u = new URL(app.launch_url)
      if (auth.activeCompanyId) u.searchParams.set('company_id', auth.activeCompanyId)
      return u.toString()
    } catch {
      return app.launch_url
    }
  }

  watch(() => auth.activeCompanyId, () => load(), { immediate: true })

  return { apps, loading, load, launchHref }
}
