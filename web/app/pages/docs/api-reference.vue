<script setup lang="ts">
import { ApiReference } from '@scalar/api-reference'
import scalarCss from '@scalar/api-reference/style.css?raw'

// Referensi lengkap dari OpenAPI (docs/swagger → public/openapi.json lewat `make swagger`).
definePageMeta({ layout: false })
// CSS Scalar dipasang sebagai <style> milik halaman ini saja (dilepas saat halaman ditinggalkan).
// Wajib `?raw`: import CSS biasa maupun `?url` dimasukkan Nuxt dev ke <link> global di semua
// halaman, dan `@layer scalar-base { body { background-color: var(--scalar-background-1) } }`
// mengalahkan latar body Nuxt UI → body transparan di console (mode gelap jadi putih).
useHead({
  title: 'API Reference · Platform Docs',
  style: [{ key: 'scalar-css', textContent: scalarCss }],
})

const config = useRuntimeConfig()
const colorMode = useColorMode()

// Mode warna mengikuti Nuxt sepenuhnya. Tanpa forceDarkModeState, Scalar membaca
// localStorage.colorMode miliknya sendiri dan bisa berbeda dengan Nuxt (header terang, isi gelap).
// forceDarkModeState hanya dibaca saat mount, jadi komponen di-remount (key) saat mode berubah.
const scalarMode = computed<'dark' | 'light'>(() => (colorMode.value === 'dark' ? 'dark' : 'light'))

const configuration = computed(() => ({
  url: '/openapi.json',
  servers: [{ url: config.public.apiBase as string, description: 'Platform API' }],
  forceDarkModeState: scalarMode.value,
  hideDarkModeToggle: true,
  metaData: { title: 'Platform API Reference' },
  // Tanpa fitur layanan cloud Scalar (AI agent, MCP, toolbar deploy/SDK, telemetry).
  agent: { disabled: true },
  mcp: { disabled: true },
  showDeveloperTools: 'never' as const,
  showToolbar: 'never' as const,
  telemetry: false,
}))

// Scalar memasang class dark-mode / light-mode (beserta background) di <body>. Bersihkan saat
// halaman ditinggalkan agar tidak bocor ke halaman lain lewat navigasi SPA.
onBeforeUnmount(() => {
  document.body.classList.remove('dark-mode', 'light-mode')
})
</script>

<template>
  <div class="min-h-dvh bg-default">
    <header class="sticky top-0 z-40 flex h-12 items-center gap-3 border-b border-default bg-default px-4">
      <UButton to="/docs" color="neutral" variant="ghost" size="sm" icon="i-lucide-arrow-left">
        Panduan integrasi
      </UButton>
      <span class="text-sm font-semibold text-highlighted">API Reference</span>
    </header>
    <ClientOnly>
      <ApiReference :key="scalarMode" :configuration="configuration" />
    </ClientOnly>
  </div>
</template>
