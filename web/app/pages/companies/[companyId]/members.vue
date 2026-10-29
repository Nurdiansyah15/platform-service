<script setup lang="ts">
import type { CompanyMemberResponse, Meta, AddCompanyMemberRequest, MemberStatus } from '~/types/api'

const route = useRoute()
const auth = useAuthStore()
const api = useApi()
const toast = useToast()

const companyId = computed(() => String(route.params.companyId))
const isForeignCompany = computed(
  () => !auth.isPlatformAdmin && auth.activeCompanyId !== companyId.value,
)

const items = ref<CompanyMemberResponse[]>([])
const meta = ref<Meta | null>(null)
const loading = ref(false)
const error = ref('')

const page = ref(1)
const search = ref('')
const status = ref<string | undefined>(undefined)

const showAdd = ref(false)
const saving = ref(false)
const form = reactive<AddCompanyMemberRequest>({ email: '', user_id: '', status: 'active' })

const statusItems = [
  { label: 'Semua status', value: undefined },
  { label: 'Active', value: 'active' },
  { label: 'Invited', value: 'invited' },
  { label: 'Inactive', value: 'inactive' },
]

async function load() {
  if (isForeignCompany.value) return
  loading.value = true
  error.value = ''
  try {
    const res = await api.get<CompanyMemberResponse[]>(
      `/api/v1/companies/${companyId.value}/members`,
      { query: { page: page.value, limit: 20, search: search.value || undefined, status: status.value } },
    )
    items.value = res.data
    meta.value = res.meta ?? null
  } catch (err) {
    error.value = (err as { message?: string }).message || 'Gagal memuat member'
  } finally {
    loading.value = false
  }
}

async function addMember() {
  saving.value = true
  try {
    const body: AddCompanyMemberRequest = { status: form.status || 'active' }
    if (form.user_id) body.user_id = form.user_id
    if (form.email) body.email = form.email
    await api.post(`/api/v1/companies/${companyId.value}/members`, body)
    toast.add({ title: 'Member ditambahkan', color: 'success' })
    showAdd.value = false
    form.email = ''
    form.user_id = ''
    await load()
  } catch (err) {
    toast.add({ title: (err as { message?: string }).message || 'Gagal menambah member', color: 'error' })
  } finally {
    saving.value = false
  }
}

const busyId = ref<string | null>(null)

async function changeStatus(m: CompanyMemberResponse, next: MemberStatus) {
  busyId.value = m.user_id
  try {
    await api.patch(`/api/v1/companies/${companyId.value}/members/${m.user_id}`, { status: next })
    toast.add({ title: 'Status member diperbarui', color: 'success' })
    await load()
  } catch (err) {
    toast.add({ title: (err as { message?: string }).message || 'Gagal mengubah status', color: 'error' })
  } finally {
    busyId.value = null
  }
}

async function removeMember(m: CompanyMemberResponse) {
  if (!confirm(`Keluarkan ${m.email} dari company? Role assignment-nya di company ini ikut dihapus.`)) return
  busyId.value = m.user_id
  try {
    await api.del(`/api/v1/companies/${companyId.value}/members/${m.user_id}`)
    toast.add({ title: 'Member dikeluarkan', color: 'success' })
    await load()
  } catch (err) {
    toast.add({ title: (err as { message?: string }).message || 'Gagal mengeluarkan member', color: 'error' })
  } finally {
    busyId.value = null
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
    <PageHeader title="Members" subtitle="Anggota company dan status keanggotaan.">
      <template #actions>
        <UButton
          icon="i-lucide-user-plus"
          color="primary"
          :disabled="isForeignCompany"
          @click="showAdd = !showAdd"
        >
          Tambah member
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
      description="Aktifkan company ini dulu (company switcher) untuk mengelola membernya."
    />

    <UCard v-if="showAdd && !isForeignCompany" class="mb-4">
      <form class="grid gap-4 sm:grid-cols-4" @submit.prevent="addMember">
        <UFormField label="Email" hint="atau User ID">
          <UInput v-model="form.email" class="w-full" placeholder="user@perusahaan.com" />
        </UFormField>
        <UFormField label="User ID" hint="opsional">
          <UInput v-model="form.user_id" class="w-full" placeholder="uuid" />
        </UFormField>
        <UFormField label="Status">
          <USelect
            v-model="form.status"
            :items="[{ label: 'Active', value: 'active' }, { label: 'Invited', value: 'invited' }]"
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
        <UInput v-model="search" icon="i-lucide-search" placeholder="Cari email / nama" class="w-64" />
        <USelect v-model="status" :items="statusItems" class="w-44" />
        <UButton color="neutral" variant="outline" icon="i-lucide-refresh-cw" @click="load">Muat ulang</UButton>
      </div>

      <UAlert v-if="error" color="error" variant="subtle" :title="error" class="mb-4" />

      <div class="overflow-x-auto">
        <table class="w-full text-sm">
          <thead>
            <tr class="border-b border-default text-left text-xs uppercase tracking-wide text-dimmed">
              <th class="px-3 py-2">Nama</th>
              <th class="px-3 py-2">Email</th>
              <th class="px-3 py-2">Status</th>
              <th class="px-3 py-2">Bergabung</th>
              <th class="px-3 py-2" />
            </tr>
          </thead>
          <tbody>
            <tr v-if="loading">
              <td colspan="5" class="px-3 py-6 text-center text-muted">Memuat...</td>
            </tr>
            <tr v-else-if="!items.length">
              <td colspan="5" class="px-3 py-6 text-center text-muted">Tidak ada member.</td>
            </tr>
            <template v-else>
              <tr v-for="m in items" :key="m.user_id" class="border-b border-default/60">
                <td class="px-3 py-2 font-medium text-highlighted">{{ m.name }}</td>
                <td class="px-3 py-2 text-muted">{{ m.email }}</td>
                <td class="px-3 py-2">
                  <UBadge
                    size="sm"
                    variant="subtle"
                    :color="m.status === 'active' ? 'success' : m.status === 'inactive' ? 'error' : 'warning'"
                  >
                    {{ m.status }}
                  </UBadge>
                </td>
                <td class="px-3 py-2 text-muted">{{ new Date(m.joined_at).toLocaleDateString() }}</td>
                <td class="px-3 py-2">
                  <div v-if="!isForeignCompany" class="flex justify-end gap-1">
                    <UButton
                      v-if="m.status === 'invited'"
                      size="xs"
                      color="neutral"
                      variant="outline"
                      icon="i-lucide-check"
                      :loading="busyId === m.user_id"
                      @click="changeStatus(m, 'active')"
                    >
                      Aktifkan
                    </UButton>
                    <UButton
                      v-if="m.status === 'inactive'"
                      size="xs"
                      color="neutral"
                      variant="outline"
                      icon="i-lucide-undo-2"
                      :loading="busyId === m.user_id"
                      @click="changeStatus(m, 'active')"
                    >
                      Aktifkan lagi
                    </UButton>
                    <UButton
                      v-if="m.status !== 'inactive'"
                      size="xs"
                      color="error"
                      variant="outline"
                      icon="i-lucide-user-minus"
                      :loading="busyId === m.user_id"
                      @click="removeMember(m)"
                    >
                      Keluarkan
                    </UButton>
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