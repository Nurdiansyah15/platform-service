const PUBLIC_ROUTES = ['/login']

/** Route lama → route baru (link lama tetap berfungsi). */
function legacyRedirect(path: string, query: Record<string, unknown>, isAdmin: boolean, companyId: string | null) {
  if (path === '/request-access') return '/login'
  if (path === '/authorization/check') return '/'
  if (path === '/companies') return isAdmin ? '/admin/companies' : companyId ? `/companies/${companyId}` : '/onboarding'
  if (path.startsWith('/solutions')) return `/admin${path}`
  const m = path.match(/^\/roles\/([^/]+)\/permissions$/)
  if (m) {
    if (typeof query.companyId === 'string' && query.companyId) return `/companies/${query.companyId}/roles/${m[1]}`
    if (typeof query.solutionId === 'string' && query.solutionId) return `/admin/solutions/${query.solutionId}/roles/${m[1]}`
    return '/'
  }
  return null
}

export default defineNuxtRouteMiddleware(async (to) => {
  const auth = useAuthStore()

  if (!auth.ready) {
    await auth.bootstrap()
  }

  if (PUBLIC_ROUTES.includes(to.path)) {
    if (auth.isAuthenticated) return navigateTo('/')
    return
  }

  if (!auth.isAuthenticated) {
    if (to.path === '/request-access') return navigateTo('/login')
    return navigateTo({ path: '/login', query: to.fullPath !== '/' ? { redirect: to.fullPath } : {} })
  }

  if (!auth.me) {
    await auth.loadMe()
  }

  const legacy = legacyRedirect(to.path, to.query, auth.isPlatformAdmin, auth.activeCompanyId)
  if (legacy && legacy !== to.path) return navigateTo(legacy)

  // User biasa tanpa company aktif hanya bisa ke onboarding & profil.
  if (!auth.hasCompany && !auth.isPlatformAdmin && !['/onboarding', '/profile'].includes(to.path)) {
    return navigateTo('/onboarding')
  }
})
