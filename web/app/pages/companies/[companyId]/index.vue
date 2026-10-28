<script setup lang="ts">
import type { CompanyResponse } from '~/types/api'

const route = useRoute()
const api = useApi()
const companyId = computed(() => String(route.params.companyId))

const company = ref<CompanyResponse | null>(null)
const loading = ref(false)
const error = ref('')

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
      </template>
    </PageHeader>

    <UAlert v-if="error" color="error" variant="subtle" :title="error" class="mb-4" />

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