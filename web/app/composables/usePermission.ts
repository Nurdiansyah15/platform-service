import type { AuthzCheckBatchResponse } from '~/types/api'

/**
 * Evaluasi permission UI lewat `POST /authz/check-batch`. Platform admin selalu
 * dianggap punya semua akses (backend tidak menegakkan RBAC untuk admin).
 */
export function usePermission() {
  const auth = useAuthStore()
  const api = useApi()
  const cache = useState<Record<string, boolean>>('authz-cache', () => ({}))

  async function check(permissions: string[]): Promise<Record<string, boolean>> {
    const out: Record<string, boolean> = {}
    if (auth.isPlatformAdmin || !permissions.length) {
      for (const p of permissions) out[p] = auth.isPlatformAdmin
      return out
    }
    const companyId = auth.activeCompanyId
    if (!companyId) {
      for (const p of permissions) out[p] = false
      return out
    }
    const missing = permissions.filter(p => cache.value[p] === undefined)
    if (missing.length) {
      try {
        const res = await api.post<AuthzCheckBatchResponse>('/api/v1/authz/check-batch', {
          company_id: companyId,
          checks: missing.map(permission => ({ permission })),
        })
        for (const r of res.data.results) cache.value[r.permission] = r.allowed
      } catch {
        // biarkan sebagai "tidak diketahui" → false
      }
    }
    for (const p of permissions) out[p] = cache.value[p] ?? false
    return out
  }

  function allowed(permission: string): boolean {
    return auth.isPlatformAdmin || cache.value[permission] === true
  }

  function reset() {
    cache.value = {}
  }

  return { check, allowed, reset }
}
