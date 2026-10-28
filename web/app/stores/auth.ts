import type { SelectableCompany, AuthTokenResponse, MeResponse, UserResponse } from '~/types/api'

const REFRESH_KEY = 'pc.refresh'

export const useAuthStore = defineStore('auth', () => {
  const accessToken = ref<string | null>(null)
  const refreshToken = ref<string | null>(null)
  const sessionId = ref<string | null>(null)
  const me = ref<MeResponse | null>(null)
  const companies = ref<SelectableCompany[]>([])
  const ready = ref(false)

  const isAuthenticated = computed(() => !!accessToken.value && !!me.value)
  const user = computed<UserResponse | null>(() => me.value?.user ?? null)
  const isPlatformAdmin = computed(() => me.value?.user.is_platform_admin ?? false)
  const activeCompanyId = computed(() => me.value?.session.company_id ?? null)
  const activeCompany = computed(
    () => companies.value.find(c => c.id === activeCompanyId.value) ?? null,
  )
  const solution = computed(() => me.value?.session.solution ?? null)
  const hasCompany = computed(() => !!activeCompanyId.value)

  function applyTokens(t: AuthTokenResponse) {
    accessToken.value = t.access_token
    refreshToken.value = t.refresh_token
    sessionId.value = t.session_id
    if (import.meta.client) localStorage.setItem(REFRESH_KEY, t.refresh_token)
  }

  function clear() {
    accessToken.value = null
    refreshToken.value = null
    sessionId.value = null
    me.value = null
    companies.value = []
    if (import.meta.client) localStorage.removeItem(REFRESH_KEY)
  }

  async function login(email: string, password: string, solutionCode: string) {
    const res = await apiFetch<AuthTokenResponse>('/api/v1/auth/login', {
      method: 'POST',
      body: { email, password, solution: solutionCode },
    })
    applyTokens(res.data)
    await loadMe()
    return res.data
  }

  async function tryRefresh(): Promise<boolean> {
    const token = refreshToken.value ?? (import.meta.client ? localStorage.getItem(REFRESH_KEY) : null)
    if (!token) return false
    try {
      const res = await apiFetch<AuthTokenResponse>('/api/v1/auth/refresh', {
        method: 'POST',
        body: { refresh_token: token },
      })
      applyTokens(res.data)
      return true
    } catch {
      clear()
      return false
    }
  }

  async function bootstrap() {
    if (import.meta.client) refreshToken.value = localStorage.getItem(REFRESH_KEY)
    if (refreshToken.value && (await tryRefresh())) {
      await loadMe()
      await loadCompanies()
    }
    ready.value = true
  }

  async function loadMe() {
    const res = await apiFetch<MeResponse>('/api/v1/auth/me', { token: accessToken.value })
    me.value = res.data
    return res.data
  }

  async function loadCompanies() {
    const res = await apiFetch<SelectableCompany[]>('/api/v1/auth/companies', {
      token: accessToken.value,
    })
    companies.value = res.data
    return res.data
  }

  async function switchCompany(companyId: string) {
    const res = await apiFetch<AuthTokenResponse>('/api/v1/auth/switch-company', {
      method: 'POST',
      body: { company_id: companyId },
      token: accessToken.value,
    })
    applyTokens(res.data)
    await loadMe()
    return res.data
  }

  async function logout() {
    try {
      await apiFetch('/api/v1/auth/logout', { method: 'POST', token: accessToken.value })
    } catch {
      // abaikan: sesi lokal tetap dibersihkan
    }
    clear()
  }

  return {
    accessToken,
    refreshToken,
    sessionId,
    me,
    companies,
    ready,
    isAuthenticated,
    user,
    isPlatformAdmin,
    activeCompanyId,
    activeCompany,
    solution,
    hasCompany,
    login,
    tryRefresh,
    bootstrap,
    loadMe,
    loadCompanies,
    switchCompany,
    logout,
    clear,
  }
})
