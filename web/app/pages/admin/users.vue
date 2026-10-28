<script setup lang="ts">
import type { CreateUserRequest, UserResponse } from '~/types/api'

const auth = useAuthStore()
const api = useApi()
const toast = useToast()

const saving = ref(false)
const created = ref<UserResponse | null>(null)
const error = ref('')

const form = reactive<CreateUserRequest>({
  email: '',
  name: '',
  password: '',
  status: 'active',
  is_platform_admin: false,
})

const statusItems = [
  { label: 'Active', value: 'active' },
  { label: 'Inactive', value: 'inactive' },
  { label: 'Suspended', value: 'suspended' },
]

async function createUser() {
  error.value = ''
  saving.value = true
  try {
    const res = await api.post<UserResponse>('/api/v1/users', {
      email: form.email,
      name: form.name,
      password: form.password || undefined,
      status: form.status,
      is_platform_admin: form.is_platform_admin,
    })
    created.value = res.data
    toast.add({ title: 'User dibuat', color: 'success' })
    form.email = ''
    form.name = ''
    form.password = ''
    form.is_platform_admin = false
  } catch (err) {
    error.value = (err as { message?: string }).message || 'Gagal membuat user'
  } finally {
    saving.value = false
  }
}
</script>

<template>
  <div>
    <PageHeader
      title="Users"
      subtitle="Pembuatan akun platform (registrasi publik tidak tersedia)."
    />

    <UAlert
      v-if="!auth.isPlatformAdmin"
      color="error"
      variant="subtle"
      icon="i-lucide-lock"
      title="Khusus platform admin"
    />

    <template v-else>
      <UCard class="max-w-2xl">
        <form class="grid gap-4 sm:grid-cols-2" @submit.prevent="createUser">
          <UFormField label="Email">
            <UInput v-model="form.email" type="email" class="w-full" placeholder="user@perusahaan.com" />
          </UFormField>
          <UFormField label="Nama">
            <UInput v-model="form.name" class="w-full" placeholder="Nama lengkap" />
          </UFormField>
          <UFormField label="Password" hint="opsional">
            <UInput v-model="form.password" type="password" class="w-full" placeholder="min. 8 karakter" />
          </UFormField>
          <UFormField label="Status">
            <USelect v-model="form.status" :items="statusItems" class="w-full" />
          </UFormField>
          <label class="flex items-center gap-2 text-sm text-muted sm:col-span-2">
            <input v-model="form.is_platform_admin" type="checkbox">
            Jadikan platform admin
          </label>
          <div class="sm:col-span-2">
            <UAlert v-if="error" color="error" variant="subtle" :title="error" class="mb-3" />
            <UButton type="submit" color="primary" :loading="saving">Buat user</UButton>
          </div>
        </form>
      </UCard>

      <UAlert
        v-if="created"
        class="mt-4 max-w-2xl"
        color="success"
        variant="subtle"
        icon="i-lucide-circle-check"
        title="User berhasil dibuat"
        :description="`${created.name} · ${created.email}`"
      />

      <p class="mt-4 text-xs text-dimmed">
        Catatan: API belum menyediakan daftar/ubah user (fase lanjutan).
      </p>
    </template>
  </div>
</template>