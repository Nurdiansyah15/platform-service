<script setup lang="ts">
import type { RoleHolder, RoleResponse } from '~/types/api'

const ctx = useCompanyContext()
const route = useRoute()
const api = useApi()
const toast = useToast()
const { toastError } = useApiError()
const { confirm } = useConfirm()
const perms = useCompanyPermissions()

const roleId = computed(() => String(route.params.roleId))
const base = computed(() => `/companies/${ctx.companyId.value}`)
const backTo = computed(() => ({ path: `${base.value}/roles`, query: route.query.solution ? { solution: route.query.solution } : {} }))

const role = ref<RoleResponse | null>(null)
const holders = ref<RoleHolder[]>([])
const error = ref('')
const tab = ref<'permissions' | 'holders'>('permissions')
const showEdit = ref(false)

const isCustom = computed(() => role.value?.scope === 'company')
const canManage = computed(() => isCustom.value && perms.can('role.manage')
  && (!perms.isConsoleSolution(role.value?.solution_id) || perms.isCompanyAdmin.value))
const tabs = computed(() => [
  { label: 'Permission', value: 'permissions', icon: 'i-lucide-key-round', badge: role.value?.permission_count },
  { label: 'Pemegang', value: 'holders', icon: 'i-lucide-users', badge: role.value?.member_count },
])

async function load() {
  error.value = ''
  try {
    const path = `/api/v1/companies/${ctx.companyId.value}/roles/${roleId.value}`
    const [r, h] = await Promise.all([api.get<RoleResponse>(path), api.get<RoleHolder[]>(`${path}/members`)])
    role.value = r.data
    holders.value = h.data
  } catch (err) {
    error.value = (err as { message?: string }).message || 'Gagal memuat role'
  }
}

watch(() => ctx.canAccess.value, ok => ok && load(), { immediate: true })

async function remove() {
  if (!role.value) return
  const ok = await confirm({
    title: `Hapus role ${role.value.name}?`,
    description: `Role dicabut dari ${role.value.member_count} anggota. Tindakan ini tidak bisa dibatalkan.`,
    confirmLabel: 'Hapus role',
    danger: true,
    typeToConfirm: role.value.code,
  })
  if (!ok) return
  try {
    await api.del(`/api/v1/companies/${ctx.companyId.value}/roles/${role.value.id}`)
    toast.add({ title: 'Role dihapus', color: 'success' })
    await navigateTo(backTo.value)
  } catch (err) {
    toastError(err)
  }
}
</script>

<template>
  <CompanyPage
    :title="role?.name ?? 'Role'"
    :section="{ label: 'Role', to: `${base}/roles` }"
    :trail="role ? [{ label: role.name }] : []"
  >
    <template #actions>
      <template v-if="canManage">
        <UButton icon="i-lucide-pencil" color="neutral" variant="outline" @click="showEdit = true">Ubah</UButton>
        <UButton icon="i-lucide-trash-2" color="error" variant="ghost" aria-label="Hapus role" @click="remove" />
      </template>
    </template>

    <UAlert v-if="error" color="error" variant="subtle" :title="error"
      :actions="[{ label: 'Kembali ke daftar role', color: 'neutral', variant: 'outline', to: backTo }]" />

    <template v-else-if="role">
      <div class="space-y-1">
        <div class="flex flex-wrap items-center gap-2">
          <h1 class="text-xl font-semibold text-highlighted">{{ role.name }}</h1>
          <UBadge variant="subtle" :color="isCustom ? 'primary' : 'neutral'">{{ isCustom ? 'custom' : 'global' }}</UBadge>
        </div>
        <p class="font-mono text-xs text-muted">{{ role.code }}</p>
        <p v-if="role.description" class="text-sm text-muted">{{ role.description }}</p>
      </div>

      <UTabs v-model="tab" :items="tabs" :content="false" variant="link" class="w-full" />

      <RolePermissionsEditor
        v-if="tab === 'permissions'"
        :role="role"
        :save-path="canManage ? `/api/v1/companies/${ctx.companyId.value}/roles/${role.id}/permissions` : undefined"
        @saved="(n: number) => role && (role.permission_count = n)"
      />

      <template v-else>
        <UEmpty v-if="!holders.length" icon="i-lucide-users" title="Belum ada pemegang" description="Tambahkan role ini dari halaman detail anggota." variant="naked" />
        <ul v-else class="divide-y divide-default rounded-lg border border-default">
          <li v-for="h in holders" :key="h.assignment_id">
            <NuxtLink :to="`${base}/members/${h.user_id}`" class="flex items-center gap-3 px-4 py-3 hover:bg-elevated/40">
              <UAvatar :text="initials(h.name)" size="sm" />
              <span class="min-w-0 flex-1">
                <span class="block truncate font-medium text-highlighted">{{ h.name }}</span>
                <span class="block truncate text-xs text-muted">{{ h.email }}</span>
              </span>
              <span class="flex items-center gap-1 text-xs text-muted">
                <UIcon :name="h.scope_type === 'company' ? 'i-lucide-building-2' : 'i-lucide-git-branch'" class="size-3.5" />
                {{ h.scope_type === 'company' ? 'Seluruh company' : h.scope_name }}
              </span>
            </NuxtLink>
          </li>
        </ul>
      </template>
    </template>

    <div v-else class="space-y-3">
      <USkeleton class="h-7 w-56" />
      <USkeleton class="h-40 w-full" />
    </div>

    <RoleFormModal
      v-if="role && isCustom"
      v-model:open="showEdit"
      :role="role"
      create-path=""
      :update-path="(id: string) => `/api/v1/companies/${ctx.companyId.value}/roles/${id}`"
      @saved="(r: RoleResponse) => { role = r; toast.add({ title: 'Role diperbarui', color: 'success' }) }"
    />
  </CompanyPage>
</template>
