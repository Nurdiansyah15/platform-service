<script setup lang="ts">
import type { CompanyResponse, CompanyStatus, UpdateCompanyRequest } from '~/types/api'

const route = useRoute()
const auth = useAuthStore()
const api = useApi()
const toast = useToast()
const companyId = computed(() => String(route.params.companyId))

const company = ref<CompanyResponse | null>(null)
const loading = ref(false)
const error = ref('')

const editing = ref(false)
const saving = ref(false)
const form = reactive<{ name: string; slug: string; status: CompanyStatus }>({ name: '', slug: '', status: 'active' })
const companyStatusItems = [
  { label: 'Active', value: 'active' },
  { label: 'Suspended', value: 'suspended' },
  { label: 'Closed', value: 'closed' },
]

function startEdit() {
  if (!company.value) return
  form.name = company.value.name
  form.slug = company.value.slug
  form.status = company.value.status as CompanyStatus
  editing.value = true
}

async function save() {
  if (!company.value) return
  saving.value = true
  try {
    // Kirim hanya field yang berubah; status hanya boleh diubah platform admin.
    const body: UpdateCompanyRequest = {}
    if (form.name !== company.value.name) body.name = form.name
    if (form.slug !== company.value.slug) body.slug = form.slug
    if (auth.isPlatformAdmin && form.status !== company.value.status) body.status = form.status
    const res = await api.patch<CompanyResponse>(`/api/v1/companies/${companyId.value}`, body)
    company.value = res.data
    editing.value = false
    toast.add({ title: 'Company diperbarui', color: 'success' })
    if (!auth.isPlatformAdmin) await auth.loadCompanies()
  } catch (err) {
    toast.add({ title: (err as { message?: string }).message || 'Gagal memperbarui company', color: 'error' })
  } finally {
    saving.value = false
  }
}

async function load() {
  loading.value = true
  error.value = ''
  try {
    const res = await api.get<CompanyResponse>(`/api/v1/companies/${companyId.value}`)
    company.value = res.data
  } catch (err) {
    error.value = (err as { message?: string }).message || 'Gagal memuat company'
  } finally {
    loading.value = false
  }
}

const links = computed(() => [
  { to: `/companies/${companyId.value}/members`, label: 'Members', icon: 'i-lucide-users', desc: 'Anggota company dan statusnya.' },
  { to: `/companies/${companyId.value}/branches`, label: 'Branches', icon: 'i-lucide-git-branch', desc: 'Unit organisasi di company.' },
  { to: `/companies/${companyId.value}/roles`, label: 'Custom Roles', icon: 'i-lucide-shield', desc: 'Role khusus untuk company ini.' },
])

onMounted(load)
</script>

<template>
  <div>
    <PageHeader :title="company?.name ?? 'Company'" :subtitle="company?.slug">
      <template #actions>
        <UButton to="/companies" color="neutral" variant="outline" icon="i-lucide-arrow-left">Kembali</UButton>
        <UButton v-if="company && !editing" color="primary" icon="i-lucide-pencil" @click="startEdit">Edit</UButton>
      </template>
    </PageHeader>

    <UAlert v-if="error" color="error" variant="subtle" :title="error" class="mb-4" />

    <UCard v-if="company && editing" class="mb-6">
      <form class="grid gap-4 sm:grid-cols-4" @submit.prevent="save">
        <UFormField label="Nama" required>
          <UInput v-model="form.name" class="w-full" />
        </UFormField>
        <UFormField label="Slug" required>
          <UInput v-model="form.slug" class="w-full" />
        </UFormField>
        <UFormField label="Status" :hint="auth.isPlatformAdmin ? undefined : 'khusus admin platform'">
          <USelect v-model="form.status" :items="companyStatusItems" :disabled="!auth.isPlatformAdmin" class="w-full" />
        </UFormField>
        <div class="flex items-end gap-2">
          <UButton type="submit" :loading="saving" color="primary">Simpan</UButton>
          <UButton color="neutral" variant="ghost" @click="editing = false">Batal</UButton>
        </div>
      </form>
    </UCard>

    <UCard v-if="company" class="mb-6">
      <div class="grid gap-4 sm:grid-cols-3">
        <div>
          <p class="text-xs uppercase tracking-wide text-dimmed">ID</p>
          <p class="mt-1 break-all font-mono text-xs text-muted">{{ company.id }}</p>
        </div>
        <div>
          <p class="text-xs uppercase tracking-wide text-dimmed">Status</p>
          <p class="mt-1">
            <UBadge
              size="sm"
              variant="subtle"
              :color="company.status === 'active' ? 'success' : company.status === 'closed' ? 'error' : 'warning'"
            >
              {{ company.status }}
            </UBadge>
          </p>
        </div>
        <div>
          <p class="text-xs uppercase tracking-wide text-dimmed">Dibuat</p>
          <p class="mt-1 text-sm text-muted">{{ new Date(company.created_at).toLocaleString() }}</p>
        </div>
      </div>
    </UCard>

    <div class="grid gap-4 sm:grid-cols-3">
      <NuxtLink
        v-for="link in links"
        :key="link.to"
        :to="link.to"
        class="rounded-lg border border-default bg-default p-4 transition hover:border-primary/40"
      >
        <div class="flex items-center gap-3">
          <UIcon :name="link.icon" class="size-5 text-primary" />
          <span class="font-medium text-highlighted">{{ link.label }}</span>
        </div>
        <p class="mt-2 text-sm text-muted">{{ link.desc }}</p>
      </NuxtLink>
    </div>
  </div>
</template>