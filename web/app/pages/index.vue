<script setup lang="ts">
import type { AdminSummary } from '~/types/api'

// Admin: ringkasan platform. User lain langsung ke ringkasan company aktif.
const auth = useAuthStore()
const api = useApi()
const summary = ref<AdminSummary | null>(null)
const error = ref('')

if (!auth.isPlatformAdmin && auth.activeCompanyId) {
  await navigateTo(`/companies/${auth.activeCompanyId}`, { replace: true })
}

onMounted(async () => {
  if (!auth.isPlatformAdmin) return
  try {
    summary.value = (await api.get<AdminSummary>('/api/v1/admin/summary')).data
  } catch (err) {
    error.value = (err as { message?: string }).message || 'Gagal memuat ringkasan'
  }
})

const shortcuts = [
  { title: 'Buat user', description: 'Akun baru untuk karyawan pelanggan.', icon: 'i-lucide-user-plus', to: '/admin/users?create=1' },
  { title: 'Kelola company', description: 'Lihat dan buka company mana pun.', icon: 'i-lucide-building-2', to: '/admin/companies' },
  { title: 'Kelola solution', description: 'Role global, permission, dan client.', icon: 'i-lucide-boxes', to: '/admin/solutions' },
]
</script>

<template>
  <AppPage :title="`Halo, ${auth.user?.name ?? ''}`" description="Ringkasan platform.">
    <UAlert v-if="error" color="error" variant="subtle" :title="error" />

    <div class="grid gap-4 sm:grid-cols-2 xl:grid-cols-4">
      <StatCard label="User aktif" icon="i-lucide-users" :value="summary?.users.active" to="/admin/users"
        :hint="summary ? `${summary.users.suspended} ditangguhkan · ${summary.users.admins} admin` : undefined" />
      <StatCard label="Company aktif" icon="i-lucide-building-2" :value="summary?.companies.active" to="/admin/companies"
        :hint="summary ? `${summary.companies.suspended} ditangguhkan · ${summary.companies.closed} ditutup` : undefined" />
      <StatCard label="Solution aktif" icon="i-lucide-boxes" :value="summary?.solutions.active" to="/admin/solutions"
        :hint="summary ? `${summary.solutions.inactive} nonaktif` : undefined" />
      <StatCard label="Client aktif" icon="i-lucide-plug" :value="summary?.active_clients" hint="Kredensial service backend" />
    </div>

    <div>
      <h2 class="mb-3 text-sm font-semibold text-highlighted">Pintasan</h2>
      <div class="grid gap-4 sm:grid-cols-3">
        <UPageCard v-for="s in shortcuts" :key="s.to" v-bind="s" variant="outline" spotlight />
      </div>
    </div>

    <UPageCard
      v-if="auth.activeCompanyId"
      :title="auth.activeCompany?.name ?? 'Company aktif'"
      description="Kelola anggota, cabang, dan role company aktifmu."
      icon="i-lucide-building"
      :to="`/companies/${auth.activeCompanyId}`"
      variant="subtle"
    />
  </AppPage>
</template>
