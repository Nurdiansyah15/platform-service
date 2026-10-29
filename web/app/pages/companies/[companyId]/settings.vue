<script setup lang="ts">
import type { FormError, FormSubmitEvent } from '@nuxt/ui'
import type { CompanyResponse, CompanyStatus } from '~/types/api'

const ctx = useCompanyContext()
const auth = useAuthStore()
const api = useApi()
const toast = useToast()
const { toastError, fieldErrors } = useApiError()
const { confirm } = useConfirm()
const formRef = useTemplateRef('formRef')

const state = reactive({ name: '', slug: '' })
const saving = ref(false)
const changingStatus = ref(false)

watch(ctx.company, (c) => {
  if (c) Object.assign(state, { name: c.name, slug: c.slug })
}, { immediate: true })

const dirty = computed(() => !!ctx.company.value
  && (state.name.trim() !== ctx.company.value.name || state.slug !== ctx.company.value.slug))

function validate(s: typeof state): FormError[] {
  const errors: FormError[] = []
  if (!s.name.trim()) errors.push({ name: 'name', message: 'Nama wajib diisi' })
  if (!/^[a-z0-9]+(?:-[a-z0-9]+)*$/.test(s.slug)) errors.push({ name: 'slug', message: 'Hanya huruf kecil, angka, dan tanda hubung' })
  return errors
}

async function onSubmit(e: FormSubmitEvent<typeof state>) {
  if (!ctx.company.value) return
  saving.value = true
  try {
    const body: Record<string, string> = {}
    if (e.data.name.trim() !== ctx.company.value.name) body.name = e.data.name.trim()
    if (e.data.slug !== ctx.company.value.slug) body.slug = e.data.slug
    const res = await api.patch<CompanyResponse>(`/api/v1/companies/${ctx.companyId.value}`, body)
    ctx.setCompany(res.data)
    await auth.loadCompanies()
    toast.add({ title: 'Company diperbarui', color: 'success' })
  } catch (err) {
    const fields = fieldErrors(err)
    if (fields.length) formRef.value?.setErrors(fields)
    else toastError(err, 'Gagal memperbarui company')
  } finally {
    saving.value = false
  }
}

const statusActions: { status: CompanyStatus; label: string; description: string; color: 'success' | 'warning' | 'error' }[] = [
  { status: 'active', label: 'Aktifkan', description: 'Anggota bisa memilih dan memakai company ini.', color: 'success' },
  { status: 'suspended', label: 'Tangguhkan', description: 'Sementara tidak bisa dipilih anggota. Bisa diaktifkan lagi.', color: 'warning' },
  { status: 'closed', label: 'Tutup', description: 'Company tidak dipakai lagi. Data tetap tersimpan.', color: 'error' },
]

async function changeStatus(next: CompanyStatus) {
  const c = ctx.company.value
  if (!c) return
  const meta = statusActions.find(a => a.status === next)!
  const ok = await confirm({
    title: `${meta.label} ${c.name}?`,
    description: meta.description,
    confirmLabel: meta.label,
    danger: next !== 'active',
    typeToConfirm: next === 'closed' ? c.slug : undefined,
  })
  if (!ok) return
  changingStatus.value = true
  try {
    const res = await api.patch<CompanyResponse>(`/api/v1/companies/${c.id}`, { status: next })
    ctx.setCompany(res.data)
    toast.add({ title: `Status: ${statusMeta(next).label}`, color: 'success' })
  } catch (err) {
    toastError(err)
  } finally {
    changingStatus.value = false
  }
}
</script>

<template>
  <CompanyPage title="Pengaturan" :section="{ label: 'Pengaturan', to: `/companies/${ctx.companyId.value}/settings` }" narrow>
    <SectionCard title="Profil company" description="Nama dan slug company.">
      <UForm ref="formRef" :state="state" :validate="validate" class="space-y-4" @submit="onSubmit">
        <UFormField label="Nama" name="name" required>
          <UInput v-model="state.name" class="w-full" />
        </UFormField>
        <UFormField label="Slug" name="slug" required help="Pengenal unik company.">
          <UInput v-model="state.slug" class="w-full" />
        </UFormField>
        <UButton type="submit" :loading="saving" :disabled="!dirty">Simpan perubahan</UButton>
      </UForm>
    </SectionCard>

    <SectionCard
      v-if="auth.isPlatformAdmin && ctx.company.value"
      title="Status company"
      description="Khusus admin platform."
    >
      <div class="flex items-center gap-2 text-sm">
        Status saat ini: <StatusBadge :status="ctx.company.value.status" />
      </div>
      <ul class="divide-y divide-default rounded-lg border border-default">
        <li v-for="a in statusActions.filter(x => x.status !== ctx.company.value?.status)" :key="a.status" class="flex items-center justify-between gap-4 p-4">
          <div>
            <p class="text-sm font-medium text-highlighted">{{ a.label }}</p>
            <p class="text-sm text-muted">{{ a.description }}</p>
          </div>
          <UButton :color="a.color" variant="outline" :loading="changingStatus" @click="changeStatus(a.status)">{{ a.label }}</UButton>
        </li>
      </ul>
    </SectionCard>
    <p v-else-if="ctx.company.value" class="text-sm text-muted">
      Status company: <StatusBadge :status="ctx.company.value.status" /> — hanya admin platform yang bisa mengubahnya.
    </p>
  </CompanyPage>
</template>
