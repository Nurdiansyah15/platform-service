<script setup lang="ts">
import type { PermissionResponse, Meta } from '~/types/api'

const route = useRoute()
const api = useApi()
const solutionId = computed(() => String(route.params.solutionId))

const items = ref<PermissionResponse[]>([])
const meta = ref<Meta | null>(null)
const loading = ref(false)
const error = ref('')

const page = ref(1)
const search = ref('')
const status = ref<string | undefined>(undefined)

const statusItems = [
  { label: 'Semua status', value: undefined },
  { label: 'Active', value: 'active' },
  { label: 'Deprecated', value: 'deprecated' },
]

async function load() {
  loading.value = true
  error.value = ''
  try {
    const res = await api.get<PermissionResponse[]>(
      `/api/v1/solutions/${solutionId.value}/permissions`,
      { query: { page: page.value, limit: 50, search: search.value || undefined, status: status.value } },
    )
    items.value = res.data
    meta.value = res.meta ?? null
  } catch (err) {
    error.value = (err as { message?: string }).message || 'Gagal memuat permission'
  } finally {
    loading.value = false
  }
}

let debounce: ReturnType<typeof setTimeout> | undefined
watch(search, () => {
  clearTimeout(debounce)
  debounce = setTimeout(() => {
    page.value = 1
    load()
  }, 300)
})
watch(status, () => {
  page.value = 1
  load()
})

onMounted(load)
</script>

<template>
  <div>
    <PageHeader title="Permissions" subtitle="Katalog permission yang disinkronkan oleh solution.">
      <template #actions>
        <UButton
          :to="`/solutions/${solutionId}/roles`"
          color="neutral"
          variant="outline"
          icon="i-lucide-shield"
        >
          Global Roles
        </UButton>
      </template>
    </PageHeader>

    <UCard>
      <div class="mb-4 flex flex-wrap items-center gap-3">
        <UInput v-model="search" icon="i-lucide-search" placeholder="Cari kode / nama" class="w-64" />
        <USelect v-model="status" :items="statusItems" class="w-44" />
        <UButton color="neutral" variant="outline" icon="i-lucide-refresh-cw" @click="load">Muat ulang</UButton>
      </div>

      <UAlert v-if="error" color="error" variant="subtle" :title="error" class="mb-4" />

      <div class="overflow-x-auto">
        <table class="w-full text-sm">
          <thead>
            <tr class="border-b border-default text-left text-xs uppercase tracking-wide text-dimmed">
              <th class="px-3 py-2">Kode</th>
              <th class="px-3 py-2">Nama</th>
              <th class="px-3 py-2">Status</th>
            </tr>
          </thead>
          <tbody>
            <tr v-if="loading">
              <td colspan="3" class="px-3 py-6 text-center text-muted">Memuat...</td>
            </tr>
            <tr v-else-if="!items.length">
              <td colspan="3" class="px-3 py-6 text-center text-muted">Belum ada permission.</td>
            </tr>
            <template v-else>
              <tr v-for="p in items" :key="p.id" class="border-b border-default/60">
                <td class="px-3 py-2 font-mono text-xs text-highlighted">{{ p.code }}</td>
                <td class="px-3 py-2 text-highlighted">{{ p.name }}</td>
                <td class="px-3 py-2">
                  <UBadge size="sm" variant="subtle" :color="p.status === 'active' ? 'success' : 'neutral'">
                    {{ p.status }}
                  </UBadge>
                </td>
              </tr>
            </template>
          </tbody>
        </table>
      </div>

      <AppPaginator v-if="meta" :meta="meta" @change="(p) => { page = p; load() }" />
    </UCard>
  </div>
</template>