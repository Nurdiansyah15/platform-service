<script setup lang="ts">
import type { SolutionResponse, Meta } from '~/types/api'

const auth = useAuthStore()
const api = useApi()
const toast = useToast()

const items = ref<SolutionResponse[]>([])
const meta = ref<Meta | null>(null)
const loading = ref(false)
const error = ref('')

const page = ref(1)
const search = ref('')

const showCreate = ref(false)
const saving = ref(false)
const form = reactive({ code: '', name: '', description: '' })

async function load() {
  loading.value = true
  error.value = ''
  try {
    const res = await api.get<SolutionResponse[]>('/api/v1/solutions', {
      query: { page: page.value, limit: 20, search: search.value || undefined },
    })
    items.value = res.data
    meta.value = res.meta ?? null
  } catch (err) {
    error.value = (err as { message?: string }).message || 'Gagal memuat solution'
  } finally {
    loading.value = false
  }
}

async function createSolution() {
  saving.value = true
  try {
    await api.post('/api/v1/solutions', {
      code: form.code,
      name: form.name,
      description: form.description || null,
    })
    toast.add({ title: 'Solution dibuat', color: 'success' })
    showCreate.value = false
    form.code = ''
    form.name = ''
    form.description = ''
    await load()
  } catch (err) {
    toast.add({ title: (err as { message?: string }).message || 'Gagal membuat solution', color: 'error' })
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

onMounted(load)
</script>

<template>
  <div>
    <PageHeader title="Solutions" subtitle="Katalog produk yang dilayani platform.">
      <template #actions>
        <UButton v-if="auth.isPlatformAdmin" icon="i-lucide-plus" color="primary" @click="showCreate = !showCreate">
          Solution baru
        </UButton>
      </template>
    </PageHeader>

    <UCard v-if="showCreate && auth.isPlatformAdmin" class="mb-4">
      <form class="grid gap-4 sm:grid-cols-4" @submit.prevent="createSolution">
        <UFormField label="Kode">
          <UInput v-model="form.code" class="w-full" placeholder="easy-hr" />
        </UFormField>
        <UFormField label="Nama">
          <UInput v-model="form.name" class="w-full" placeholder="Easy HR" />
        </UFormField>
        <UFormField label="Deskripsi">
          <UInput v-model="form.description" class="w-full" placeholder="opsional" />
        </UFormField>
        <div class="flex items-end">
          <UButton type="submit" :loading="saving" color="primary">Simpan</UButton>
        </div>
      </form>
    </UCard>

    <UCard>
      <div class="mb-4 flex items-center gap-3">
        <UInput v-model="search" icon="i-lucide-search" placeholder="Cari nama / kode" class="w-64" />
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
              <th class="px-3 py-2"></th>
            </tr>
          </thead>
          <tbody>
            <tr v-if="loading">
              <td colspan="4" class="px-3 py-6 text-center text-muted">Memuat...</td>
            </tr>
            <tr v-else-if="!items.length">
              <td colspan="4" class="px-3 py-6 text-center text-muted">Belum ada solution.</td>
            </tr>
            <template v-else>
              <tr v-for="s in items" :key="s.id" class="border-b border-default/60">
                <td class="px-3 py-2 font-mono text-xs text-highlighted">{{ s.code }}</td>
                <td class="px-3 py-2 text-highlighted">{{ s.name }}</td>
                <td class="px-3 py-2">
                  <UBadge size="sm" variant="subtle" :color="s.status === 'active' ? 'success' : 'neutral'">
                    {{ s.status }}
                  </UBadge>
                </td>
                <td class="px-3 py-2 text-right">
                  <UButton size="xs" color="neutral" variant="outline" :to="`/solutions/${s.id}`">Detail</UButton>
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