<script setup lang="ts">
import type { SolutionResponse } from '~/types/api'

const route = useRoute()
const api = useApi()
const solutionId = computed(() => String(route.params.solutionId))

const solution = ref<SolutionResponse | null>(null)
const error = ref('')

async function load() {
  error.value = ''
  try {
    const res = await api.get<SolutionResponse>(`/api/v1/solutions/${solutionId.value}`)
    solution.value = res.data
  } catch (err) {
    error.value = (err as { message?: string }).message || 'Gagal memuat solution'
  }
}

const links = computed(() => [
  { to: `/solutions/${solutionId.value}/roles`, label: 'Global Roles', icon: 'i-lucide-shield', desc: 'Role global milik solution.' },
  { to: `/solutions/${solutionId.value}/permissions`, label: 'Permissions', icon: 'i-lucide-key-round', desc: 'Katalog permission solution.' },
])

onMounted(load)
</script>

<template>
  <div>
    <PageHeader :title="solution?.name ?? 'Solution'" :subtitle="solution?.code">
      <template #actions>
        <UButton to="/solutions" color="neutral" variant="outline" icon="i-lucide-arrow-left">Kembali</UButton>
      </template>
    </PageHeader>

    <UAlert v-if="error" color="error" variant="subtle" :title="error" class="mb-4" />

    <UCard v-if="solution" class="mb-6">
      <div class="grid gap-4 sm:grid-cols-2">
        <div>
          <p class="text-xs uppercase tracking-wide text-dimmed">Deskripsi</p>
          <p class="mt-1 text-sm text-muted">{{ solution.description || '-' }}</p>
        </div>
        <div class="grid grid-cols-2 gap-4">
          <div>
            <p class="text-xs uppercase tracking-wide text-dimmed">Status</p>
            <p class="mt-1">
              <UBadge size="sm" variant="subtle" :color="solution.status === 'active' ? 'success' : 'neutral'">
                {{ solution.status }}
              </UBadge>
            </p>
          </div>
          <div>
            <p class="text-xs uppercase tracking-wide text-dimmed">Katalog versi</p>
            <p class="mt-1 text-sm text-muted">{{ solution.permission_catalog_version || '-' }}</p>
          </div>
        </div>
      </div>
    </UCard>

    <div class="grid gap-4 sm:grid-cols-2">
      <NuxtLink
        v-for="link in links"
        :key="link.to"
        :to="link.to"
        class="rounded-lg border border-default bg-default p-4 transition hover:border-primary/40"
      >
        <div class="flex items-center gap-3">
          <UIcon :name="link.icon" class="size-5 text-primary" />
          <span class="font-medium text-highlighted">{{ link.label }}</span>
        </div>
        <p class="mt-2 text-sm text-muted">{{ link.desc }}</p>
      </NuxtLink>
    </div>
  </div>
</template>