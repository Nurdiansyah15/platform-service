// Halaman khusus platform admin; user lain diarahkan ke beranda.
export default defineNuxtRouteMiddleware(() => {
  const auth = useAuthStore()
  if (!auth.isPlatformAdmin) return navigateTo('/')
})
