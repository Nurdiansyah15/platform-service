<script setup lang="ts">
import type { FormError, FormSubmitEvent } from '@nuxt/ui'
import type { UserResponse } from '~/types/api'

const open = defineModel<boolean>('open', { default: false })
const emit = defineEmits<{ created: [user: UserResponse] }>()

const api = useApi()
const { toastError, fieldErrors } = useApiError()
const formRef = useTemplateRef('formRef')

const state = reactive({ name: '', email: '', password: '', admin: false })
const showPassword = ref(false)
const saving = ref(false)

watch(open, (v) => {
  if (v) Object.assign(state, { name: '', email: '', password: '', admin: false })
})

function generate() {
  const chars = 'ABCDEFGHJKLMNPQRSTUVWXYZabcdefghijkmnpqrstuvwxyz23456789'
  const buf = new Uint32Array(14)
  crypto.getRandomValues(buf)
  state.password = Array.from(buf, n => chars[n % chars.length]).join('')
  showPassword.value = true
}

function validate(s: typeof state): FormError[] {
  const errors: FormError[] = []
  if (!s.name.trim()) errors.push({ name: 'name', message: 'Nama wajib diisi' })
  if (!/^\S+@\S+\.\S+$/.test(s.email.trim())) errors.push({ name: 'email', message: 'Format email tidak valid' })
  if (s.password && s.password.length < 8) errors.push({ name: 'password', message: 'Minimal 8 karakter' })
  return errors
}

async function onSubmit(e: FormSubmitEvent<typeof state>) {
  saving.value = true
  try {
    const res = await api.post<UserResponse>('/api/v1/users', {
      name: e.data.name.trim(),
      email: e.data.email.trim(),
      password: e.data.password || undefined,
      is_platform_admin: e.data.admin,
    })
    open.value = false
    emit('created', res.data)
  } catch (err) {
    const fields = fieldErrors(err)
    if (fields.length) formRef.value?.setErrors(fields)
    else toastError(err, 'Gagal membuat user')
  } finally {
    saving.value = false
  }
}
</script>

<template>
  <UModal v-model:open="open" title="Buat user" description="Registrasi publik tidak tersedia; akun dibuat admin.">
    <template #body>
      <UForm ref="formRef" :state="state" :validate="validate" class="space-y-4" @submit="onSubmit">
        <UFormField label="Nama" name="name" required>
          <UInput v-model="state.name" placeholder="Budi Santoso" class="w-full" autofocus />
        </UFormField>
        <UFormField label="Email" name="email" required>
          <UInput v-model="state.email" type="email" placeholder="budi@perusahaan.com" class="w-full" />
        </UFormField>
        <UFormField label="Password awal" name="password" help="Kosongkan bila belum ingin user login; bisa di-set nanti.">
          <UFieldGroup class="w-full">
            <UInput v-model="state.password" :type="showPassword ? 'text' : 'password'" class="flex-1" placeholder="Min. 8 karakter" />
            <UButton color="neutral" variant="outline" :icon="showPassword ? 'i-lucide-eye-off' : 'i-lucide-eye'" aria-label="Tampilkan password" @click="showPassword = !showPassword" />
            <UButton color="neutral" variant="outline" icon="i-lucide-wand-sparkles" @click="generate">Buat</UButton>
          </UFieldGroup>
        </UFormField>
        <USwitch v-model="state.admin" label="Platform admin" description="Akses penuh ke semua company, solution, dan user." />
        <div class="flex justify-end gap-2 pt-2">
          <UButton color="neutral" variant="ghost" @click="open = false">Batal</UButton>
          <UButton type="submit" :loading="saving">Buat user</UButton>
        </div>
      </UForm>
    </template>
  </UModal>
</template>
