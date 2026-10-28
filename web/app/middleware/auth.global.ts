const PUBLIC_ROUTES = ['/login', '/request-access']

export default defineNuxtRouteMiddleware(async (to) => {
  const auth = useAuthStore()

  if (!auth.ready) {
    await auth.bootstrap()
  }

  const isPublic = PUBLIC_ROUTES.includes(to.path)

  if (isPublic) {
    if (auth.isAuthenticated) return navigateTo('/')
    return
  }

  if (!auth.isAuthenticated) {
    return navigateTo({ path: '/login', query: to.fullPath !== '/' ? { redirect: to.fullPath } : {} })
  }

  if (!auth.me) {
    await auth.loadMe()
  }

  if (!auth.hasCompany && !auth.isPlatformAdmin && to.path !== '/onboarding') {
    return navigateTo('/onboarding')
  }
})
