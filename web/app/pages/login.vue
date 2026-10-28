<script setup lang="ts">
definePageMeta({ layout: 'auth' })

const auth = useAuthStore()
const route = useRoute()
const config = useRuntimeConfig()

const email = ref('')
const password = ref('')
const solution = ref(String(config.public.defaultSolution || ''))
const showAdvanced = ref(false)
const loading = ref(false)
const error = ref('')

async function onSubmit() {
  error.value = ''
  loading.value = true
  try {
    await auth.login(email.value, password.value, solution.value)
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
  <UCard>
    <div class="space-y-5">
      <div>
        <h1 class="text-lg font-semibold text-highlighted">Masuk</h1>
        <p class="mt-1 text-sm text-muted">Gunakan akun platform-mu untuk mengelola tenant.</p>
      </div>

      <UAlert v-if="error" color="error" variant="subtle" icon="i-lucide-circle-alert" :title="error" />

      <form class="space-y-4" @submit.prevent="onSubmit">
        <UFormField label="Email" name="email">
          <UInput v-model="email" type="email" autocomplete="username" placeholder="admin@platform.local" class="w-full" />
        </UFormField>

        <UFormField label="Password" name="password">
          <UInput v-model="password" type="password" autocomplete="current-password" placeholder="••••••••" class="w-full" />
        </UFormField>

        <div>
          <button
            type="button"
            class="text-xs text-muted underline-offset-2 hover:text-highlighted hover:underline"
            @click="showAdvanced = !showAdvanced"
          >
            {{ showAdvanced ? 'Sembunyikan' : 'Ubah' }} konteks solution
          </button>
          <UFormField v-if="showAdvanced" label="Solution (audience)" name="solution" class="mt-2">
            <UInput v-model="solution" placeholder="platform-console" class="w-full" />
          </UFormField>
        </div>

        <UButton type="submit" block :loading="loading" color="primary">Masuk</UButton>
      </form>

      <p class="text-center text-sm text-muted">
        Belum punya akses?
        <NuxtLink to="/request-access" class="text-primary hover:underline">Ajukan akses</NuxtLink>
      </p>
    </div>
  </UCard>
</template>
