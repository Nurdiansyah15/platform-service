<script setup lang="ts">
import type { FormError, FormSubmitEvent } from '@nuxt/ui'
import type { CompanyResponse, UpdateUserRequest, UserDetailResponse, UserResponse, UserStatus } from '~/types/api'

definePageMeta({ middleware: 'admin' })

const route = useRoute()
const auth = useAuthStore()
const api = useApi()
const toast = useToast()
const { toastError, fieldErrors } = useApiError()
const { confirm } = useConfirm()
const profileForm = useTemplateRef('profileForm')

const userId = computed(() => String(route.params.userId))
const isSelf = computed(() => auth.user?.id === userId.value)
const user = ref<UserDetailResponse | null>(null)
const error = ref('')

const profile = reactive({ name: '', email: '', admin: false })
const savingProfile = ref(false)
const newPassword = ref('')
const showPassword = ref(false)
const resetting = ref(false)

function fill(u: UserResponse) {
  Object.assign(profile, { name: u.name, email: u.email, admin: u.is_platform_admin })
}

async function load() {
  error.value = ''
  try {
    user.value = (await api.get<UserDetailResponse>(`/api/v1/users/${userId.value}`)).data
    fill(user.value)
  } catch (err) {
    error.value = (err as { message?: string }).message || 'Gagal memuat user'
  }
}
watch(userId, load, { immediate: true })

function apply(u: UserResponse) {
  if (user.value) user.value = { ...user.value, ...u }
  fill(u)
}

const dirty = computed(() => !!user.value && (
  profile.name.trim() !== user.value.name
  || profile.email.trim() !== user.value.email
  || profile.admin !== user.value.is_platform_admin))

function validateProfile(s: typeof profile): FormError[] {
  const errors: FormError[] = []
  if (!s.name.trim()) errors.push({ name: 'name', message: 'Nama wajib diisi' })
  if (!/^\S+@\S+\.\S+$/.test(s.email.trim())) errors.push({ name: 'email', message: 'Format email tidak valid' })
  return errors
}

async function saveProfile(e: FormSubmitEvent<typeof profile>) {
  if (!user.value) return
  const body: UpdateUserRequest = {}
  if (e.data.name.trim() !== user.value.name) body.name = e.data.name.trim()
  if (e.data.email.trim() !== user.value.email) body.email = e.data.email.trim()
  if (e.data.admin !== user.value.is_platform_admin) {
    const ok = await confirm({
      title: e.data.admin ? 'Jadikan platform admin?' : 'Cabut akses platform admin?',
      description: e.data.admin
        ? `${user.value.name} akan punya akses penuh ke semua company, solution, dan user.`
        : `${user.value.name} hanya bisa mengakses company tempat ia menjadi anggota.`,
      confirmLabel: e.data.admin ? 'Jadikan admin' : 'Cabut akses',
      danger: e.data.admin,
    })
    if (!ok) return
    body.is_platform_admin = e.data.admin
  }
  savingProfile.value = true
  try {
    apply((await api.patch<UserResponse>(`/api/v1/users/${userId.value}`, body)).data)
    toast.add({ title: 'User diperbarui', color: 'success' })
  } catch (err) {
    const fields = fieldErrors(err)
    if (fields.length) profileForm.value?.setErrors(fields)
    else toastError(err, 'Gagal memperbarui user')
  } finally {
    savingProfile.value = false
  }
}

function generate() {
  const chars = 'ABCDEFGHJKLMNPQRSTUVWXYZabcdefghijkmnpqrstuvwxyz23456789'
  const buf = new Uint32Array(14)
  crypto.getRandomValues(buf)
  newPassword.value = Array.from(buf, n => chars[n % chars.length]).join('')
  showPassword.value = true
}

async function resetPassword() {
  const ok = await confirm({
    title: 'Set password baru?',
    description: 'Password lama langsung tidak berlaku. Sampaikan password baru ke user lewat jalur aman.',
    confirmLabel: 'Set password',
  })
  if (!ok) return
  resetting.value = true
  try {
    apply((await api.patch<UserResponse>(`/api/v1/users/${userId.value}`, { password: newPassword.value })).data)
    newPassword.value = ''
    showPassword.value = false
    toast.add({ title: 'Password di-set', color: 'success' })
  } catch (err) {
    toastError(err, 'Gagal set password')
  } finally {
    resetting.value = false
  }
}

const statusActions: { status: UserStatus; label: string; description: string; color: 'success' | 'warning' | 'error' }[] = [
  { status: 'active', label: 'Aktifkan', description: 'User bisa login lagi.', color: 'success' },
  { status: 'inactive', label: 'Nonaktifkan', description: 'User tidak bisa login; token-nya langsung ditolak.', color: 'warning' },
  { status: 'suspended', label: 'Tangguhkan', description: 'Untuk pelanggaran; user tidak bisa login.', color: 'error' },
]

async function changeStatus(next: UserStatus) {
  if (!user.value) return
  const meta = statusActions.find(a => a.status === next)!
  if (next !== 'active') {
    const ok = await confirm({ title: `${meta.label} ${user.value.name}?`, description: meta.description, confirmLabel: meta.label, danger: true })
    if (!ok) return
  }
  try {
    apply((await api.patch<UserResponse>(`/api/v1/users/${userId.value}/status`, { status: next })).data)
    toast.add({ title: `Status: ${statusMeta(next).label}`, color: 'success' })
  } catch (err) {
    toastError(err)
  }
}

// ── Tambahkan ke company ──
const showAddCompany = ref(false)
const companies = ref<CompanyResponse[]>([])
const companyId = ref<string | undefined>()
const addStatus = ref<'active' | 'invited'>('active')
const addingCompany = ref(false)
const companyItems = computed(() => companies.value
  .filter(c => !user.value?.memberships.some(m => m.company_id === c.id && m.member_status !== 'inactive'))
  .map(c => ({ label: c.name, value: c.id, description: c.slug })))

watch(showAddCompany, async (v) => {
  if (!v) return
  companyId.value = undefined
  addStatus.value = 'active'
  try {
    companies.value = (await api.get<CompanyResponse[]>('/api/v1/companies', { query: { limit: 100, status: 'active' } })).data
  } catch (err) {
    toastError(err, 'Gagal memuat company')
  }
})

async function addToCompany() {
  if (!companyId.value) return
  addingCompany.value = true
  try {
    await api.post(`/api/v1/companies/${companyId.value}/members`, { user_id: userId.value, status: addStatus.value })
    toast.add({ title: 'Ditambahkan ke company', color: 'success' })
    showAddCompany.value = false
    load()
  } catch (err) {
    toastError(err, 'Gagal menambahkan ke company')
  } finally {
    addingCompany.value = false
  }
}
</script>

<template>
  <AppPage
    :title="user?.name ?? 'User'"
    :breadcrumbs="[{ label: 'Users', icon: 'i-lucide-user-cog', to: '/admin/users' }, { label: user?.name ?? '…' }]"
  >
    <UAlert v-if="error" color="error" variant="subtle" :title="error"
      :actions="[{ label: 'Kembali ke daftar', color: 'neutral', variant: 'outline', to: '/admin/users' }]" />

    <template v-else-if="user">
      <div class="flex flex-wrap items-center gap-3">
        <UAvatar :text="initials(user.name)" size="xl" />
        <div class="min-w-0">
          <p class="flex flex-wrap items-center gap-2 text-lg font-semibold text-highlighted">
            {{ user.name }}
            <StatusBadge :status="user.status" />
            <UBadge v-if="user.is_platform_admin" variant="subtle" color="primary" size="sm">Platform admin</UBadge>
            <UBadge v-if="isSelf" variant="outline" color="neutral" size="sm">Kamu</UBadge>
          </p>
          <p class="text-sm text-muted">{{ user.email }} · dibuat {{ formatDate(user.created_at) }}</p>
        </div>
      </div>

      <div class="grid gap-6 lg:grid-cols-2">
        <SectionCard title="Profil">
          <UForm ref="profileForm" :state="profile" :validate="validateProfile" class="space-y-4" @submit="saveProfile">
            <UFormField label="Nama" name="name" required>
              <UInput v-model="profile.name" class="w-full" />
            </UFormField>
            <UFormField label="Email" name="email" required>
              <UInput v-model="profile.email" type="email" class="w-full" />
            </UFormField>
            <USwitch
              v-model="profile.admin"
              label="Platform admin"
              :disabled="isSelf && user.is_platform_admin"
              :description="isSelf && user.is_platform_admin ? 'Kamu tidak bisa mencabut akses admin milikmu sendiri.' : 'Akses penuh ke semua company, solution, dan user.'"
            />
            <UButton type="submit" :loading="savingProfile" :disabled="!dirty">Simpan perubahan</UButton>
          </UForm>
        </SectionCard>

        <div class="space-y-6">
          <SectionCard title="Password" :description="user.has_password ? 'Set password baru untuk user ini.' : 'User belum punya password dan belum bisa login.'">
            <form class="space-y-3" @submit.prevent="resetPassword">
              <UFieldGroup class="w-full">
                <UInput v-model="newPassword" :type="showPassword ? 'text' : 'password'" placeholder="Min. 8 karakter" class="flex-1" autocomplete="new-password" />
                <UButton color="neutral" variant="outline" :icon="showPassword ? 'i-lucide-eye-off' : 'i-lucide-eye'" aria-label="Tampilkan password" @click="showPassword = !showPassword" />
                <UButton color="neutral" variant="outline" icon="i-lucide-wand-sparkles" @click="generate">Buat</UButton>
              </UFieldGroup>
              <div class="flex items-center gap-2">
                <UButton type="submit" :loading="resetting" :disabled="newPassword.length < 8">Set password</UButton>
                <CopyText v-if="newPassword && showPassword" :value="newPassword" label="Salin" />
              </div>
            </form>
          </SectionCard>

          <SectionCard title="Status akun">
            <ul class="divide-y divide-default rounded-lg border border-default">
              <li v-for="a in statusActions.filter(x => x.status !== user?.status)" :key="a.status" class="flex items-center justify-between gap-4 p-3">
                <div>
                  <p class="text-sm font-medium text-highlighted">{{ a.label }}</p>
                  <p class="text-xs text-muted">{{ a.description }}</p>
                </div>
                <UButton
                  size="sm"
                  :color="a.color"
                  variant="outline"
                  :disabled="isSelf && a.status !== 'active'"
                  @click="changeStatus(a.status)"
                >
                  {{ a.label }}
                </UButton>
              </li>
            </ul>
            <p v-if="isSelf" class="text-xs text-muted">Kamu tidak bisa menonaktifkan akunmu sendiri.</p>
          </SectionCard>
        </div>
      </div>

      <SectionCard title="Keanggotaan company">
        <template #actions>
          <UButton icon="i-lucide-plus" size="sm" @click="showAddCompany = true">Tambahkan ke company</UButton>
        </template>
        <UEmpty v-if="!user.memberships.length" icon="i-lucide-building-2" title="Belum menjadi anggota company" variant="naked" />
        <ul v-else class="divide-y divide-default rounded-lg border border-default">
          <li v-for="m in user.memberships" :key="m.company_id">
            <NuxtLink :to="`/companies/${m.company_id}/members/${user.id}`" class="flex items-center gap-3 px-4 py-3 hover:bg-elevated/40">
              <UAvatar :text="initials(m.company_name)" size="sm" class="rounded-md" />
              <span class="min-w-0 flex-1">
                <span class="block truncate font-medium text-highlighted">{{ m.company_name }}</span>
                <span class="block truncate text-xs text-muted">{{ m.company_slug }} · bergabung {{ formatDate(m.joined_at) }}</span>
              </span>
              <StatusBadge :status="m.member_status" />
              <UIcon name="i-lucide-chevron-right" class="size-4 text-dimmed" />
            </NuxtLink>
          </li>
        </ul>
      </SectionCard>
    </template>

    <div v-else class="space-y-3">
      <USkeleton class="h-12 w-64" />
      <USkeleton class="h-56 w-full" />
    </div>

    <UModal v-model:open="showAddCompany" title="Tambahkan ke company" :description="user ? `Jadikan ${user.name} anggota company.` : undefined">
      <template #body>
        <div class="space-y-4">
          <UFormField label="Company" required>
            <USelectMenu
              v-model="companyId"
              :items="companyItems"
              value-key="value"
              placeholder="Pilih company"
              :search-input="{ placeholder: 'Cari company...' }"
              class="w-full"
            />
          </UFormField>
          <UFormField label="Status">
            <URadioGroup
              v-model="addStatus"
              orientation="horizontal"
              :items="[{ label: 'Aktif', value: 'active' }, { label: 'Diundang', value: 'invited' }]"
            />
          </UFormField>
        </div>
      </template>
      <template #footer>
        <div class="flex w-full justify-end gap-2">
          <UButton color="neutral" variant="ghost" @click="showAddCompany = false">Batal</UButton>
          <UButton :disabled="!companyId" :loading="addingCompany" @click="addToCompany">Tambahkan</UButton>
        </div>
      </template>
    </UModal>
  </AppPage>
</template>
