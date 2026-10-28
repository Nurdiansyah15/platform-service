export default defineNuxtConfig({
  compatibilityDate: '2025-07-15',
  ssr: false,
  devtools: { enabled: true },
  modules: ['@nuxt/ui', '@pinia/nuxt'],
  css: ['~/assets/css/main.css'],
  runtimeConfig: {
    public: {
      apiBase: 'http://localhost:8080',
      defaultSolution: 'platform-console',
    },
  },
  app: {
    head: {
      title: 'Platform Console',
      meta: [{ name: 'viewport', content: 'width=device-width, initial-scale=1' }],
    },
  },
  devServer: { host: '0.0.0.0' },
  vite: {
    server: {
      watch: {
        usePolling: process.env.NUXT_WATCH_POLLING === 'true',
      },
    },
  },
})