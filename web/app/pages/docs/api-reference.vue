<script setup lang="ts">
import { ApiReference } from '@scalar/api-reference'
import '@scalar/api-reference/style.css'

// Referensi lengkap dari OpenAPI (docs/swagger → public/openapi.json lewat `make swagger`).
definePageMeta({ layout: false })
useHead({ title: 'API Reference · Platform Docs' })

const config = useRuntimeConfig()
const colorMode = useColorMode()

const configuration = computed(() => ({
  url: '/openapi.json',
  servers: [{ url: config.public.apiBase as string, description: 'Platform API' }],
  darkMode: colorMode.value === 'dark',
  metaData: { title: 'Platform API Reference' },
  // Tanpa fitur layanan cloud Scalar (AI agent, MCP, toolbar deploy/SDK, telemetry).
  agent: { disabled: true },
  mcp: { disabled: true },
  showDeveloperTools: 'never' as const,
  showToolbar: 'never' as const,
  telemetry: false,
}))
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
      <ApiReference :configuration="configuration" />
    </ClientOnly>
  </div>
</template>
