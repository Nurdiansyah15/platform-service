<script setup lang="ts">
import type { RoleResponse } from '~/types/api'

definePageMeta({ middleware: 'admin' })

const ctx = useSolutionContext()
const route = useRoute()
const api = useApi()
const toast = useToast()
const { toastError } = useApiError()
const { confirm } = useConfirm()

const roleId = computed(() => String(route.params.roleId))
const base = computed(() => `/admin/solutions/${ctx.solutionId.value}/roles`)
const role = ref<RoleResponse | null>(null)
const error = ref('')
const showEdit = ref(false)

async function load() {
  error.value = ''
  try {
    role.value = (await api.get<RoleResponse>(`/api/v1/roles/${roleId.value}`)).data
  } catch (err) {
    error.value = (err as { message?: string }).message || 'Gagal memuat role'
  }
}
watch(roleId, load, { immediate: true })

async function remove() {
  if (!role.value) return
  const ok = await confirm({
    title: `Hapus role ${role.value.name}?`,
    description: `Role dicabut dari ${role.value.member_count} pemegang di semua company. Tindakan ini tidak bisa dibatalkan.`,
    confirmLabel: 'Hapus role',
    danger: true,
    typeToConfirm: role.value.code,
  })
  if (!ok) return
  try {
    await api.del(`/api/v1/roles/${role.value.id}`)
    toast.add({ title: 'Role dihapus', color: 'success' })
    await navigateTo(base.value)
  } catch (err) {
    toastError(err)
  }
}
</script>

<template>
  <SolutionPage
    :title="role?.name ?? 'Role'"
    :trail="[{ label: 'Role global', to: base }, ...(role ? [{ label: role.name }] : [])]"
  >
    <template #actions>
      <template v-if="role && !role.is_system">
        <UButton icon="i-lucide-pencil" color="neutral" variant="outline" @click="showEdit = true">Ubah</UButton>
        <UButton icon="i-lucide-trash-2" color="error" variant="ghost" aria-label="Hapus role" @click="remove" />
      </template>
    </template>

    <UAlert v-if="error" color="error" variant="subtle" :title="error" />
    <template v-else-if="role">
      <div class="space-y-1">
        <div class="flex flex-wrap items-center gap-2">
          <h1 class="text-xl font-semibold text-highlighted">{{ role.name }}</h1>
          <UBadge v-if="role.is_system" variant="subtle" color="warning">sistem — tidak bisa diubah</UBadge>
        </div>
        <p class="font-mono text-xs text-muted">{{ role.code }} · {{ role.member_count }} pemegang di semua company</p>
        <p v-if="role.description" class="text-sm text-muted">{{ role.description }}</p>
      </div>

      <RolePermissionsEditor
        :role="role"
        :save-path="role.is_system ? undefined : `/api/v1/roles/${role.id}/permissions`"
        @saved="(n: number) => role && (role.permission_count = n)"
      />
    </template>
    <USkeleton v-else class="h-48 w-full" />

    <RoleFormModal
      v-if="role"
      v-model:open="showEdit"
      :role="role"
      create-path=""
      :update-path="(id: string) => `/api/v1/roles/${id}`"
      @saved="(r: RoleResponse) => { role = r; toast.add({ title: 'Role diperbarui', color: 'success' }) }"
    />
  </SolutionPage>
</template>
