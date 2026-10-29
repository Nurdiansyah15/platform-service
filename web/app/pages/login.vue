<script setup lang="ts">
import type { FormError, FormSubmitEvent } from '@nuxt/ui'

definePageMeta({ layout: 'auth' })

const auth = useAuthStore()
const route = useRoute()
const config = useRuntimeConfig()

const state = reactive({ email: '', password: '' })
const showPassword = ref(false)
const loading = ref(false)
const error = ref('')

function validate(s: typeof state): FormError[] {
  const errors: FormError[] = []
  if (!s.email.trim()) errors.push({ name: 'email', message: 'Email wajib diisi' })
  else if (!/^\S+@\S+\.\S+$/.test(s.email.trim())) errors.push({ name: 'email', message: 'Format email tidak valid' })
  if (!s.password) errors.push({ name: 'password', message: 'Password wajib diisi' })
  return errors
}

async function onSubmit(e: FormSubmitEvent<typeof state>) {
  error.value = ''
  loading.value = true
  try {
    await auth.login(e.data.email.trim(), e.data.password, String(config.public.defaultSolution || ''))
    const redirect = typeof route.query.redirect === 'string' ? route.query.redirect : '/'
    await navigateTo(redirect)
  } catch (err) {
    error.value = (err as { message?: string }).message || 'Login gagal'
  } finally {
    loading.value = false
  }
}
</script>

<template>
  <UCard :ui="{ body: 'p-6 sm:p-8' }">
    <div class="space-y-6">
      <div class="space-y-1 text-center">
        <h1 class="text-xl font-semibold text-highlighted">Masuk ke akunmu</h1>
        <p class="text-sm text-muted">Kelola company, anggota, dan hak akses.</p>
      </div>

      <UAlert v-if="error" color="error" variant="subtle" icon="i-lucide-circle-alert" :title="error" />

      <UForm :state="state" :validate="validate" :validate-on="['blur']" class="space-y-4" @submit="onSubmit">
        <UFormField label="Email" name="email">
          <UInput v-model="state.email" type="email" autocomplete="username" placeholder="nama@perusahaan.com" class="w-full" autofocus />
        </UFormField>
        <UFormField label="Password" name="password">
          <UInput
            v-model="state.password"
            :type="showPassword ? 'text' : 'password'"
            autocomplete="current-password"
            placeholder="••••••••"
            class="w-full"
            :ui="{ trailing: 'pe-1' }"
          >
            <template #trailing>
              <UButton
                color="neutral"
                variant="link"
                size="sm"
                :icon="showPassword ? 'i-lucide-eye-off' : 'i-lucide-eye'"
                :aria-label="showPassword ? 'Sembunyikan password' : 'Tampilkan password'"
                @click="showPassword = !showPassword"
              />
            </template>
          </UInput>
        </UFormField>
        <UButton type="submit" block size="lg" :loading="loading">Masuk</UButton>
      </UForm>

      <p class="text-center text-sm text-muted">
        Belum punya akun? Hubungi admin platform di perusahaanmu.
      </p>
      <p class="text-center text-sm">
        <ULink to="/docs" class="text-primary">Dokumentasi integrasi solution</ULink>
      </p>
    </div>
  </UCard>
</template>
