<script setup lang="ts">
import type {
  BranchResponse,
  CompanyMemberResponse,
  CreateRoleAssignmentRequest,
  EffectivePermission,
  RoleAssignmentDetail,
  RoleResponse,
  SolutionResponse,
} from '~/types/api'

const route = useRoute()
const auth = useAuthStore()
const api = useApi()
const toast = useToast()

const companyId = computed(() => String(route.params.companyId))
const userId = computed(() => String(route.params.userId))
const isForeignCompany = computed(
  () => !auth.isPlatformAdmin && auth.activeCompanyId !== companyId.value,
)

const member = ref<CompanyMemberResponse | null>(null)
const error = ref('')

// Role & permission selalu dilihat per solution (bukan solution sesi console).
const solutions = ref<SolutionResponse[]>([])
const solutionId = ref<string | undefined>(undefined)
const solutionItems = computed(() => solutions.value.map(s => ({ label: `${s.name} (${s.code})`, value: s.id })))

const assignments = ref<RoleAssignmentDetail[]>([])
const permissions = ref<EffectivePermission[]>([])
const loadingAccess = ref(false)
const revokingId = ref<string | null>(null)

const roles = ref<RoleResponse[]>([])
const branches = ref<BranchResponse[]>([])
const assigning = ref(false)
const form = reactive<Required<CreateRoleAssignmentRequest>>({ role_id: '', scope_type: 'company', scope_id: '' })

const roleItems = computed(() =>
  roles.value.map(r => ({ label: `${r.name} (${r.code})${r.company_id ? ' · custom' : ''}`, value: r.id })),
)
const branchItems = computed(() => branches.value.map(b => ({ label: `${b.name} (${b.code})`, value: b.id })))

async function loadMember() {
  error.value = ''
  try {
    const res = await api.get<CompanyMemberResponse>(`/api/v1/companies/${companyId.value}/members/${userId.value}`)
    member.value = res.data
  } catch (err) {
    error.value = (err as { message?: string }).message || 'Gagal memuat member'
  }
}

async function loadOptions() {
  try {
    const [sol, br] = await Promise.all([
      api.get<SolutionResponse[]>('/api/v1/solutions', { query: { limit: 100 } }),
      api.get<BranchResponse[]>(`/api/v1/companies/${companyId.value}/branches`, { query: { limit: 100 } }),
    ])
    solutions.value = sol.data
    branches.value = br.data
    // Default: solution pertama selain console (role produk biasanya yang dicari).
    const preferred = sol.data.find(s => s.id !== auth.solution?.id) ?? sol.data[0]
    solutionId.value = preferred?.id
  } catch (err) {
    toast.add({ title: (err as { message?: string }).message || 'Gagal memuat solution / branch', color: 'error' })
  }
}

async function loadAccess() {
  if (!solutionId.value) return
  loadingAccess.value = true
  try {
    const base = `/api/v1/companies/${companyId.value}/users/${userId.value}`
    const query = { solution_id: solutionId.value }
    const [asg, perm, rl] = await Promise.all([
      api.get<RoleAssignmentDetail[]>(`${base}/role-assignments`, { query }),
      api.get<EffectivePermission[]>(`${base}/effective-permissions`, { query }),
      api.get<RoleResponse[]>(`/api/v1/companies/${companyId.value}/solutions/${solutionId.value}/roles`, { query: { limit: 100 } }),
    ])
    assignments.value = asg.data
    permissions.value = perm.data
    roles.value = rl.data
  } catch (err) {
    toast.add({ title: (err as { message?: string }).message || 'Gagal memuat role', color: 'error' })
  } finally {
    loadingAccess.value = false
  }
}

async function assign() {
  assigning.value = true
  try {
    await api.post(`/api/v1/companies/${companyId.value}/users/${userId.value}/role-assignments`, {
      role_id: form.role_id,
      scope_type: form.scope_type,
      scope_id: form.scope_type === 'branch' ? form.scope_id : undefined,
    })
    toast.add({ title: 'Role di-assign', color: 'success' })
    form.role_id = ''
    form.scope_id = ''
    await loadAccess()
  } catch (err) {
    toast.add({ title: (err as { message?: string }).message || 'Gagal assign role', color: 'error' })
  } finally {
    assigning.value = false
  }
}

async function revoke(a: RoleAssignmentDetail) {
  if (!confirm(`Cabut role ${a.role_name} (${a.scope_type}: ${a.scope_name || a.scope_id})?`)) return
  revokingId.value = a.id
  try {
    await api.del(`/api/v1/role-assignments/${a.id}`)
    toast.add({ title: 'Role dicabut', color: 'success' })
    await loadAccess()
  } catch (err) {
    toast.add({ title: (err as { message?: string }).message || 'Gagal mencabut role', color: 'error' })
  } finally {
    revokingId.value = null
  }
}

function scopeLabel(scopeType: string, scopeId: string) {
  if (scopeType === 'company') return 'Seluruh company'
  return branches.value.find(b => b.id === scopeId)?.name ?? scopeId
}

watch(solutionId, loadAccess)

onMounted(async () => {
  if (isForeignCompany.value) return
  await Promise.all([loadMember(), loadOptions()])
})
</script>

<template>
  <div>
    <PageHeader :title="member?.name ?? 'Member'" :subtitle="member?.email">
      <template #actions>
        <UButton :to="`/companies/${companyId}/members`" color="neutral" variant="outline" icon="i-lucide-arrow-left">
          Kembali
        </UButton>
      </template>
    </PageHeader>

    <UAlert
      v-if="isForeignCompany"
      class="mb-4"
      color="warning"
      variant="subtle"
      icon="i-lucide-triangle-alert"
      title="Company ini bukan company aktif"
      description="Aktifkan company ini dulu (company switcher) untuk mengelola role member."
    />
    <UAlert v-if="error" color="error" variant="subtle" :title="error" class="mb-4" />

    <template v-if="member && !isForeignCompany">
      <UCard class="mb-6">
        <div class="flex flex-wrap items-end gap-6">
          <div>
            <p class="text-xs uppercase tracking-wide text-dimmed">Status member</p>
            <UBadge
              class="mt-1"
              size="sm"
              variant="subtle"
              :color="member.status === 'active' ? 'success' : member.status === 'inactive' ? 'error' : 'warning'"
            >
              {{ member.status }}
            </UBadge>
          </div>
          <UFormField label="Solution" class="min-w-72">
            <USelect v-model="solutionId" :items="solutionItems" placeholder="Pilih solution" class="w-full" />
          </UFormField>
        </div>
        <UAlert
          v-if="member.status !== 'active'"
          class="mt-4"
          color="warning"
          variant="subtle"
          icon="i-lucide-info"
          title="Member tidak aktif"
          description="Role hanya bisa di-assign ke member active, dan permission member non-active selalu kosong."
        />
      </UCard>

      <div class="grid gap-6 lg:grid-cols-2">
        <UCard>
          <h2 class="mb-4 font-medium text-highlighted">Role assignments</h2>

          <form class="mb-4 grid gap-3 sm:grid-cols-2" @submit.prevent="assign">
            <UFormField label="Role" class="sm:col-span-2">
              <USelect v-model="form.role_id" :items="roleItems" placeholder="Pilih role" class="w-full" />
            </UFormField>
            <UFormField label="Scope">
              <USelect
                v-model="form.scope_type"
                :items="[{ label: 'Seluruh company', value: 'company' }, { label: 'Branch', value: 'branch' }]"
                class="w-full"
              />
            </UFormField>
            <UFormField v-if="form.scope_type === 'branch'" label="Branch">
              <USelect v-model="form.scope_id" :items="branchItems" placeholder="Pilih branch" class="w-full" />
            </UFormField>
            <div class="sm:col-span-2">
              <UButton
                type="submit"
                color="primary"
                icon="i-lucide-plus"
                :loading="assigning"
                :disabled="!form.role_id || (form.scope_type === 'branch' && !form.scope_id) || member.status !== 'active'"
              >
                Assign role
              </UButton>
            </div>
          </form>

          <p v-if="loadingAccess" class="text-sm text-muted">Memuat...</p>
          <p v-else-if="!assignments.length" class="text-sm text-muted">Belum ada role untuk solution ini.</p>
          <ul v-else class="divide-y divide-default">
            <li v-for="a in assignments" :key="a.id" class="flex items-center justify-between gap-3 py-2">
              <div>
                <p class="font-medium text-highlighted">
                  {{ a.role_name }}
                  <UBadge size="sm" variant="subtle" :color="a.role_is_global ? 'neutral' : 'primary'" class="ml-1">
                    {{ a.role_is_global ? 'global' : 'custom' }}
                  </UBadge>
                </p>
                <p class="text-xs text-muted">
                  <span class="font-mono">{{ a.role_code }}</span> · {{ a.scope_type === 'company' ? 'Seluruh company' : `Branch ${a.scope_name}` }}
                </p>
              </div>
              <UButton
                size="xs"
                color="error"
                variant="outline"
                icon="i-lucide-x"
                :loading="revokingId === a.id"
                @click="revoke(a)"
              >
                Cabut
              </UButton>
            </li>
          </ul>
        </UCard>

        <UCard>
          <h2 class="mb-1 font-medium text-highlighted">Effective permissions</h2>
          <p class="mb-4 text-sm text-muted">Semua permission yang dimiliki member di solution ini, beserta asalnya.</p>
          <p v-if="loadingAccess" class="text-sm text-muted">Memuat...</p>
          <p v-else-if="!permissions.length" class="text-sm text-muted">Tidak ada permission.</p>
          <div v-else class="overflow-x-auto">
            <table class="w-full text-sm">
              <thead>
                <tr class="border-b border-default text-left text-xs uppercase tracking-wide text-dimmed">
                  <th class="px-2 py-2">Permission</th>
                  <th class="px-2 py-2">Dari role</th>
                </tr>
              </thead>
              <tbody>
                <tr v-for="p in permissions" :key="p.permission" class="border-b border-default/60 align-top">
                  <td class="px-2 py-2 font-mono text-xs text-highlighted">{{ p.permission }}</td>
                  <td class="px-2 py-2">
                    <div class="flex flex-wrap gap-1">
                      <UBadge v-for="(g, i) in p.grants" :key="i" size="sm" color="neutral" variant="subtle">
                        {{ g.role_code }} · {{ scopeLabel(g.scope_type, g.scope_id) }}
                      </UBadge>
                    </div>
                  </td>
                </tr>
              </tbody>
            </table>
          </div>
        </UCard>
      </div>
    </template>
  </div>
</template>
