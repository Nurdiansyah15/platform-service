<script setup lang="ts">
import type { RoleResponse } from '~/types/api'

const route = useRoute()
const auth = useAuthStore()
const api = useApi()
const toast = useToast()

const solutionId = computed(() => String(route.params.solutionId))

const items = ref<RoleResponse[]>([])
const loading = ref(false)
const error = ref('')

const showCreate = ref(false)
const saving = ref(false)
const form = reactive({ code: '', name: '', description: '' })

async function load() {
  loading.value = true
  error.value = ''
  try {
    const res = await api.get<RoleResponse[]>(`/api/v1/solutions/${solutionId.value}/roles`)
    items.value = res.data
  } catch (err) {
    error.value = (err as { message?: string }).message || 'Gagal memuat role'
  } finally {
    loading.value = false
  }
}

async function createRole() {
  saving.value = true
  try {
    await api.post(`/api/v1/solutions/${solutionId.value}/roles`, {
      code: form.code,
      name: form.name,
      description: form.description || null,
    })
    toast.add({ title: 'Role global dibuat', color: 'success' })
    showCreate.value = false
    form.code = ''
    form.name = ''
    form.description = ''
    await load()
  } catch (err) {
    toast.add({ title: (err as { message?: string }).message || 'Gagal membuat role', color: 'error' })
  } finally {
    saving.value = false
  }
}

onMounted(load)
</script>

<template>
  <div>
    <PageHeader title="Global Roles" subtitle="Role global milik solution ini.">
      <template #actions>
        <UButton v-if="auth.isPlatformAdmin" icon="i-lucide-plus" color="primary" @click="showCreate = !showCreate">
          Role global
        </UButton>
      </template>
    </PageHeader>

    <UCard v-if="showCreate && auth.isPlatformAdmin" class="mb-4">
      <form class="grid gap-4 sm:grid-cols-4" @submit.prevent="createRole">
        <UFormField label="Kode">
          <UInput v-model="form.code" class="w-full" placeholder="admin" />
        </UFormField>
        <UFormField label="Nama">
          <UInput v-model="form.name" class="w-full" placeholder="Administrator" />
        </UFormField>
        <UFormField label="Deskripsi">
          <UInput v-model="form.description" class="w-full" placeholder="opsional" />
        </UFormField>
        <div class="flex items-end">
          <UButton type="submit" :loading="saving" color="primary">Simpan</UButton>
        </div>
      </form>
    </UCard>

    <UCard>
      <UAlert v-if="error" color="error" variant="subtle" :title="error" class="mb-4" />

      <div class="overflow-x-auto">
        <table class="w-full text-sm">
          <thead>
            <tr class="border-b border-default text-left text-xs uppercase tracking-wide text-dimmed">
              <th class="px-3 py-2">Kode</th>
              <th class="px-3 py-2">Nama</th>
              <th class="px-3 py-2">Sistem</th>
              <th class="px-3 py-2"></th>
            </tr>
          </thead>
          <tbody>
            <tr v-if="loading">
              <td colspan="4" class="px-3 py-6 text-center text-muted">Memuat...</td>
            </tr>
            <tr v-else-if="!items.length">
              <td colspan="4" class="px-3 py-6 text-center text-muted">Belum ada role global.</td>
            </tr>
            <template v-else>
              <tr v-for="r in items" :key="r.id" class="border-b border-default/60">
                <td class="px-3 py-2 font-mono text-xs text-highlighted">{{ r.code }}</td>
                <td class="px-3 py-2 text-highlighted">{{ r.name }}</td>
                <td class="px-3 py-2">
                  <UBadge size="sm" variant="subtle" :color="r.is_system ? 'warning' : 'neutral'">
                    {{ r.is_system ? 'system' : 'custom' }}
                  </UBadge>
                </td>
                <td class="px-3 py-2 text-right">
                  <UButton
                    size="xs"
                    color="neutral"
                    variant="outline"
                    :to="`/roles/${r.id}/permissions?solutionId=${solutionId}`"
                  >
                    Permission
                  </UButton>
                </td>
              </tr>
            </template>
          </tbody>
        </table>
      </div>
    </UCard>
  </div>
</template>