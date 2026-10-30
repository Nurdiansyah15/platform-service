<script setup lang="ts">
import type { FormError, FormSubmitEvent } from '@nuxt/ui'
import type { SolutionResponse, UpdateSolutionRequest } from '~/types/api'

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

// ── Tautan aplikasi (SSO) ──
const links = reactive({ launch_url: '', icon_url: '', redirect_uris: '' })
const savingLinks = ref(false)

const splitLines = (v: string) => v.split('\n').map(l => l.trim()).filter(Boolean)

watch(ctx.solution, (s) => {
  if (s) {
    Object.assign(links, {
      launch_url: s.launch_url ?? '',
      icon_url: s.icon_url ?? '',
      redirect_uris: (s.redirect_uris ?? []).join('\n'),
    })
  }
}, { immediate: true })

const linksDirty = computed(() => {
  const s = ctx.solution.value
  if (!s) return false
  return links.launch_url.trim() !== (s.launch_url ?? '')
    || links.icon_url.trim() !== (s.icon_url ?? '')
    || splitLines(links.redirect_uris).join('\n') !== (s.redirect_uris ?? []).join('\n')
})

function isAppURL(v: string) {
  try {
    const u = new URL(v)
    return u.protocol === 'https:' || (u.protocol === 'http:' && ['localhost', '127.0.0.1'].includes(u.hostname))
  } catch {
    return false
  }
}

function validateLinks(s: typeof links): FormError[] {
  const errors: FormError[] = []
  const msg = 'URL harus https (http hanya untuk localhost)'
  if (s.launch_url.trim() && !isAppURL(s.launch_url.trim())) errors.push({ name: 'launch_url', message: msg })
  if (s.icon_url.trim() && !isAppURL(s.icon_url.trim())) errors.push({ name: 'icon_url', message: msg })
  const uris = splitLines(s.redirect_uris)
  if (uris.length > 10) errors.push({ name: 'redirect_uris', message: 'Maksimal 10 redirect URI' })
  else if (uris.some(u => !isAppURL(u) || u.includes('#'))) errors.push({ name: 'redirect_uris', message: `${msg}, tanpa #fragment` })
  return errors
}

async function onSubmitLinks(e: FormSubmitEvent<typeof links>) {
  savingLinks.value = true
  try {
    await patch({
      launch_url: e.data.launch_url.trim(),
      icon_url: e.data.icon_url.trim(),
      redirect_uris: splitLines(e.data.redirect_uris),
    }, 'Tautan aplikasi diperbarui')
  } catch (err) {
    toastError(err, 'Gagal memperbarui tautan aplikasi')
  } finally {
    savingLinks.value = false
  }
}

async function patch(body: UpdateSolutionRequest, successTitle: string) {
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

const isActive = computed(() => ctx.solution.value?.status === 'active')

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
        title="Tautan aplikasi (SSO)"
        description="Dipakai launcher console dan menu apps. Solution tanpa URL aplikasi tidak muncul di launcher."
      >
        <UForm :state="links" :validate="validateLinks" class="space-y-4" @submit="onSubmitLinks">
          <UFormField label="URL aplikasi" name="launch_url" help="Halaman awal solution, mis. https://kasir.contoh.com/">
            <UInput v-model="links.launch_url" placeholder="https://kasir.contoh.com/" class="w-full" />
          </UFormField>
          <UFormField label="URL ikon" name="icon_url" help="Opsional. Gambar persegi untuk launcher.">
            <UInput v-model="links.icon_url" placeholder="https://kasir.contoh.com/icon.svg" class="w-full" />
          </UFormField>
          <UFormField
            label="Redirect URI"
            name="redirect_uris"
            help="Satu per baris, maksimal 10. Harus sama persis dengan redirect_uri yang dikirim solution saat SSO."
          >
            <UTextarea
              v-model="links.redirect_uris"
              :rows="3"
              placeholder="https://kasir.contoh.com/auth/callback"
              class="w-full font-mono text-xs"
            />
          </UFormField>
          <UButton type="submit" :loading="savingLinks" :disabled="!linksDirty">Simpan tautan</UButton>
        </UForm>
      </SectionCard>

      <SectionCard
        :title="isActive ? 'Nonaktifkan solution' : 'Aktifkan solution'"
        :description="isActive
          ? 'Login user dan token client untuk solution ini akan ditolak.'
          : 'User dan client bisa memakai solution ini lagi.'"
      >
        <template #actions>
          <UButton :color="isActive ? 'error' : 'success'" variant="outline" @click="toggleStatus">
            {{ isActive ? 'Nonaktifkan' : 'Aktifkan' }}
          </UButton>
        </template>
      </SectionCard>
    </template>
    <USkeleton v-else class="h-48 w-full" />
  </SolutionPage>
</template>
