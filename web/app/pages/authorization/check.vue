<script setup lang="ts">
import type {
  AuthzCheckBatchResponse,
  CreateRoleAssignmentRequest,
  RoleAssignmentResponse,
  RoleResponse,
} from '~/types/api'

const auth = useAuthStore()
const api = useApi()
const toast = useToast()

// ── Permission check ───────────────────────────────────────────────────────
const permissionsInput = ref('')
const checking = ref(false)
const results = ref<AuthzCheckBatchResponse | null>(null)
const checkError = ref('')

const canCheckSelf = computed(() => auth.isPlatformAdmin || !!auth.activeCompanyId)

async function runCheck() {
  checkError.value = ''
  results.value = null
  const permissions = permissionsInput.value
    .split(/[\n,]+/)
    .map(s => s.trim())
    .filter(Boolean)
  if (!permissions.length) {
    checkError.value = 'Isi minimal satu kode permission.'
    return
  }
  checking.value = true
  try {
    const res = await api.post<AuthzCheckBatchResponse>('/api/v1/authz/check-batch', {
      checks: permissions.map(permission => ({ permission })),
    })
    results.value = res.data
  } catch (err) {
    checkError.value = (err as { message?: string }).message || 'Gagal mengecek permission'
  } finally {
    checking.value = false
  }
}

// ── Role assignment ────────────────────────────────────────────────────────
const roles = ref<RoleResponse[]>([])
const loadingRoles = ref(false)
const assigning = ref(false)
const assignment = reactive<CreateRoleAssignmentRequest>({
  role_id: '',
  scope_type: 'company',
  scope_id: '',
})
const assignUserId = ref('')
const assignError = ref('')

const roleItems = computed(() =>
  roles.value.map(r => ({ label: `${r.name} (${r.code})`, value: r.id })),
)

async function loadRoles() {
  if (!auth.activeCompanyId || !auth.solution) return
  loadingRoles.value = true
  try {
    const res = await api.get<RoleResponse[]>(
      `/api/v1/companies/${auth.activeCompanyId}/solutions/${auth.solution.id}/roles`,
    )
    roles.value = res.data
  } catch {
    roles.value = []
  } finally {
    loadingRoles.value = false
  }
}

async function submitAssignment() {
  assignError.value = ''
  if (!auth.activeCompanyId) {
    assignError.value = 'Butuh company aktif untuk assignment.'
    return
  }
  assigning.value = true
  try {
    await api.post<RoleAssignmentResponse>(
      `/api/v1/companies/${auth.activeCompanyId}/users/${assignUserId.value}/role-assignments`,
      {
        role_id: assignment.role_id,
        scope_type: assignment.scope_type,
        scope_id: assignment.scope_type === 'company'
          ? (assignment.scope_id || auth.activeCompanyId)
          : assignment.scope_id,
      },
    )
    toast.add({ title: 'Role berhasil di-assign', color: 'success' })
    assignment.role_id = ''
    assignment.scope_id = ''
    assignUserId.value = ''
  } catch (err) {
    assignError.value = (err as { message?: string }).message || 'Gagal assign role'
  } finally {
    assigning.value = false
  }
}

onMounted(loadRoles)
</script>

<template>
  <div>
    <PageHeader
      title="Authorization"
      subtitle="Uji permission dan assign role pada company aktif."
    />

    <UAlert
      v-if="!canCheckSelf"
      class="mb-4"
      color="warning"
      variant="subtle"
      icon="i-lucide-triangle-alert"
      title="Company aktif belum dipilih"
      description="Uji permission sebagai user membutuhkan company aktif. Pilih company lewat switcher di atas."
    />

    <div class="grid gap-6 lg:grid-cols-2">
      <UCard>
        <h2 class="mb-1 font-medium text-highlighted">Cek permission</h2>
        <p class="mb-4 text-sm text-muted">
          Evaluasi permission untuk dirimu pada company aktif (authz/check-batch).
        </p>
        <form class="space-y-3" @submit.prevent="runCheck">
          <UFormField label="Permission (pisahkan koma / baris baru)">
            <UTextarea
              v-model="permissionsInput"
              :rows="3"
              class="w-full"
              placeholder="org.read, roles.write"
            />
          </UFormField>
          <UAlert v-if="checkError" color="error" variant="subtle" :title="checkError" />
          <UButton type="submit" color="primary" :loading="checking" :disabled="!canCheckSelf">
            Cek
          </UButton>
        </form>

        <div v-if="results" class="mt-4 overflow-x-auto">
          <table class="w-full text-sm">
            <thead>
              <tr class="border-b border-default text-left text-xs uppercase tracking-wide text-dimmed">
                <th class="px-2 py-2">Permission</th>
                <th class="px-2 py-2">Allowed</th>
                <th class="px-2 py-2">Granted by</th>
              </tr>
            </thead>
            <tbody>
              <tr v-for="r in results.results" :key="r.permission" class="border-b border-default/60">
                <td class="px-2 py-2 font-mono text-xs text-highlighted">{{ r.permission }}</td>
                <td class="px-2 py-2">
                  <UBadge size="sm" variant="subtle" :color="r.allowed ? 'success' : 'error'">
                    {{ r.allowed ? 'yes' : 'no' }}
                  </UBadge>
                </td>
                <td class="px-2 py-2 text-xs text-muted">{{ (r.granted_by || []).join(', ') || '-' }}</td>
              </tr>
            </tbody>
          </table>
        </div>
      </UCard>

      <UCard>
        <h2 class="mb-1 font-medium text-highlighted">Assign role</h2>
        <p class="mb-4 text-sm text-muted">
          Pasang role ke user member company aktif. API belum menyediakan daftar assignment.
        </p>
        <form class="space-y-3" @submit.prevent="submitAssignment">
          <UFormField label="User ID">
            <UInput v-model="assignUserId" class="w-full" placeholder="uuid user (lihat Members)" />
          </UFormField>
          <UFormField label="Role">
            <USelect
              v-model="assignment.role_id"
              :items="roleItems"
              :loading="loadingRoles"
              class="w-full"
              placeholder="Pilih role"
            />
          </UFormField>
          <div class="grid grid-cols-2 gap-3">
            <UFormField label="Scope type">
              <USelect
                v-model="assignment.scope_type"
                :items="[{ label: 'Company', value: 'company' }, { label: 'Branch', value: 'branch' }]"
                class="w-full"
              />
            </UFormField>
            <UFormField label="Scope ID" hint="kosong = company">
              <UInput v-model="assignment.scope_id" class="w-full" placeholder="uuid branch" />
            </UFormField>
          </div>
          <UAlert v-if="assignError" color="error" variant="subtle" :title="assignError" />
          <UButton type="submit" color="primary" :loading="assigning" :disabled="!auth.activeCompanyId">
            Assign
          </UButton>
        </form>
      </UCard>
    </div>
  </div>
</template>