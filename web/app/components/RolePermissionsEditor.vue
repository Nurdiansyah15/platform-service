<script setup lang="ts">
import type { PermissionResponse, RolePermissionResponse, RoleResponse } from '~/types/api'

// Editor permission role: memuat katalog solution (semua halaman) + permission terpasang,
// menyimpan dengan replace-all. Tanpa savePath = read-only.
const props = defineProps<{ role: RoleResponse; savePath?: string }>()
const emit = defineEmits<{ saved: [count: number] }>()

const api = useApi()
const toast = useToast()
const { toastError } = useApiError()

const catalog = ref<PermissionResponse[]>([])
const original = ref<string[]>([])
const selected = ref<string[]>([])
const loading = ref(true)
const saving = ref(false)
const readonly = computed(() => !props.savePath)

const changes = computed(() => {
  const added = selected.value.filter(c => !original.value.includes(c)).length
  const removed = original.value.filter(c => !selected.value.includes(c)).length
  return added + removed
})

async function loadCatalog() {
  const all: PermissionResponse[] = []
  for (let page = 1; ; page++) {
    const res = await api.get<PermissionResponse[]>(`/api/v1/solutions/${props.role.solution_id}/permissions`, { query: { page, limit: 200 } })
    all.push(...res.data)
    if (!res.meta || page >= res.meta.total_pages) break
  }
  return all
}

async function load() {
  loading.value = true
  try {
    const [cat, assigned] = await Promise.all([
      loadCatalog(),
      api.get<RolePermissionResponse[]>(`/api/v1/roles/${props.role.id}/permissions`),
    ])
    // Permission usang yang masih terpasang tetap ditampilkan.
    const extra = assigned.data.filter(a => !cat.some(c => c.code === a.code))
      .map(a => ({ id: a.id, code: a.code, name: a.name, status: a.status } as PermissionResponse))
    catalog.value = [...cat, ...extra]
    original.value = assigned.data.map(p => p.code)
    selected.value = [...original.value]
  } catch (err) {
    toastError(err, 'Gagal memuat permission')
  } finally {
    loading.value = false
  }
}

async function save() {
  if (!props.savePath) return
  saving.value = true
  try {
    const res = await api.put<RolePermissionResponse[]>(props.savePath, { permission_codes: selected.value })
    original.value = res.data.map(p => p.code)
    selected.value = [...original.value]
    toast.add({ title: 'Permission disimpan', color: 'success' })
    emit('saved', original.value.length)
  } catch (err) {
    toastError(err, 'Gagal menyimpan permission')
  } finally {
    saving.value = false
  }
}

const { confirm } = useConfirm()
onBeforeRouteLeave(async () => {
  if (changes.value === 0) return true
  return confirm({
    title: 'Tinggalkan halaman?',
    description: 'Ada perubahan permission yang belum disimpan dan akan hilang.',
    confirmLabel: 'Tinggalkan',
    danger: true,
  })
})

watch(() => props.role.id, load, { immediate: true })
</script>

<template>
  <div class="space-y-4">
    <div v-if="loading" class="space-y-2">
      <USkeleton v-for="i in 4" :key="i" class="h-10 w-full" />
    </div>
    <template v-else>
      <UAlert
        v-if="readonly"
        color="neutral"
        variant="subtle"
        icon="i-lucide-lock"
        title="Hanya-baca"
        :description="role.scope === 'company' ? undefined : 'Role global dikelola admin platform. Buat custom role bila butuh kombinasi lain.'"
      />
      <PermissionPicker v-model="selected" :catalog="catalog" :readonly="readonly" />

      <div
        v-if="!readonly && changes > 0"
        class="sticky bottom-0 flex flex-wrap items-center justify-between gap-3 rounded-lg border border-default bg-default/95 p-3 shadow-lg backdrop-blur"
      >
        <span class="text-sm text-highlighted">{{ changes }} perubahan belum disimpan · {{ selected.length }} permission dipilih</span>
        <div class="flex gap-2">
          <UButton color="neutral" variant="ghost" @click="selected = [...original]">Batalkan</UButton>
          <UButton :loading="saving" @click="save">Simpan permission</UButton>
        </div>
      </div>
    </template>
  </div>
</template>
