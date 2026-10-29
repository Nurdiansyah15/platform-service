<script setup lang="ts">
import type { FormError, FormSubmitEvent } from '@nuxt/ui'
import type { UpdateMeRequest } from '~/types/api'

const auth = useAuthStore()
const api = useApi()
const toast = useToast()
const colorMode = useColorMode()
const { toastError, fieldErrors } = useApiError()
const pwForm = useTemplateRef('pwForm')

const profile = reactive({ name: auth.user?.name ?? '' })
const savingName = ref(false)
const pw = reactive({ current: '', next: '', confirm: '' })
const savingPassword = ref(false)

const themes = [
  { label: 'Terang', value: 'light', icon: 'i-lucide-sun' },
  { label: 'Gelap', value: 'dark', icon: 'i-lucide-moon' },
  { label: 'Sistem', value: 'system', icon: 'i-lucide-monitor' },
]

async function update(body: UpdateMeRequest) {
  await api.patch('/api/v1/users/me', body)
  await auth.loadMe()
}

async function saveName(e: FormSubmitEvent<typeof profile>) {
  savingName.value = true
  try {
    await update({ name: e.data.name.trim() })
    toast.add({ title: 'Nama diperbarui', color: 'success' })
  } catch (err) {
    toastError(err, 'Gagal memperbarui nama')
  } finally {
    savingName.value = false
  }
}

function validatePw(s: typeof pw): FormError[] {
  const errors: FormError[] = []
  if (!s.current) errors.push({ name: 'current', message: 'Wajib diisi' })
  if (s.next.length < 8) errors.push({ name: 'next', message: 'Minimal 8 karakter' })
  if (s.confirm !== s.next) errors.push({ name: 'confirm', message: 'Konfirmasi tidak sama' })
  return errors
}

async function savePassword(e: FormSubmitEvent<typeof pw>) {
  savingPassword.value = true
  try {
    await update({ password: e.data.next, current_password: e.data.current })
    toast.add({ title: 'Password diganti', color: 'success' })
    pw.current = ''
    pw.next = ''
    pw.confirm = ''
  } catch (err) {
    const fields = fieldErrors(err)
    if (fields.length) pwForm.value?.setErrors(fields)
    else pwForm.value?.setErrors([{ name: 'current', message: (err as { message?: string }).message ?? 'Gagal' }])
  } finally {
    savingPassword.value = false
  }
}
</script>

<template>
  <AppPage title="Profil" narrow>
    <SectionCard title="Data diri" description="Nama yang tampil untuk anggota lain.">
      <UForm :state="profile" class="space-y-4" @submit="saveName">
        <UFormField label="Email" help="Hubungi admin platform untuk mengubah email.">
          <UInput :model-value="auth.user?.email" disabled class="w-full" />
        </UFormField>
        <UFormField label="Nama" name="name" required>
          <UInput v-model="profile.name" class="w-full" />
        </UFormField>
        <UButton type="submit" :loading="savingName" :disabled="!profile.name.trim() || profile.name.trim() === auth.user?.name">
          Simpan
        </UButton>
      </UForm>
    </SectionCard>

    <SectionCard title="Ganti password" description="Gunakan minimal 8 karakter.">
      <UForm ref="pwForm" :state="pw" :validate="validatePw" :validate-on="['blur']" class="space-y-4" @submit="savePassword">
        <UFormField label="Password saat ini" name="current" required>
          <UInput v-model="pw.current" type="password" autocomplete="current-password" class="w-full" />
        </UFormField>
        <div class="grid gap-4 sm:grid-cols-2">
          <UFormField label="Password baru" name="next" required>
            <UInput v-model="pw.next" type="password" autocomplete="new-password" class="w-full" />
          </UFormField>
          <UFormField label="Ulangi password baru" name="confirm" required>
            <UInput v-model="pw.confirm" type="password" autocomplete="new-password" class="w-full" />
          </UFormField>
        </div>
        <UButton type="submit" :loading="savingPassword">Ganti password</UButton>
      </UForm>
    </SectionCard>

    <SectionCard title="Tampilan" description="Tema console di perangkat ini.">
      <div class="flex flex-wrap gap-2">
        <UButton
          v-for="t in themes"
          :key="t.value"
          :icon="t.icon"
          :color="colorMode.preference === t.value ? 'primary' : 'neutral'"
          :variant="colorMode.preference === t.value ? 'soft' : 'outline'"
          @click="colorMode.preference = t.value"
        >
          {{ t.label }}
        </UButton>
      </div>
    </SectionCard>
  </AppPage>
</template>
