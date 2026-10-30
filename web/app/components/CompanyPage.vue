<script setup lang="ts">
import type { BreadcrumbItem } from '@nuxt/ui'

// Kerangka halaman di dalam company: judul section, tab navigasi company, breadcrumb,
// dan penjaga akses (member wajib mengaktifkan company dulu).
const props = defineProps<{
  title: string
  /** Breadcrumb tambahan setelah nama company & section. */
  trail?: BreadcrumbItem[]
  section?: { label: string; to: string }
  narrow?: boolean
}>()

const ctx = useCompanyContext()
const auth = useAuthStore()
const perms = useCompanyPermissions()
const toast = useToast()
const { toastError } = useApiError()
const switching = ref(false)

const base = computed(() => `/companies/${ctx.companyId.value}`)
const navItems = computed(() => [
  { label: 'Ringkasan', icon: 'i-lucide-layout-dashboard', to: base.value, exact: true },
  { label: 'Anggota', icon: 'i-lucide-users', to: `${base.value}/members` },
  { label: 'Cabang', icon: 'i-lucide-git-branch', to: `${base.value}/branches` },
  { label: 'Role', icon: 'i-lucide-shield', to: `${base.value}/roles` },
  { label: 'Pengaturan', icon: 'i-lucide-settings', to: `${base.value}/settings` },
])

const breadcrumbs = computed<BreadcrumbItem[]>(() => {
  const items: BreadcrumbItem[] = [
    { label: ctx.company.value?.name ?? 'Company', icon: 'i-lucide-building-2', to: base.value },
  ]
  if (props.section) items.push({ label: props.section.label, to: props.section.to })
  return [...items, ...(props.trail ?? [])]
})

async function activate() {
  switching.value = true
  try {
    await auth.switchCompany(ctx.companyId.value)
    toast.add({ title: 'Company diaktifkan', color: 'success' })
    await ctx.load()
  } catch (err) {
    toastError(err, 'Gagal mengaktifkan company')
  } finally {
    switching.value = false
  }
}

watch(ctx.companyId, () => ctx.load(), { immediate: true })
watch(ctx.canAccess, v => v && ctx.load())
</script>

<template>
  <AppPage :title="title" :breadcrumbs="ctx.canAccess.value && breadcrumbs.length > 1 ? breadcrumbs : undefined" :narrow="narrow">
    <template #actions>
      <slot v-if="ctx.canAccess.value" name="actions" />
    </template>
    <template v-if="ctx.canAccess.value" #toolbar>
      <UNavigationMenu :items="navItems" highlight class="-mx-1 flex-1" />
      <UTooltip
        v-if="perms.readOnly.value"
        text="Kamu tidak punya izin mengelola company ini. Minta Admin company memberimu role."
      >
        <UBadge color="neutral" variant="subtle" icon="i-lucide-eye" class="shrink-0">Hanya lihat</UBadge>
      </UTooltip>
    </template>

    <UEmpty
      v-if="!ctx.canAccess.value"
      icon="i-lucide-building-2"
      :title="ctx.isMemberOf.value ? 'Company ini belum aktif' : 'Tidak ada akses'"
      :description="ctx.isMemberOf.value
        ? 'Aktifkan company ini untuk mengelola anggota, cabang, dan role-nya.'
        : 'Kamu bukan anggota aktif company ini.'"
      :actions="ctx.isMemberOf.value
        ? [{ label: 'Aktifkan company ini', icon: 'i-lucide-check', loading: switching, onClick: activate }]
        : [{ label: 'Kembali ke beranda', color: 'neutral', variant: 'outline', to: '/' }]"
    />
    <UAlert v-else-if="ctx.error.value" color="error" variant="subtle" :title="ctx.error.value" />
    <slot v-else />
  </AppPage>
</template>
