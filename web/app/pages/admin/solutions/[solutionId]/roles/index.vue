<script setup lang="ts">
import type { RoleResponse } from '~/types/api'

definePageMeta({ middleware: 'admin' })

const ctx = useSolutionContext()
const api = useApi()
const toast = useToast()
const { toastError } = useApiError()

const roles = ref<RoleResponse[]>([])
const loading = ref(false)
const search = ref('')
const showCreate = ref(false)
const base = computed(() => `/admin/solutions/${ctx.solutionId.value}/roles`)

const filtered = computed(() => {
  const q = search.value.trim().toLowerCase()
  return roles.value.filter(r => !q || r.name.toLowerCase().includes(q) || r.code.includes(q))
})

async function load() {
  loading.value = true
  try {
    roles.value = (await api.get<RoleResponse[]>(`/api/v1/solutions/${ctx.solutionId.value}/roles`, { query: { limit: 200 } })).data
  } catch (err) {
    toastError(err, 'Gagal memuat role')
  } finally {
    loading.value = false
  }
}
watch(ctx.solutionId, load, { immediate: true })

function onCreated(role: RoleResponse) {
  toast.add({ title: `Role ${role.name} dibuat`, description: 'Atur permission-nya sekarang.', color: 'success' })
  navigateTo(`${base.value}/${role.id}`)
}
</script>

<template>
  <SolutionPage title="Role global" :trail="[{ label: 'Role global' }]">
    <template #actions>
      <UButton icon="i-lucide-plus" @click="showCreate = true">Buat role global</UButton>
    </template>

    <p class="text-sm text-muted">Role global tersedia untuk semua company yang memakai solution ini.</p>
    <UInput v-model="search" icon="i-lucide-search" placeholder="Cari role" class="w-full sm:w-72" />

    <div v-if="loading && !roles.length" class="grid gap-4 sm:grid-cols-2 xl:grid-cols-3">
      <USkeleton v-for="i in 3" :key="i" class="h-32 w-full" />
    </div>
    <UEmpty
      v-else-if="!filtered.length"
      icon="i-lucide-shield"
      :title="roles.length ? 'Tidak ada role yang cocok' : 'Belum ada role global'"
      :actions="roles.length ? [] : [{ label: 'Buat role global', icon: 'i-lucide-plus', onClick: () => (showCreate = true) }]"
    />
    <div v-else class="grid gap-4 sm:grid-cols-2 xl:grid-cols-3">
      <UPageCard v-for="r in filtered" :key="r.id" :to="`${base}/${r.id}`" variant="outline" spotlight :ui="{ container: 'gap-y-2' }">
        <div class="flex items-start justify-between gap-2">
          <div class="min-w-0">
            <p class="truncate font-semibold text-highlighted">{{ r.name }}</p>
            <p class="truncate font-mono text-xs text-muted">{{ r.code }}</p>
          </div>
          <UBadge v-if="r.is_system" size="sm" variant="subtle" color="warning">sistem</UBadge>
        </div>
        <p class="line-clamp-2 min-h-10 text-sm text-muted">{{ r.description || 'Tanpa deskripsi' }}</p>
        <div class="flex gap-4 text-xs text-muted">
          <span class="flex items-center gap-1"><UIcon name="i-lucide-key-round" class="size-3.5" />{{ r.permission_count }} permission</span>
          <span class="flex items-center gap-1"><UIcon name="i-lucide-users" class="size-3.5" />{{ r.member_count }} pemegang</span>
        </div>
      </UPageCard>
    </div>

    <RoleFormModal
      v-model:open="showCreate"
      :create-path="`/api/v1/solutions/${ctx.solutionId.value}/roles`"
      :permissions-path="(id: string) => `/api/v1/roles/${id}/permissions`"
      :templates="roles"
      @saved="onCreated"
    />
  </SolutionPage>
</template>
