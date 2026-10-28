<script setup lang="ts">
import type { BranchResponse, Meta } from '~/types/api'

const route = useRoute()
const auth = useAuthStore()
const api = useApi()
const toast = useToast()

const companyId = computed(() => String(route.params.companyId))
const isForeignCompany = computed(
  () => !auth.isPlatformAdmin && auth.activeCompanyId !== companyId.value,
)

const items = ref<BranchResponse[]>([])
const meta = ref<Meta | null>(null)
const loading = ref(false)
const error = ref('')

const page = ref(1)
const search = ref('')
const status = ref<string | undefined>(undefined)

const showCreate = ref(false)
const saving = ref(false)
const form = reactive({ name: '', code: '', status: 'active' as 'active' | 'inactive' })

const statusItems = [
  { label: 'Semua status', value: undefined },
  { label: 'Active', value: 'active' },
  { label: 'Inactive', value: 'inactive' },
]

async function load() {
  if (isForeignCompany.value) return
  loading.value = true
  error.value = ''
  try {
    const res = await api.get<BranchResponse[]>(
      `/api/v1/companies/${companyId.value}/branches`,
      { query: { page: page.value, limit: 20, search: search.value || undefined, status: status.value } },
    )
    items.value = res.data
    meta.value = res.meta ?? null
  } catch (err) {
    error.value = (err as { message?: string }).message || 'Gagal memuat branch'
  } finally {
    loading.value = false
  }
}

async function createBranch() {
  saving.value = true
  try {
    await api.post(`/api/v1/companies/${companyId.value}/branches`, {
      name: form.name,
      code: form.code,
      status: form.status,
    })
    toast.add({ title: 'Branch dibuat', color: 'success' })
    showCreate.value = false
    form.name = ''
    form.code = ''
    await load()
  } catch (err) {
    toast.add({ title: (err as { message?: string }).message || 'Gagal membuat branch', color: 'error' })
  } finally {
    saving.value = false
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
    <PageHeader title="Branches" subtitle="Unit organisasi di dalam company.">
      <template #actions>
        <UButton icon="i-lucide-plus" color="primary" :disabled="isForeignCompany" @click="showCreate = !showCreate">
          Branch baru
        </UButton>
      </template>
    </PageHeader>

    <UAlert
      v-if="isForeignCompany"
      class="mb-4"
      color="warning"
      variant="subtle"
      icon="i-lucide-triangle-alert"
      title="Company ini bukan company aktif"
      description="Aktifkan company ini dulu untuk mengelola branch."
    />

    <UCard v-if="showCreate && !isForeignCompany" class="mb-4">
      <form class="grid gap-4 sm:grid-cols-4" @submit.prevent="createBranch">
        <UFormField label="Nama">
          <UInput v-model="form.name" class="w-full" placeholder="Cabang Jakarta" />
        </UFormField>
        <UFormField label="Kode">
          <UInput v-model="form.code" class="w-full" placeholder="JKT" />
        </UFormField>
        <UFormField label="Status">
          <USelect
            v-model="form.status"
            :items="[{ label: 'Active', value: 'active' }, { label: 'Inactive', value: 'inactive' }]"
            class="w-full"
          />
        </UFormField>
        <div class="flex items-end">
          <UButton type="submit" :loading="saving" color="primary">Simpan</UButton>
        </div>
      </form>
    </UCard>

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
              <th class="px-3 py-2">Dibuat</th>
            </tr>
          </thead>
          <tbody>
            <tr v-if="loading">
              <td colspan="4" class="px-3 py-6 text-center text-muted">Memuat...</td>
            </tr>
            <tr v-else-if="!items.length">
              <td colspan="4" class="px-3 py-6 text-center text-muted">Belum ada branch.</td>
            </tr>
            <template v-else>
              <tr v-for="b in items" :key="b.id" class="border-b border-default/60">
                <td class="px-3 py-2 font-mono text-xs text-highlighted">{{ b.code }}</td>
                <td class="px-3 py-2 text-highlighted">{{ b.name }}</td>
                <td class="px-3 py-2">
                  <UBadge size="sm" variant="subtle" :color="b.status === 'active' ? 'success' : 'neutral'">
                    {{ b.status }}
                  </UBadge>
                </td>
                <td class="px-3 py-2 text-muted">{{ new Date(b.created_at).toLocaleDateString() }}</td>
              </tr>
            </template>
          </tbody>
        </table>
      </div>

      <AppPaginator v-if="meta" :meta="meta" @change="(p) => { page = p; load() }" />
    </UCard>
  </div>
</template>