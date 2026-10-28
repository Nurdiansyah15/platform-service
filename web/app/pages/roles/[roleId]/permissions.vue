<script setup lang="ts">
import type { PermissionResponse, RolePermissionResponse } from '~/types/api'

const route = useRoute()
const auth = useAuthStore()
const api = useApi()
const toast = useToast()

const roleId = computed(() => String(route.params.roleId))
const companyId = computed(() => (typeof route.query.companyId === 'string' ? route.query.companyId : ''))
const solutionId = computed(() =>
  typeof route.query.solutionId === 'string' && route.query.solutionId
    ? route.query.solutionId
    : (auth.solution?.id ?? ''),
)

const assigned = ref<RolePermissionResponse[]>([])
const catalog = ref<PermissionResponse[]>([])
const selected = ref<string[]>([])
const loading = ref(false)
const saving = ref(false)
const error = ref('')

const isCompanyRole = computed(() => !!companyId.value)
const foreignCompany = computed(
  () => isCompanyRole.value && !auth.isPlatformAdmin && auth.activeCompanyId !== companyId.value,
)
const canEdit = computed(() => {
  if (foreignCompany.value) return false
  if (isCompanyRole.value) return true
  return auth.isPlatformAdmin
})

async function load() {
  loading.value = true
  error.value = ''
  try {
    const assignedRes = await api.get<RolePermissionResponse[]>(
      `/api/v1/roles/${roleId.value}/permissions`,
    )
    assigned.value = assignedRes.data
    selected.value = assignedRes.data.map(p => p.code)

    if (solutionId.value) {
      const catalogRes = await api.get<PermissionResponse[]>(
        `/api/v1/solutions/${solutionId.value}/permissions`,
        { query: { limit: 100 } },
      )
      catalog.value = catalogRes.data
    }
  } catch (err) {
    error.value = (err as { message?: string }).message || 'Gagal memuat permission role'
  } finally {
    loading.value = false
  }
}

function toggle(code: string) {
  if (!canEdit.value) return
  const idx = selected.value.indexOf(code)
  if (idx >= 0) selected.value.splice(idx, 1)
  else selected.value.push(code)
}

async function save() {
  saving.value = true
  try {
    const path = isCompanyRole.value
      ? `/api/v1/companies/${companyId.value}/roles/${roleId.value}/permissions`
      : `/api/v1/roles/${roleId.value}/permissions`
    await api.put(path, { permission_codes: selected.value })
    toast.add({ title: 'Permission role disimpan', color: 'success' })
    await load()
  } catch (err) {
    toast.add({ title: (err as { message?: string }).message || 'Gagal menyimpan', color: 'error' })
  } finally {
    saving.value = false
  }
}

onMounted(load)
</script>

<template>
  <div>
    <PageHeader title="Role Permissions" subtitle="Pilih permission yang diberikan role ini.">
      <template #actions>
        <UButton color="neutral" variant="outline" icon="i-lucide-arrow-left" @click="$router.back()">
          Kembali
        </UButton>
        <UButton v-if="canEdit" color="primary" :loading="saving" @click="save">Simpan</UButton>
      </template>
    </PageHeader>

    <UAlert
      v-if="foreignCompany"
      class="mb-4"
      color="warning"
      variant="subtle"
      title="Company ini bukan company aktif"
      description="Aktifkan company role ini dulu untuk mengubah permission."
    />
    <UAlert
      v-else-if="!canEdit"
      class="mb-4"
      color="info"
      variant="subtle"
      title="Hanya platform admin yang dapat mengubah role global"
    />

    <UCard>
      <UAlert v-if="error" color="error" variant="subtle" :title="error" class="mb-4" />

      <div v-if="loading" class="py-6 text-center text-sm text-muted">Memuat...</div>
      <div v-else-if="!catalog.length" class="py-6 text-center text-sm text-muted">
        Katalog permission tidak tersedia.
      </div>
      <div v-else class="grid gap-2 sm:grid-cols-2 lg:grid-cols-3">
        <label
          v-for="p in catalog"
          :key="p.id"
          class="flex cursor-pointer items-start gap-3 rounded-md border border-default px-3 py-2"
          :class="canEdit ? 'hover:border-primary/40' : 'opacity-70'"
        >
          <input
            type="checkbox"
            class="mt-1"
            :checked="selected.includes(p.code)"
            :disabled="!canEdit"
            @change="toggle(p.code)"
          >
          <span class="min-w-0">
            <span class="block font-mono text-xs text-highlighted">{{ p.code }}</span>
            <span class="block truncate text-xs text-muted">{{ p.name }}</span>
          </span>
        </label>
      </div>

      <p class="mt-4 text-xs text-dimmed">
        Terpilih: {{ selected.length }} · Katalog ditampilkan maksimal 100 permission.
      </p>
    </UCard>
  </div>
</template>