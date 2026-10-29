<script setup lang="ts">
import type {
  BranchResponse,
  CompanyMemberResponse,
  EffectivePermission,
  MemberStatus,
  RoleAssignmentDetail,
  RoleResponse,
} from '~/types/api'

const ctx = useCompanyContext()
const route = useRoute()
const api = useApi()
const toast = useToast()
const { toastError } = useApiError()
const { confirm } = useConfirm()

const userId = computed(() => String(route.params.userId))
const base = computed(() => `/companies/${ctx.companyId.value}`)

const member = ref<CompanyMemberResponse | null>(null)
const memberError = ref('')
const solutionId = ref<string | undefined>()

const assignments = ref<RoleAssignmentDetail[]>([])
const permissions = ref<EffectivePermission[]>([])
const roles = ref<RoleResponse[]>([])
const branches = ref<BranchResponse[]>([])
const loadingAccess = ref(false)

async function loadMember() {
  memberError.value = ''
  try {
    member.value = (await api.get<CompanyMemberResponse>(`/api/v1/companies/${ctx.companyId.value}/members/${userId.value}`)).data
  } catch (err) {
    memberError.value = (err as { message?: string }).message || 'Gagal memuat anggota'
  }
}

async function loadBranches() {
  try {
    branches.value = (await api.get<BranchResponse[]>(`/api/v1/companies/${ctx.companyId.value}/branches`, { query: { limit: 100 } })).data
  } catch {
    branches.value = []
  }
}

async function loadAccess() {
  if (!solutionId.value) return
  loadingAccess.value = true
  try {
    const userBase = `/api/v1/companies/${ctx.companyId.value}/users/${userId.value}`
    const query = { solution_id: solutionId.value }
    const [a, p, r] = await Promise.all([
      api.get<RoleAssignmentDetail[]>(`${userBase}/role-assignments`, { query }),
      api.get<EffectivePermission[]>(`${userBase}/effective-permissions`, { query }),
      api.get<RoleResponse[]>(`/api/v1/companies/${ctx.companyId.value}/solutions/${solutionId.value}/roles`, { query: { limit: 200 } }),
    ])
    assignments.value = a.data
    permissions.value = p.data
    roles.value = r.data
  } catch (err) {
    toastError(err, 'Gagal memuat role')
  } finally {
    loadingAccess.value = false
  }
}

watch(solutionId, loadAccess)
watch(ctx.canAccess, (ok) => {
  if (ok) {
    loadMember()
    loadBranches()
  }
}, { immediate: true })

// ── Status anggota ──
async function setStatus(next: MemberStatus) {
  if (!member.value) return
  if (next === 'inactive') {
    const ok = await confirm({
      title: `Keluarkan ${member.value.name}?`,
      description: 'Semua role-nya di company ini ikut dicabut.',
      confirmLabel: 'Keluarkan',
      danger: true,
    })
    if (!ok) return
  }
  try {
    member.value = (await api.patch<CompanyMemberResponse>(
      `/api/v1/companies/${ctx.companyId.value}/members/${userId.value}`, { status: next },
    )).data
    toast.add({ title: 'Status anggota diperbarui', color: 'success' })
    loadAccess()
  } catch (err) {
    toastError(err)
  }
}

// ── Assign role ──
const showAssign = ref(false)
const assigning = ref(false)
const form = reactive<{ role_id?: string; scope_type: 'company' | 'branch'; scope_id?: string }>({ scope_type: 'company' })

const roleItems = computed(() => roles.value.map(r => ({
  label: r.name,
  value: r.id,
  description: `${r.code} · ${r.scope === 'company' ? 'custom' : 'global'} · ${r.permission_count} permission`,
})))
const branchItems = computed(() => branches.value.map(b => ({ label: b.name, value: b.id, description: b.code })))

watch(showAssign, (v) => {
  if (v) Object.assign(form, { role_id: undefined, scope_type: 'company', scope_id: undefined })
})

async function assign() {
  assigning.value = true
  try {
    await api.post(`/api/v1/companies/${ctx.companyId.value}/users/${userId.value}/role-assignments`, {
      role_id: form.role_id,
      scope_type: form.scope_type,
      scope_id: form.scope_type === 'branch' ? form.scope_id : undefined,
    })
    toast.add({ title: 'Role ditambahkan', color: 'success' })
    showAssign.value = false
    loadAccess()
  } catch (err) {
    toastError(err, 'Gagal menambahkan role')
  } finally {
    assigning.value = false
  }
}

async function revoke(a: RoleAssignmentDetail) {
  const ok = await confirm({
    title: `Cabut role ${a.role_name}?`,
    description: a.scope_type === 'company' ? 'Berlaku di seluruh company.' : `Berlaku di cabang ${a.scope_name}.`,
    confirmLabel: 'Cabut',
    danger: true,
  })
  if (!ok) return
  try {
    await api.del(`/api/v1/role-assignments/${a.id}`)
    toast.add({ title: 'Role dicabut', color: 'success' })
    loadAccess()
  } catch (err) {
    toastError(err)
  }
}

// ── Permission efektif dikelompokkan per modul ──
const permissionGroups = computed(() => {
  const map = new Map<string, EffectivePermission[]>()
  for (const p of permissions.value) {
    const key = p.permission.includes('.') ? p.permission.split('.')[0]! : 'lainnya'
    map.set(key, [...(map.get(key) ?? []), p])
  }
  return [...map.entries()].map(([name, items]) => ({ name, items }))
})

function scopeLabel(type: string, id: string) {
  if (type === 'company') return 'seluruh company'
  return branches.value.find(b => b.id === id)?.name ?? 'cabang'
}
</script>

<template>
  <CompanyPage
    :title="member?.name ?? 'Anggota'"
    :section="{ label: 'Anggota', to: `${base}/members` }"
    :trail="member ? [{ label: member.name }] : []"
  >
    <UAlert v-if="memberError" color="error" variant="subtle" :title="memberError" />

    <template v-else>
      <!-- Header anggota -->
      <div class="flex flex-wrap items-center justify-between gap-4">
        <div class="flex min-w-0 items-center gap-3">
          <USkeleton v-if="!member" class="size-12 rounded-full" />
          <UAvatar v-else :text="initials(member.name)" size="xl" />
          <div class="min-w-0">
            <USkeleton v-if="!member" class="h-5 w-40" />
            <template v-else>
              <p class="truncate text-lg font-semibold text-highlighted">{{ member.name }}</p>
              <p class="flex flex-wrap items-center gap-2 text-sm text-muted">
                {{ member.email }} <StatusBadge :status="member.status" />
                <span class="text-dimmed">· bergabung {{ formatDate(member.joined_at) }}</span>
              </p>
            </template>
          </div>
        </div>
        <div v-if="member" class="flex gap-2">
          <UButton v-if="member.status !== 'active'" icon="i-lucide-check" variant="soft" @click="setStatus('active')">Aktifkan</UButton>
          <UButton v-if="member.status !== 'inactive'" icon="i-lucide-user-minus" color="error" variant="ghost" @click="setStatus('inactive')">
            Keluarkan
          </UButton>
        </div>
      </div>

      <!-- Pilih solution -->
      <div class="flex flex-wrap items-center gap-3 rounded-lg border border-default bg-elevated/30 p-3">
        <span class="text-sm text-muted">Role dan akses untuk solution</span>
        <SolutionSelect v-model="solutionId" />
      </div>

      <div class="grid gap-6 lg:grid-cols-2">
        <!-- Role -->
        <SectionCard title="Role" description="Role menentukan permission anggota di solution ini.">
          <template #actions>
            <UButton
              icon="i-lucide-plus"
              size="sm"
              :disabled="!solutionId || member?.status !== 'active'"
              @click="showAssign = true"
            >
              Tambah role
            </UButton>
          </template>

          <div v-if="loadingAccess && !assignments.length" class="space-y-2">
            <USkeleton v-for="i in 2" :key="i" class="h-14 w-full" />
          </div>
          <UEmpty
            v-else-if="!assignments.length"
            icon="i-lucide-shield-off"
            title="Belum punya role"
            :description="member?.status === 'active' ? 'Tambahkan role agar anggota punya akses.' : 'Aktifkan anggota dulu untuk menambahkan role.'"
            variant="naked"
          />
          <ul v-else class="divide-y divide-default rounded-lg border border-default">
            <li v-for="a in assignments" :key="a.id" class="flex items-center justify-between gap-3 px-4 py-3">
              <div class="min-w-0">
                <NuxtLink :to="`${base}/roles/${a.role_id}`" class="font-medium text-highlighted hover:text-primary">
                  {{ a.role_name }}
                </NuxtLink>
                <p class="flex flex-wrap items-center gap-1.5 text-xs text-muted">
                  <UBadge size="xs" variant="subtle" :color="a.role_is_global ? 'neutral' : 'primary'">
                    {{ a.role_is_global ? 'global' : 'custom' }}
                  </UBadge>
                  <UIcon :name="a.scope_type === 'company' ? 'i-lucide-building-2' : 'i-lucide-git-branch'" class="size-3.5" />
                  {{ a.scope_type === 'company' ? 'Seluruh company' : a.scope_name }}
                </p>
              </div>
              <UTooltip text="Cabut role">
                <UButton icon="i-lucide-x" color="neutral" variant="ghost" size="sm" aria-label="Cabut role" @click="revoke(a)" />
              </UTooltip>
            </li>
          </ul>
        </SectionCard>

        <!-- Permission efektif -->
        <SectionCard title="Akses (permission)" description="Gabungan permission dari semua role di atas.">
          <div v-if="loadingAccess && !permissions.length" class="space-y-2">
            <USkeleton v-for="i in 3" :key="i" class="h-8 w-full" />
          </div>
          <UEmpty v-else-if="!permissions.length" icon="i-lucide-key-round" title="Tidak ada akses" description="Anggota belum punya permission di solution ini." variant="naked" />
          <div v-else class="space-y-4">
            <div v-for="g in permissionGroups" :key="g.name">
              <p class="mb-1.5 text-xs font-semibold uppercase tracking-wide text-dimmed">{{ g.name }}</p>
              <ul class="space-y-1.5">
                <li v-for="p in g.items" :key="p.permission" class="flex flex-wrap items-center justify-between gap-2 text-sm">
                  <span class="font-mono text-xs text-highlighted">{{ p.permission }}</span>
                  <span class="flex flex-wrap gap-1">
                    <UBadge v-for="(gr, i) in p.grants" :key="i" size="xs" color="neutral" variant="outline">
                      {{ gr.role_code }} · {{ scopeLabel(gr.scope_type, gr.scope_id) }}
                    </UBadge>
                  </span>
                </li>
              </ul>
            </div>
          </div>
        </SectionCard>
      </div>
    </template>

    <UModal v-model:open="showAssign" title="Tambah role" :description="member ? `Untuk ${member.name}` : undefined">
      <template #body>
        <div class="space-y-4">
          <UFormField label="Role" required>
            <USelectMenu
              v-model="form.role_id"
              :items="roleItems"
              value-key="value"
              placeholder="Pilih role"
              :search-input="{ placeholder: 'Cari role...' }"
              class="w-full"
            />
          </UFormField>
          <UFormField label="Berlaku di">
            <URadioGroup
              v-model="form.scope_type"
              :items="[
                { label: 'Seluruh company', value: 'company', description: 'Termasuk semua cabang.' },
                { label: 'Cabang tertentu', value: 'branch', description: 'Hanya di satu cabang.', disabled: !branches.length },
              ]"
            />
          </UFormField>
          <UFormField v-if="form.scope_type === 'branch'" label="Cabang" required>
            <USelectMenu v-model="form.scope_id" :items="branchItems" value-key="value" placeholder="Pilih cabang" class="w-full" />
          </UFormField>
        </div>
      </template>
      <template #footer>
        <div class="flex w-full justify-end gap-2">
          <UButton color="neutral" variant="ghost" @click="showAssign = false">Batal</UButton>
          <UButton
            :loading="assigning"
            :disabled="!form.role_id || (form.scope_type === 'branch' && !form.scope_id)"
            @click="assign"
          >
            Tambahkan
          </UButton>
        </div>
      </template>
    </UModal>
  </CompanyPage>
</template>
