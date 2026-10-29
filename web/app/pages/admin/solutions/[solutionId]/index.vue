<script setup lang="ts">
import type { FormError, FormSubmitEvent } from '@nuxt/ui'
import type { SolutionResponse } from '~/types/api'

definePageMeta({ middleware: 'admin' })

const ctx = useSolutionContext()
const api = useApi()
const toast = useToast()
const { toastError } = useApiError()
const { confirm } = useConfirm()

const state = reactive({ name: '', description: '' })
const saving = ref(false)

watch(ctx.solution, (s) => {
  if (s) Object.assign(state, { name: s.name, description: s.description ?? '' })
}, { immediate: true })

const dirty = computed(() => !!ctx.solution.value
  && (state.name.trim() !== ctx.solution.value.name || state.description.trim() !== (ctx.solution.value.description ?? '')))

function validate(s: typeof state): FormError[] {
  return s.name.trim() ? [] : [{ name: 'name', message: 'Nama wajib diisi' }]
}

async function patch(body: Record<string, string>, successTitle: string) {
  const res = await api.patch<SolutionResponse>(`/api/v1/solutions/${ctx.solutionId.value}`, body)
  ctx.setSolution(res.data)
  toast.add({ title: successTitle, color: 'success' })
}

async function onSubmit(e: FormSubmitEvent<typeof state>) {
  saving.value = true
  try {
    await patch({ name: e.data.name.trim(), description: e.data.description.trim() }, 'Solution diperbarui')
  } catch (err) {
    toastError(err, 'Gagal memperbarui solution')
  } finally {
    saving.value = false
  }
}

async function toggleStatus() {
  const s = ctx.solution.value
  if (!s) return
  const deactivate = s.status === 'active'
  if (deactivate) {
    const ok = await confirm({
      title: `Nonaktifkan ${s.name}?`,
      description: 'User tidak bisa login ke solution ini dan client-nya tidak bisa menukar token.',
      confirmLabel: 'Nonaktifkan',
      danger: true,
      typeToConfirm: s.code,
    })
    if (!ok) return
  }
  try {
    await patch({ status: deactivate ? 'inactive' : 'active' }, deactivate ? 'Solution dinonaktifkan' : 'Solution diaktifkan')
  } catch (err) {
    toastError(err)
  }
}
</script>

<template>
  <SolutionPage :title="ctx.solution.value?.name ?? 'Solution'" narrow>
    <template v-if="ctx.solution.value">
      <SectionCard title="Informasi">
        <dl class="grid gap-4 text-sm sm:grid-cols-3">
          <div>
            <dt class="text-muted">Kode (audience)</dt>
            <dd class="mt-1"><CopyText :value="ctx.solution.value.code" mono /></dd>
          </div>
          <div>
            <dt class="text-muted">Status</dt>
            <dd class="mt-1"><StatusBadge :status="ctx.solution.value.status" /></dd>
          </div>
          <div>
            <dt class="text-muted">Versi katalog permission</dt>
            <dd class="mt-1 font-mono text-xs">{{ ctx.solution.value.permission_catalog_version ?? 'belum sync' }}</dd>
          </div>
        </dl>
      </SectionCard>

      <SectionCard title="Ubah solution">
        <UForm :state="state" :validate="validate" class="space-y-4" @submit="onSubmit">
          <UFormField label="Nama" name="name" required>
            <UInput v-model="state.name" class="w-full" />
          </UFormField>
          <UFormField label="Deskripsi" name="description">
            <UTextarea v-model="state.description" :rows="3" class="w-full" />
          </UFormField>
          <UButton type="submit" :loading="saving" :disabled="!dirty">Simpan perubahan</UButton>
        </UForm>
      </SectionCard>

      <SectionCard
        :title="ctx.solution.value.status === 'active' ? 'Nonaktifkan solution' : 'Aktifkan solution'"
        :description="ctx.solution.value.status === 'active'
          ? 'Login user dan token client untuk solution ini akan ditolak.'
          : 'User dan client bisa memakai solution ini lagi.'"
      >
        <template #actions>
          <UButton :color="ctx.solution.value.status === 'active' ? 'error' : 'success'" variant="outline" @click="toggleStatus">
            {{ ctx.solution.value.status === 'active' ? 'Nonaktifkan' : 'Aktifkan' }}
          </UButton>
        </template>
      </SectionCard>
    </template>
    <USkeleton v-else class="h-48 w-full" />
  </SolutionPage>
</template>
