<script setup lang="ts">
import {
  SERVICE_SCOPES,
  type Meta,
  type SolutionClientResponse,
  type SolutionClientWithSecretResponse,
} from '~/types/api'

const route = useRoute()
const auth = useAuthStore()
const api = useApi()
const toast = useToast()

const solutionId = computed(() => String(route.params.solutionId))

const items = ref<SolutionClientResponse[]>([])
const meta = ref<Meta | null>(null)
const loading = ref(false)
const error = ref('')

const page = ref(1)
const search = ref('')
const status = ref<string | undefined>(undefined)

const statusItems = [
  { label: 'Semua status', value: undefined },
  { label: 'Active', value: 'active' },
  { label: 'Revoked', value: 'revoked' },
]
const scopeItems = SERVICE_SCOPES.map(s => ({ label: s, value: s }))

// Form create / edit dipakai bersama; editingId null = create.
const showForm = ref(false)
const editingId = ref<string | null>(null)
const saving = ref(false)
const form = reactive<{ name: string; scopes: string[] }>({ name: '', scopes: [] })

// Secret hanya dikembalikan sekali (create / rotate) — tampilkan sampai ditutup.
const revealed = ref<SolutionClientWithSecretResponse | null>(null)
const busyId = ref<string | null>(null)

async function load() {
  loading.value = true
  error.value = ''
  try {
    const res = await api.get<SolutionClientResponse[]>(`/api/v1/solutions/${solutionId.value}/clients`, {
      query: { page: page.value, limit: 20, search: search.value || undefined, status: status.value },
    })
    items.value = res.data
    meta.value = res.meta ?? null
  } catch (err) {
    error.value = (err as { message?: string }).message || 'Gagal memuat client'
  } finally {
    loading.value = false
  }
}

function openCreate() {
  editingId.value = null
  form.name = ''
  form.scopes = []
  showForm.value = true
}

function openEdit(c: SolutionClientResponse) {
  editingId.value = c.id
  form.name = c.name
  form.scopes = [...c.scopes]
  showForm.value = true
}

async function save() {
  saving.value = true
  try {
    const body = { name: form.name, scopes: form.scopes }
    if (editingId.value) {
      await api.patch(`/api/v1/solution-clients/${editingId.value}`, body)
      toast.add({ title: 'Client diperbarui', color: 'success' })
    } else {
      const res = await api.post<SolutionClientWithSecretResponse>(`/api/v1/solutions/${solutionId.value}/clients`, body)
      revealed.value = res.data
      toast.add({ title: 'Client dibuat', color: 'success' })
    }
    showForm.value = false
    await load()
  } catch (err) {
    toast.add({ title: (err as { message?: string }).message || 'Gagal menyimpan client', color: 'error' })
  } finally {
    saving.value = false
  }
}

async function rotate(c: SolutionClientResponse) {
  if (!confirm(`Rotate secret "${c.name}"? Secret lama langsung tidak berlaku.`)) return
  busyId.value = c.id
  try {
    const res = await api.post<SolutionClientWithSecretResponse>(`/api/v1/solution-clients/${c.id}/rotate-secret`)
    revealed.value = res.data
    toast.add({ title: 'Secret dirotasi', color: 'success' })
  } catch (err) {
    toast.add({ title: (err as { message?: string }).message || 'Gagal rotate secret', color: 'error' })
  } finally {
    busyId.value = null
  }
}

async function revoke(c: SolutionClientResponse) {
  if (!confirm(`Revoke "${c.name}"? Client tidak bisa dipakai lagi dan token yang sudah terbit langsung ditolak.`)) return
  busyId.value = c.id
  try {
    await api.post(`/api/v1/solution-clients/${c.id}/revoke`)
    toast.add({ title: 'Client di-revoke', color: 'success' })
    await load()
  } catch (err) {
    toast.add({ title: (err as { message?: string }).message || 'Gagal revoke client', color: 'error' })
  } finally {
    busyId.value = null
  }
}

async function copy(text: string) {
  try {
    await navigator.clipboard.writeText(text)
    toast.add({ title: 'Disalin', color: 'success' })
  } catch {
    toast.add({ title: 'Gagal menyalin', color: 'error' })
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
    <PageHeader title="Solution Clients" subtitle="Kredensial client credentials untuk backend solution (OAuth2).">
      <template #actions>
        <UButton :to="`/solutions/${solutionId}`" color="neutral" variant="outline" icon="i-lucide-arrow-left">Kembali</UButton>
        <UButton v-if="auth.isPlatformAdmin" icon="i-lucide-plus" color="primary" @click="openCreate">Buat client</UButton>
      </template>
    </PageHeader>

    <UAlert
      v-if="!auth.isPlatformAdmin"
      class="mb-4"
      color="warning"
      variant="subtle"
      icon="i-lucide-triangle-alert"
      title="Khusus admin platform"
      description="Hanya admin platform yang bisa mengelola solution client."
    />

    <UCard v-if="revealed" class="mb-4">
      <UAlert
        color="warning"
        variant="subtle"
        icon="i-lucide-key-round"
        title="Simpan client secret sekarang"
        description="Secret ini hanya ditampilkan sekali. Setelah ditutup, secret tidak bisa dilihat lagi (rotate untuk membuat yang baru)."
        class="mb-4"
      />
      <div class="grid gap-4 sm:grid-cols-2">
        <UFormField label="client_id">
          <div class="flex gap-2">
            <UInput :model-value="revealed.client_id" readonly class="w-full font-mono" />
            <UButton color="neutral" variant="outline" icon="i-lucide-copy" @click="copy(revealed.client_id)" />
          </div>
        </UFormField>
        <UFormField label="client_secret">
          <div class="flex gap-2">
            <UInput :model-value="revealed.client_secret" readonly class="w-full font-mono" />
            <UButton color="neutral" variant="outline" icon="i-lucide-copy" @click="copy(revealed.client_secret)" />
          </div>
        </UFormField>
      </div>
      <div class="mt-4 flex justify-end">
        <UButton color="neutral" variant="outline" @click="revealed = null">Sudah disimpan, tutup</UButton>
      </div>
    </UCard>

    <UCard v-if="showForm" class="mb-4">
      <form class="grid gap-4 sm:grid-cols-3" @submit.prevent="save">
        <UFormField label="Nama" required>
          <UInput v-model="form.name" class="w-full" placeholder="Easy HR Backend Prod" />
        </UFormField>
        <UFormField label="Scopes" hint="kosong = tanpa akses API">
          <USelectMenu v-model="form.scopes" :items="scopeItems" value-key="value" multiple class="w-full" placeholder="Pilih scope" />
        </UFormField>
        <div class="flex items-end gap-2">
          <UButton type="submit" :loading="saving" color="primary">{{ editingId ? 'Simpan' : 'Buat' }}</UButton>
          <UButton color="neutral" variant="ghost" @click="showForm = false">Batal</UButton>
        </div>
      </form>
    </UCard>

    <UCard>
      <div class="mb-4 flex flex-wrap items-center gap-3">
        <UInput v-model="search" icon="i-lucide-search" placeholder="Cari nama / client_id" class="w-64" />
        <USelect v-model="status" :items="statusItems" class="w-44" />
        <UButton color="neutral" variant="outline" icon="i-lucide-refresh-cw" @click="load">Muat ulang</UButton>
      </div>

      <UAlert v-if="error" color="error" variant="subtle" :title="error" class="mb-4" />

      <div class="overflow-x-auto">
        <table class="w-full text-sm">
          <thead>
            <tr class="border-b border-default text-left text-xs uppercase tracking-wide text-dimmed">
              <th class="px-3 py-2">Nama</th>
              <th class="px-3 py-2">Client ID</th>
              <th class="px-3 py-2">Scopes</th>
              <th class="px-3 py-2">Status</th>
              <th class="px-3 py-2">Terakhir dipakai</th>
              <th class="px-3 py-2" />
            </tr>
          </thead>
          <tbody>
            <tr v-if="loading">
              <td colspan="6" class="px-3 py-6 text-center text-muted">Memuat...</td>
            </tr>
            <tr v-else-if="!items.length">
              <td colspan="6" class="px-3 py-6 text-center text-muted">Belum ada client.</td>
            </tr>
            <template v-else>
              <tr v-for="c in items" :key="c.id" class="border-b border-default/60">
                <td class="px-3 py-2 font-medium text-highlighted">{{ c.name }}</td>
                <td class="px-3 py-2 font-mono text-xs text-muted">{{ c.client_id }}</td>
                <td class="px-3 py-2">
                  <div class="flex flex-wrap gap-1">
                    <UBadge v-for="s in c.scopes" :key="s" size="sm" color="neutral" variant="subtle">{{ s }}</UBadge>
                    <span v-if="!c.scopes.length" class="text-xs text-dimmed">—</span>
                  </div>
                </td>
                <td class="px-3 py-2">
                  <UBadge size="sm" variant="subtle" :color="c.status === 'active' ? 'success' : 'error'">{{ c.status }}</UBadge>
                </td>
                <td class="px-3 py-2 text-muted">{{ c.last_used_at ? new Date(c.last_used_at).toLocaleString() : '—' }}</td>
                <td class="px-3 py-2">
                  <div v-if="c.status === 'active' && auth.isPlatformAdmin" class="flex justify-end gap-1">
                    <UButton size="xs" color="neutral" variant="outline" icon="i-lucide-pencil" @click="openEdit(c)">Edit</UButton>
                    <UButton size="xs" color="neutral" variant="outline" icon="i-lucide-rotate-cw" :loading="busyId === c.id" @click="rotate(c)">Rotate</UButton>
                    <UButton size="xs" color="error" variant="outline" icon="i-lucide-ban" :loading="busyId === c.id" @click="revoke(c)">Revoke</UButton>
                  </div>
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
