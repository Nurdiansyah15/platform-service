import type { EffectivePermission } from '~/types/api'

/** Permission platform untuk administrasi company (katalog solution console). */
export type PlatformPermission = 'company.manage' | 'member.manage' | 'branch.manage' | 'role.manage' | 'role.assign'

export const PLATFORM_PERMISSIONS: PlatformPermission[] = ['company.manage', 'member.manage', 'branch.manage', 'role.manage', 'role.assign']

/**
 * Permission platform user di company yang sedang dibuka, untuk menyembunyikan aksi tulis
 * yang pasti ditolak API. Platform admin selalu boleh. Dimuat dari effective-permissions
 * solution console (solution sesi console) dan di-cache per company.
 */
export function useCompanyPermissions() {
  const auth = useAuthStore()
  const api = useApi()
  const ctx = useCompanyContext()
  const cache = useState<Record<string, EffectivePermission[]>>('company-permissions', () => ({}))
  const loaded = computed(() => auth.isPlatformAdmin || !!cache.value[ctx.companyId.value])

  async function load(force = false) {
    const cid = ctx.companyId.value
    const userId = auth.user?.id
    const solutionId = auth.solution?.id
    if (auth.isPlatformAdmin || !cid || !userId || !solutionId || !ctx.canAccess.value) return
    if (!force && cache.value[cid]) return
    try {
      const res = await api.get<EffectivePermission[]>(`/api/v1/companies/${cid}/users/${userId}/effective-permissions`, {
        query: { solution_id: solutionId },
      })
      cache.value = { ...cache.value, [cid]: res.data }
    } catch {
      cache.value = { ...cache.value, [cid]: [] }
    }
  }

  /** Permission di level company (berlaku juga untuk semua cabang). */
  function can(permission: PlatformPermission) {
    if (auth.isPlatformAdmin) return true
    const p = cache.value[ctx.companyId.value]?.find(e => e.permission === permission)
    return !!p?.grants.some(g => g.scope_type === 'company')
  }

  /** Permission di cabang tertentu (assignment level company atau cabang itu). */
  function canInBranch(permission: PlatformPermission, branchId: string) {
    if (can(permission)) return true
    const p = cache.value[ctx.companyId.value]?.find(e => e.permission === permission)
    return !!p?.grants.some(g => g.scope_type === 'branch' && g.scope_id === branchId)
  }

  /** Punya permission di scope mana pun (mis. untuk menampilkan menu). */
  function canAny(permission: PlatformPermission) {
    if (auth.isPlatformAdmin) return true
    return !!cache.value[ctx.companyId.value]?.some(e => e.permission === permission && e.grants.length)
  }

  /**
   * Pemegang seluruh permission platform di level company (mis. role Admin company). Hanya
   * mereka yang boleh mengelola / memberi role milik solution console (anti eskalasi).
   */
  const isCompanyAdmin = computed(() => PLATFORM_PERMISSIONS.every(p => can(p)))

  /** Tidak punya permission platform sama sekali → company hanya bisa dilihat. */
  const readOnly = computed(() => loaded.value && !PLATFORM_PERMISSIONS.some(p => canAny(p)))

  /** Role milik solution console berisi permission platform. */
  function isConsoleSolution(solutionId?: string | null) {
    return !!solutionId && solutionId === auth.solution?.id
  }

  watch([() => ctx.companyId.value, () => auth.sessionId], () => load(), { immediate: true })

  return { loaded, load, can, canInBranch, canAny, isCompanyAdmin, readOnly, isConsoleSolution }
}
