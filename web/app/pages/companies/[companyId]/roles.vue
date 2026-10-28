<script setup lang="ts">
import type { RoleResponse } from '~/types/api'

const route = useRoute()
const auth = useAuthStore()
const api = useApi()
const toast = useToast()

const companyId = computed(() => String(route.params.companyId))
const solutionId = computed(() => auth.solution?.id ?? '')
const isForeignCompany = computed(
  () => !auth.isPlatformAdmin && auth.activeCompanyId !== companyId.value,
)

const items = ref<RoleResponse[]>([])
const loading = ref(false)
const error = ref('')

const showCreate = ref(false)
const saving = ref(false)
const form = reactive({ code: '', name: '', description: '' })

async function load() {
  if (!solutionId.value || isForeignCompany.value) return
  loading.value = true
  error.value = ''
  try {
    const res = await api.get<RoleResponse[]>(
      `/api/v1/companies/${companyId.value}/solutions/${solutionId.value}/roles`,
    )
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
    await api.post(`/api/v1/companies/${companyId.value}/solutions/${solutionId.value}/roles`, {
      code: form.code,
      name: form.name,
      description: form.description || null,
    })
    toast.add({ title: 'Role dibuat', color: 'success' })
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
    <PageHeader title="Custom Roles" subtitle="Role khusus company ini (di atas role global solution).">
      <template #actions>
        <UButton
          icon="i-lucide-plus"
          color="primary"
          :disabled="isForeignCompany || !solutionId"
          @click="showCreate = !showCreate"
        >
          Role baru
        </UButton>
      </template>
    </PageHeader>

    <UAlert
      v-if="isForeignCompany"
      class="mb-4"
      color="warning"
      variant="subtle"
      title="Company ini bukan company aktif"
      description="Aktifkan company ini dulu untuk mengelola role."
    />
    <UAlert
      v-else-if="!solutionId"
      class="mb-4"
      color="warning"
      variant="subtle"
      title="Solution sesi tidak diketahui"
    />

    <UCard v-if="showCreate && !isForeignCompany && solutionId" class="mb-4">
      <form class="grid gap-4 sm:grid-cols-3" @submit.prevent="createRole">
        <UFormField label="Kode">
          <UInput v-model="form.code" class="w-full" placeholder="branch_manager" />
        </UFormField>
        <UFormField label="Nama">
          <UInput v-model="form.name" class="w-full" placeholder="Branch Manager" />
        </UFormField>
        <UFormField label="Deskripsi">
          <UInput v-model="form.description" class="w-full" placeholder="opsional" />
        </UFormField>
        <div class="sm:col-span-3">
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
              <th class="px-3 py-2">Scope</th>
              <th class="px-3 py-2"></th>
            </tr>
          </thead>
          <tbody>
            <tr v-if="loading">
              <td colspan="4" class="px-3 py-6 text-center text-muted">Memuat...</td>
            </tr>
            <tr v-else-if="!items.length">
              <td colspan="4" class="px-3 py-6 text-center text-muted">Belum ada role.</td>
            </tr>
            <template v-else>
              <tr v-for="r in items" :key="r.id" class="border-b border-default/60">
                <td class="px-3 py-2 font-mono text-xs text-highlighted">{{ r.code }}</td>
                <td class="px-3 py-2 text-highlighted">{{ r.name }}</td>
                <td class="px-3 py-2">
                  <UBadge size="sm" variant="subtle" :color="r.scope === 'company' ? 'primary' : 'neutral'">
                    {{ r.scope }}
                  </UBadge>
                </td>
                <td class="px-3 py-2 text-right">
                  <UButton
                    size="xs"
                    color="neutral"
                    variant="outline"
                    :to="`/roles/${r.id}/permissions?companyId=${companyId}&solutionId=${solutionId}`"
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