<script setup lang="ts">
import type { RoleResponse } from '~/types/api'

const ctx = useCompanyContext()
const perms = useCompanyPermissions()
const api = useApi()
const toast = useToast()
const { toastError } = useApiError()

const solutionId = ref<string | undefined>()
const kind = ref<'all' | 'global' | 'custom'>('all')
const search = ref('')
const roles = ref<RoleResponse[]>([])
const loading = ref(false)
const showCreate = ref(false)

const base = computed(() => `/companies/${ctx.companyId.value}/roles`)
// Role custom solution console (berisi permission platform) hanya untuk Admin company.
const canCreate = computed(() => perms.can('role.manage') && (!perms.isConsoleSolution(solutionId.value) || perms.isCompanyAdmin.value))
const kindItems = [
  { label: 'Semua', value: 'all' },
  { label: 'Global', value: 'global' },
  { label: 'Custom', value: 'custom' },
]

const filtered = computed(() => {
  const q = search.value.trim().toLowerCase()
  return roles.value.filter(r =>
    (kind.value === 'all' || (kind.value === 'custom') === (r.scope === 'company'))
    && (!q || r.name.toLowerCase().includes(q) || r.code.includes(q)),
  )
})
const globalRoles = computed(() => roles.value.filter(r => r.scope !== 'company'))

async function load() {
  if (!solutionId.value || !ctx.canAccess.value) return
  loading.value = true
  try {
    roles.value = (await api.get<RoleResponse[]>(
      `/api/v1/companies/${ctx.companyId.value}/solutions/${solutionId.value}/roles`, { query: { limit: 200 } },
    )).data
  } catch (err) {
    toastError(err, 'Gagal memuat role')
  } finally {
    loading.value = false
  }
}

watch([solutionId, () => ctx.canAccess.value], load)

function onCreated(role: RoleResponse) {
  toast.add({ title: `Role ${role.name} dibuat`, description: 'Atur permission-nya sekarang.', color: 'success' })
  navigateTo({ path: `${base.value}/${role.id}`, query: { solution: solutionId.value } })
}
</script>

<template>
  <CompanyPage title="Role" :section="{ label: 'Role', to: base }">
    <template #actions>
      <UButton v-if="canCreate" icon="i-lucide-plus" :disabled="!solutionId" @click="showCreate = true">Buat custom role</UButton>
    </template>

    <p class="text-sm text-muted">
      Role global disediakan solution untuk semua company. Buat <strong class="text-highlighted">custom role</strong>
      bila company-mu butuh kombinasi permission sendiri.
    </p>

    <div class="flex flex-wrap items-center gap-2">
      <SolutionSelect v-model="solutionId" />
      <UInput v-model="search" icon="i-lucide-search" placeholder="Cari role" class="w-full sm:w-56" />
      <UTabs v-model="kind" :items="kindItems" :content="false" size="sm" class="w-full sm:w-auto" />
    </div>

    <div v-if="loading && !roles.length" class="grid gap-4 sm:grid-cols-2 xl:grid-cols-3">
      <USkeleton v-for="i in 3" :key="i" class="h-32 w-full" />
    </div>
    <UEmpty
      v-else-if="!filtered.length"
      icon="i-lucide-shield"
      :title="roles.length ? 'Tidak ada role yang cocok' : 'Belum ada role'"
      :description="roles.length ? 'Coba ubah pencarian atau filter.' : 'Solution ini belum punya role. Buat custom role pertama.'"
      :actions="roles.length || !canCreate ? [] : [{ label: 'Buat custom role', icon: 'i-lucide-plus', onClick: () => (showCreate = true) }]"
    />
    <div v-else class="grid gap-4 sm:grid-cols-2 xl:grid-cols-3">
      <UPageCard
        v-for="r in filtered"
        :key="r.id"
        :to="{ path: `${base}/${r.id}`, query: { solution: solutionId } }"
        variant="outline"
        spotlight
        :ui="{ container: 'gap-y-2' }"
      >
        <div class="flex items-start justify-between gap-2">
          <div class="min-w-0">
            <p class="truncate font-semibold text-highlighted">{{ r.name }}</p>
            <p class="truncate font-mono text-xs text-muted">{{ r.code }}</p>
          </div>
          <UBadge size="sm" variant="subtle" :color="r.scope === 'company' ? 'primary' : 'neutral'">
            {{ r.scope === 'company' ? 'custom' : r.scope === 'system' ? 'sistem' : 'global' }}
          </UBadge>
        </div>
        <p class="line-clamp-2 min-h-10 text-sm text-muted">{{ r.description || 'Tanpa deskripsi' }}</p>
        <div class="flex gap-4 text-xs text-muted">
          <span class="flex items-center gap-1"><UIcon name="i-lucide-key-round" class="size-3.5" />{{ r.permission_count }} permission</span>
          <span class="flex items-center gap-1"><UIcon name="i-lucide-users" class="size-3.5" />{{ r.member_count }} anggota</span>
        </div>
      </UPageCard>
    </div>

    <RoleFormModal
      v-if="solutionId"
      v-model:open="showCreate"
      :create-path="`/api/v1/companies/${ctx.companyId.value}/solutions/${solutionId}/roles`"
      :permissions-path="(id: string) => `/api/v1/companies/${ctx.companyId.value}/roles/${id}/permissions`"
      :templates="globalRoles"
      @saved="onCreated"
    />
  </CompanyPage>
</template>
