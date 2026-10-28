<script setup lang="ts">
import type { CompanyResponse, Meta } from '~/types/api'

const auth = useAuthStore()
const api = useApi()
const toast = useToast()

const items = ref<CompanyResponse[]>([])
const meta = ref<Meta | null>(null)
const loading = ref(false)
const error = ref('')

const page = ref(1)
const limit = ref(20)
const search = ref('')
const status = ref<string | undefined>(undefined)

const showCreate = ref(false)
const creating = ref(false)
const form = reactive({ name: '', slug: '' })

const statusItems = [
  { label: 'Semua status', value: undefined },
  { label: 'Active', value: 'active' },
  { label: 'Suspended', value: 'suspended' },
  { label: 'Closed', value: 'closed' },
]

async function load() {
  loading.value = true
  error.value = ''
  try {
    const res = await api.get<CompanyResponse[]>('/api/v1/companies', {
      query: { page: page.value, limit: limit.value, search: search.value || undefined, status: status.value },
    })
    items.value = res.data
    meta.value = res.meta ?? null
  } catch (err) {
    error.value = (err as { message?: string }).message || 'Gagal memuat company'
  } finally {
    loading.value = false
  }
}

async function createCompany() {
  creating.value = true
  try {
    const res = await api.post<CompanyResponse>('/api/v1/companies', { name: form.name, slug: form.slug })
    toast.add({ title: 'Company dibuat', color: 'success' })
    showCreate.value = false
    form.name = ''
    form.slug = ''
    await auth.loadCompanies()
    await load()
    await navigateTo(`/companies/${res.data.id}`)
  } catch (err) {
    toast.add({ title: (err as { message?: string }).message || 'Gagal membuat company', color: 'error' })
  } finally {
    creating.value = false
  }
}

function onPage(p: number) {
  page.value = p
  load()
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
    <PageHeader title="Companies" subtitle="Company adalah tenant tertinggi pada platform.">
      <template #actions>
        <UButton icon="i-lucide-plus" color="primary" @click="showCreate = !showCreate">
          Company baru
        </UButton>
      </template>
    </PageHeader>

    <UCard v-if="showCreate" class="mb-4">
      <form class="grid gap-4 sm:grid-cols-3" @submit.prevent="createCompany">
        <UFormField label="Nama">
          <UInput v-model="form.name" class="w-full" placeholder="PT Contoh" />
        </UFormField>
        <UFormField label="Slug">
          <UInput v-model="form.slug" class="w-full" placeholder="pt-contoh" />
        </UFormField>
        <div class="flex items-end">
          <UButton type="submit" :loading="creating" color="primary">Simpan</UButton>
        </div>
      </form>
    </UCard>

    <UCard>
      <div class="mb-4 flex flex-wrap items-center gap-3">
        <UInput v-model="search" icon="i-lucide-search" placeholder="Cari name / slug" class="w-64" />
        <USelect v-model="status" :items="statusItems" class="w-44" />
        <UButton color="neutral" variant="outline" icon="i-lucide-refresh-cw" @click="load">Muat ulang</UButton>
      </div>

      <UAlert v-if="error" color="error" variant="subtle" :title="error" class="mb-4" />

      <div class="overflow-x-auto">
        <table class="w-full text-sm">
          <thead>
            <tr class="border-b border-default text-left text-xs uppercase tracking-wide text-dimmed">
              <th class="px-3 py-2">Nama</th>
              <th class="px-3 py-2">Slug</th>
              <th class="px-3 py-2">Status</th>
              <th class="px-3 py-2">Dibuat</th>
              <th class="px-3 py-2"></th>
            </tr>
          </thead>
          <tbody>
            <tr v-if="loading">
              <td colspan="5" class="px-3 py-6 text-center text-muted">Memuat...</td>
            </tr>
            <tr v-else-if="!items.length">
              <td colspan="5" class="px-3 py-6 text-center text-muted">Tidak ada data.</td>
            </tr>
            <template v-else>
              <tr
                v-for="c in items"
                :key="c.id"
                class="border-b border-default/60 hover:bg-elevated/60"
              >
                <td class="px-3 py-2 font-medium text-highlighted">{{ c.name }}</td>
                <td class="px-3 py-2 text-muted">{{ c.slug }}</td>
                <td class="px-3 py-2">
                  <UBadge
                    size="sm"
                    variant="subtle"
                    :color="c.status === 'active' ? 'success' : c.status === 'closed' ? 'error' : 'warning'"
                  >
                    {{ c.status }}
                  </UBadge>
                </td>
                <td class="px-3 py-2 text-muted">{{ new Date(c.created_at).toLocaleDateString() }}</td>
                <td class="px-3 py-2 text-right">
                  <UButton size="xs" color="neutral" variant="outline" :to="`/companies/${c.id}`">
                    Detail
                  </UButton>
                </td>
              </tr>
            </template>
          </tbody>
        </table>
      </div>

      <AppPaginator v-if="meta" :meta="meta" @change="onPage" />
    </UCard>
  </div>
</template>
