<script setup lang="ts">
import type { UpdateUserRequest, UserDetailResponse, UserResponse, UserStatus } from '~/types/api'

const route = useRoute()
const auth = useAuthStore()
const api = useApi()
const toast = useToast()

const userId = computed(() => String(route.params.userId))
const isSelf = computed(() => auth.user?.id === userId.value)

const user = ref<UserDetailResponse | null>(null)
const loading = ref(false)
const error = ref('')

const saving = ref(false)
const form = reactive({ name: '', email: '', is_platform_admin: false })
const newPassword = ref('')
const resetting = ref(false)
const changingStatus = ref(false)

const statusActions: { status: UserStatus; label: string; icon: string; color: 'success' | 'warning' | 'error' }[] = [
  { status: 'active', label: 'Aktifkan', icon: 'i-lucide-circle-check', color: 'success' },
  { status: 'inactive', label: 'Nonaktifkan', icon: 'i-lucide-circle-pause', color: 'warning' },
  { status: 'suspended', label: 'Suspend', icon: 'i-lucide-ban', color: 'error' },
]

function fillForm(u: UserResponse) {
  form.name = u.name
  form.email = u.email
  form.is_platform_admin = u.is_platform_admin
}

async function load() {
  if (!auth.isPlatformAdmin) return
  loading.value = true
  error.value = ''
  try {
    const res = await api.get<UserDetailResponse>(`/api/v1/users/${userId.value}`)
    user.value = res.data
    fillForm(res.data)
  } catch (err) {
    error.value = (err as { message?: string }).message || 'Gagal memuat user'
  } finally {
    loading.value = false
  }
}

function applyUpdate(u: UserResponse) {
  if (user.value) user.value = { ...user.value, ...u }
  fillForm(u)
}

async function save() {
  if (!user.value) return
  saving.value = true
  try {
    // Kirim hanya field yang berubah.
    const body: UpdateUserRequest = {}
    if (form.name !== user.value.name) body.name = form.name
    if (form.email !== user.value.email) body.email = form.email
    if (form.is_platform_admin !== user.value.is_platform_admin) body.is_platform_admin = form.is_platform_admin
    const res = await api.patch<UserResponse>(`/api/v1/users/${userId.value}`, body)
    applyUpdate(res.data)
    toast.add({ title: 'User diperbarui', color: 'success' })
  } catch (err) {
    toast.add({ title: (err as { message?: string }).message || 'Gagal memperbarui user', color: 'error' })
  } finally {
    saving.value = false
  }
}

async function resetPassword() {
  resetting.value = true
  try {
    const res = await api.patch<UserResponse>(`/api/v1/users/${userId.value}`, { password: newPassword.value })
    applyUpdate(res.data)
    newPassword.value = ''
    toast.add({ title: 'Password di-reset', description: 'Sampaikan password baru ke user lewat jalur aman.', color: 'success' })
  } catch (err) {
    toast.add({ title: (err as { message?: string }).message || 'Gagal reset password', color: 'error' })
  } finally {
    resetting.value = false
  }
}

async function changeStatus(next: UserStatus) {
  if (next !== 'active' && !confirm(`Ubah status menjadi ${next}? User tidak bisa login dan token-nya langsung ditolak.`)) return
  changingStatus.value = true
  try {
    const res = await api.patch<UserResponse>(`/api/v1/users/${userId.value}/status`, { status: next })
    applyUpdate(res.data)
    toast.add({ title: `Status: ${next}`, color: 'success' })
  } catch (err) {
    toast.add({ title: (err as { message?: string }).message || 'Gagal mengubah status', color: 'error' })
  } finally {
    changingStatus.value = false
  }
}

function statusColor(s: string) {
  return s === 'active' ? 'success' : s === 'suspended' || s === 'closed' ? 'error' : 'warning'
}

onMounted(load)
</script>

<template>
  <div>
    <PageHeader :title="user?.name ?? 'User'" :subtitle="user?.email">
      <template #actions>
        <UButton to="/admin/users" color="neutral" variant="outline" icon="i-lucide-arrow-left">Kembali</UButton>
      </template>
    </PageHeader>

    <UAlert v-if="!auth.isPlatformAdmin" color="error" variant="subtle" icon="i-lucide-lock" title="Khusus platform admin" />
    <UAlert v-if="error" color="error" variant="subtle" :title="error" class="mb-4" />
    <p v-if="loading" class="text-sm text-muted">Memuat...</p>

    <div v-if="user" class="grid gap-6 lg:grid-cols-2">
      <UCard>
        <h2 class="mb-4 font-medium text-highlighted">Profil</h2>
        <form class="space-y-4" @submit.prevent="save">
          <UFormField label="Nama" required>
            <UInput v-model="form.name" class="w-full" />
          </UFormField>
          <UFormField label="Email" required>
            <UInput v-model="form.email" type="email" class="w-full" />
          </UFormField>
          <UCheckbox
            v-model="form.is_platform_admin"
            label="Platform admin"
            :disabled="isSelf && user.is_platform_admin"
            :description="isSelf ? 'Kamu tidak bisa mencabut flag admin milikmu sendiri.' : undefined"
          />
          <UButton type="submit" color="primary" :loading="saving">Simpan</UButton>
        </form>
      </UCard>

      <div class="space-y-6">
        <UCard>
          <h2 class="mb-1 font-medium text-highlighted">Status akun</h2>
          <p class="mb-4 text-sm text-muted">
            Saat ini:
            <UBadge size="sm" variant="subtle" :color="statusColor(user.status)">{{ user.status }}</UBadge>
          </p>
          <div class="flex flex-wrap gap-2">
            <UButton
              v-for="a in statusActions"
              :key="a.status"
              size="sm"
              variant="outline"
              :color="a.color"
              :icon="a.icon"
              :loading="changingStatus"
              :disabled="user.status === a.status || (isSelf && a.status !== 'active')"
              @click="changeStatus(a.status)"
            >
              {{ a.label }}
            </UButton>
          </div>
          <p v-if="isSelf" class="mt-2 text-xs text-dimmed">Kamu tidak bisa menonaktifkan akunmu sendiri.</p>
        </UCard>

        <UCard>
          <h2 class="mb-1 font-medium text-highlighted">Reset password</h2>
          <p class="mb-4 text-sm text-muted">
            {{ user.has_password ? 'Ganti password user.' : 'User belum punya password; set agar bisa login.' }}
          </p>
          <form class="flex gap-2" @submit.prevent="resetPassword">
            <UInput v-model="newPassword" type="password" placeholder="Password baru (min. 8 karakter)" class="flex-1" />
            <UButton type="submit" color="neutral" variant="outline" :loading="resetting" :disabled="newPassword.length < 8">
              Reset
            </UButton>
          </form>
        </UCard>
      </div>

      <UCard class="lg:col-span-2">
        <h2 class="mb-4 font-medium text-highlighted">Keanggotaan company</h2>
        <p v-if="!user.memberships.length" class="text-sm text-muted">User belum menjadi member company mana pun.</p>
        <div v-else class="overflow-x-auto">
          <table class="w-full text-sm">
            <thead>
              <tr class="border-b border-default text-left text-xs uppercase tracking-wide text-dimmed">
                <th class="px-3 py-2">Company</th>
                <th class="px-3 py-2">Status company</th>
                <th class="px-3 py-2">Status member</th>
                <th class="px-3 py-2">Bergabung</th>
              </tr>
            </thead>
            <tbody>
              <tr v-for="m in user.memberships" :key="m.company_id" class="border-b border-default/60">
                <td class="px-3 py-2">
                  <NuxtLink :to="`/companies/${m.company_id}/members/${user.id}`" class="font-medium text-highlighted hover:text-primary">
                    {{ m.company_name }}
                  </NuxtLink>
                  <p class="text-xs text-muted">{{ m.company_slug }}</p>
                </td>
                <td class="px-3 py-2">
                  <UBadge size="sm" variant="subtle" :color="statusColor(m.company_status)">{{ m.company_status }}</UBadge>
                </td>
                <td class="px-3 py-2">
                  <UBadge size="sm" variant="subtle" :color="statusColor(m.member_status)">{{ m.member_status }}</UBadge>
                </td>
                <td class="px-3 py-2 text-muted">{{ new Date(m.joined_at).toLocaleDateString() }}</td>
              </tr>
            </tbody>
          </table>
        </div>
      </UCard>
    </div>
  </div>
</template>
