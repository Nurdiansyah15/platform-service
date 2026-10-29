<script setup lang="ts">
import type { CreateUserRequest, Meta, UserResponse, UserStatus } from '~/types/api'

const auth = useAuthStore()
const api = useApi()
const toast = useToast()

const items = ref<UserResponse[]>([])
const meta = ref<Meta | null>(null)
const loading = ref(false)
const error = ref('')

const page = ref(1)
const search = ref('')
const status = ref<UserStatus | undefined>(undefined)
const role = ref<'admin' | 'user' | undefined>(undefined)

const statusItems = [
  { label: 'Semua status', value: undefined },
  { label: 'Active', value: 'active' },
  { label: 'Inactive', value: 'inactive' },
  { label: 'Suspended', value: 'suspended' },
]
const roleItems = [
  { label: 'Semua user', value: undefined },
  { label: 'Platform admin', value: 'admin' },
  { label: 'User biasa', value: 'user' },
]

const showCreate = ref(false)
const saving = ref(false)
const form = reactive<CreateUserRequest>({
  email: '',
  name: '',
  password: '',
  status: 'active',
  is_platform_admin: false,
})

async function load() {
  if (!auth.isPlatformAdmin) return
  loading.value = true
  error.value = ''
  try {
    const res = await api.get<UserResponse[]>('/api/v1/users', {
      query: {
        page: page.value,
        limit: 20,
        search: search.value || undefined,
        status: status.value,
        is_platform_admin: role.value === undefined ? undefined : role.value === 'admin',
      },
    })
    items.value = res.data
    meta.value = res.meta ?? null
  } catch (err) {
    error.value = (err as { message?: string }).message || 'Gagal memuat user'
  } finally {
    loading.value = false
  }
}

async function createUser() {
  saving.value = true
  try {
    const res = await api.post<UserResponse>('/api/v1/users', {
      email: form.email,
      name: form.name,
      password: form.password || undefined,
      status: form.status,
      is_platform_admin: form.is_platform_admin,
    })
    toast.add({ title: 'User dibuat', description: `${res.data.name} · ${res.data.email}`, color: 'success' })
    form.email = ''
    form.name = ''
    form.password = ''
    form.is_platform_admin = false
    showCreate.value = false
    await load()
  } catch (err) {
    toast.add({ title: (err as { message?: string }).message || 'Gagal membuat user', color: 'error' })
  } finally {
    saving.value = false
  }
}

function statusColor(s: string) {
  return s === 'active' ? 'success' : s === 'suspended' ? 'error' : 'warning'
}

let debounce: ReturnType<typeof setTimeout> | undefined
watch(search, () => {
  clearTimeout(debounce)
  debounce = setTimeout(() => {
    page.value = 1
    load()
  }, 300)
})
watch([status, role], () => {
  page.value = 1
  load()
})

onMounted(load)
</script>

<template>
  <div>
    <PageHeader title="Users" subtitle="Akun platform. Registrasi publik tidak tersedia; akun dibuat admin.">
      <template #actions>
        <UButton v-if="auth.isPlatformAdmin" icon="i-lucide-user-plus" color="primary" @click="showCreate = !showCreate">
          Buat user
        </UButton>
      </template>
    </PageHeader>

    <UAlert
      v-if="!auth.isPlatformAdmin"
      color="error"
      variant="subtle"
      icon="i-lucide-lock"
      title="Khusus platform admin"
    />

    <template v-else>
      <UCard v-if="showCreate" class="mb-4">
        <form class="grid gap-4 sm:grid-cols-2" @submit.prevent="createUser">
          <UFormField label="Email" required>
            <UInput v-model="form.email" type="email" class="w-full" placeholder="user@perusahaan.com" />
          </UFormField>
          <UFormField label="Nama" required>
            <UInput v-model="form.name" class="w-full" placeholder="Nama lengkap" />
          </UFormField>
          <UFormField label="Password" hint="opsional; tanpa password user belum bisa login">
            <UInput v-model="form.password" type="password" class="w-full" placeholder="min. 8 karakter" />
          </UFormField>
          <UFormField label="Status">
            <USelect v-model="form.status" :items="statusItems.slice(1)" class="w-full" />
          </UFormField>
          <UCheckbox v-model="form.is_platform_admin" label="Jadikan platform admin" class="sm:col-span-2" />
          <div class="flex gap-2 sm:col-span-2">
            <UButton type="submit" color="primary" :loading="saving">Buat user</UButton>
            <UButton color="neutral" variant="ghost" @click="showCreate = false">Batal</UButton>
          </div>
        </form>
      </UCard>

      <UCard>
        <div class="mb-4 flex flex-wrap items-center gap-3">
          <UInput v-model="search" icon="i-lucide-search" placeholder="Cari email / nama" class="w-64" />
          <USelect v-model="status" :items="statusItems" class="w-44" />
          <USelect v-model="role" :items="roleItems" class="w-44" />
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
                <th class="px-3 py-2">Peran</th>
                <th class="px-3 py-2">Dibuat</th>
              </tr>
            </thead>
            <tbody>
              <tr v-if="loading">
                <td colspan="5" class="px-3 py-6 text-center text-muted">Memuat...</td>
              </tr>
              <tr v-else-if="!items.length">
                <td colspan="5" class="px-3 py-6 text-center text-muted">Tidak ada user.</td>
              </tr>
              <template v-else>
                <tr v-for="u in items" :key="u.id" class="border-b border-default/60">
                  <td class="px-3 py-2">
                    <NuxtLink :to="`/admin/users/${u.id}`" class="font-medium text-highlighted hover:text-primary">
                      {{ u.name }}
                    </NuxtLink>
                  </td>
                  <td class="px-3 py-2 text-muted">{{ u.email }}</td>
                  <td class="px-3 py-2">
                    <UBadge size="sm" variant="subtle" :color="statusColor(u.status)">{{ u.status }}</UBadge>
                  </td>
                  <td class="px-3 py-2">
                    <UBadge v-if="u.is_platform_admin" size="sm" color="primary" variant="subtle">Platform admin</UBadge>
                    <span v-else class="text-xs text-muted">User</span>
                    <UBadge v-if="!u.has_password" size="sm" color="warning" variant="subtle" class="ml-1">tanpa password</UBadge>
                  </td>
                  <td class="px-3 py-2 text-muted">{{ new Date(u.created_at).toLocaleDateString() }}</td>
                </tr>
              </template>
            </tbody>
          </table>
        </div>

        <AppPaginator v-if="meta" :meta="meta" @change="(p) => { page = p; load() }" />
      </UCard>
    </template>
  </div>
</template>
