<script setup lang="ts">
import type { CompanySummary } from '~/types/api'

const ctx = useCompanyContext()
const api = useApi()
const summary = ref<CompanySummary | null>(null)

const base = computed(() => `/companies/${ctx.companyId.value}`)

watch([ctx.companyId, ctx.canAccess], async ([id, ok]) => {
  summary.value = null
  if (!id || !ok) return
  try {
    summary.value = (await api.get<CompanySummary>(`/api/v1/companies/${id}/summary`)).data
  } catch {
    // kartu tetap tampil sebagai skeleton; error utama ditangani CompanyPage
  }
}, { immediate: true })

const actions = computed(() => [
  { title: 'Tambah anggota', description: 'Cari user terdaftar dan tambahkan ke company.', icon: 'i-lucide-user-plus', to: `${base.value}/members?add=1` },
  { title: 'Atur role', description: 'Buat custom role dan pilih permission-nya.', icon: 'i-lucide-shield-plus', to: `${base.value}/roles` },
  { title: 'Tambah cabang', description: 'Unit organisasi untuk membatasi scope role.', icon: 'i-lucide-git-branch-plus', to: `${base.value}/branches?add=1` },
])
</script>

<template>
  <CompanyPage title="Ringkasan">
    <div class="flex flex-wrap items-center gap-3">
      <UAvatar :text="initials(ctx.company.value?.name)" size="xl" class="rounded-lg" />
      <div class="min-w-0">
        <h1 class="truncate text-xl font-semibold text-highlighted">{{ ctx.company.value?.name }}</h1>
        <p class="flex items-center gap-2 text-sm text-muted">
          <span>{{ ctx.company.value?.slug }}</span>
          <StatusBadge :status="ctx.company.value?.status" />
        </p>
      </div>
    </div>

    <div class="grid gap-4 sm:grid-cols-2 xl:grid-cols-4">
      <StatCard label="Anggota aktif" icon="i-lucide-users" :value="summary?.members.active" :to="`${base}/members`"
        :hint="summary && summary.members.invited ? `${summary.members.invited} menunggu (diundang)` : undefined" />
      <StatCard label="Cabang" icon="i-lucide-git-branch" :value="summary?.branches" :to="`${base}/branches`" />
      <StatCard label="Custom role" icon="i-lucide-shield" :value="summary?.custom_roles" :to="`${base}/roles`" />
      <StatCard label="Penugasan role" icon="i-lucide-badge-check" :value="summary?.role_assignments" hint="Total role yang di-assign ke anggota" />
    </div>

    <div>
      <h2 class="mb-3 text-sm font-semibold text-highlighted">Mulai dari sini</h2>
      <div class="grid gap-4 sm:grid-cols-3">
        <UPageCard v-for="a in actions" :key="a.to" v-bind="a" variant="outline" spotlight />
      </div>
    </div>
  </CompanyPage>
</template>
