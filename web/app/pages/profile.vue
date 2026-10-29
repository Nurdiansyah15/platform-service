<script setup lang="ts">
import type { UpdateMeRequest, UserResponse } from '~/types/api'

const auth = useAuthStore()
const api = useApi()
const toast = useToast()

const name = ref(auth.user?.name ?? '')
const savingName = ref(false)

const pw = reactive({ current: '', next: '', confirm: '' })
const savingPassword = ref(false)
const pwError = computed(() => {
  if (pw.next && pw.next.length < 8) return 'Password baru minimal 8 karakter.'
  if (pw.confirm && pw.next !== pw.confirm) return 'Konfirmasi password tidak sama.'
  return ''
})

async function update(body: UpdateMeRequest, successTitle: string) {
  const res = await api.patch<UserResponse>('/api/v1/users/me', body)
  await auth.loadMe()
  toast.add({ title: successTitle, color: 'success' })
  return res.data
}

async function saveName() {
  savingName.value = true
  try {
    const u = await update({ name: name.value }, 'Nama diperbarui')
    name.value = u.name
  } catch (err) {
    toast.add({ title: (err as { message?: string }).message || 'Gagal memperbarui nama', color: 'error' })
  } finally {
    savingName.value = false
  }
}

async function savePassword() {
  savingPassword.value = true
  try {
    await update({ password: pw.next, current_password: pw.current }, 'Password diganti')
    pw.current = ''
    pw.next = ''
    pw.confirm = ''
  } catch (err) {
    toast.add({ title: (err as { message?: string }).message || 'Gagal mengganti password', color: 'error' })
  } finally {
    savingPassword.value = false
  }
}
</script>

<template>
  <div>
    <PageHeader title="Profil" subtitle="Data akunmu di platform." />

    <div class="grid max-w-4xl gap-6 lg:grid-cols-2">
      <UCard>
        <h2 class="mb-4 font-medium text-highlighted">Data diri</h2>
        <form class="space-y-4" @submit.prevent="saveName">
          <UFormField label="Email" hint="hubungi admin platform untuk mengubah">
            <UInput :model-value="auth.user?.email" disabled class="w-full" />
          </UFormField>
          <UFormField label="Nama" required>
            <UInput v-model="name" class="w-full" />
          </UFormField>
          <UButton type="submit" color="primary" :loading="savingName" :disabled="!name.trim() || name === auth.user?.name">
            Simpan
          </UButton>
        </form>
      </UCard>

      <UCard>
        <h2 class="mb-4 font-medium text-highlighted">Ganti password</h2>
        <form class="space-y-4" @submit.prevent="savePassword">
          <UFormField label="Password lama" required>
            <UInput v-model="pw.current" type="password" autocomplete="current-password" class="w-full" />
          </UFormField>
          <UFormField label="Password baru" required>
            <UInput v-model="pw.next" type="password" autocomplete="new-password" class="w-full" />
          </UFormField>
          <UFormField label="Ulangi password baru" required>
            <UInput v-model="pw.confirm" type="password" autocomplete="new-password" class="w-full" />
          </UFormField>
          <UAlert v-if="pwError" color="warning" variant="subtle" :title="pwError" />
          <UButton
            type="submit"
            color="primary"
            :loading="savingPassword"
            :disabled="!pw.current || !pw.next || pw.next !== pw.confirm || !!pwError"
          >
            Ganti password
          </UButton>
        </form>
      </UCard>
    </div>
  </div>
</template>
