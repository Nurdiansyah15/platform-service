<script setup lang="ts">
import type { AuthorizeRequest, AuthorizeResponse } from '~/types/api'

// SSO console → solution (authorization code + PKCE). Solution mengarahkan user ke sini;
// console (sudah login) menerbitkan kode sekali pakai lalu mengembalikan user ke
// redirect_uri solution. Saat gagal, user TIDAK diarahkan ke redirect_uri.
definePageMeta({ layout: 'auth' })

const auth = useAuthStore()
const route = useRoute()
const api = useApi()

const q = (k: string) => (typeof route.query[k] === 'string' ? (route.query[k] as string) : '')
const params = computed(() => ({
  solution: q('solution'),
  redirect_uri: q('redirect_uri'),
  code_challenge: q('code_challenge'),
  code_challenge_method: q('code_challenge_method') || 'S256',
  state: q('state'),
  company_id: q('company_id'),
}))

const missing = computed(() => !params.value.solution || !params.value.redirect_uri || !params.value.code_challenge)
const error = ref('')
const busyId = ref<string | null>(null)
const redirecting = ref(false)

// Pemilih company hanya bila solution tidak menentukan company, console belum punya
// company aktif, dan user punya lebih dari satu company.
const needsPicker = computed(() =>
  !params.value.company_id && !auth.activeCompanyId && auth.companies.length > 1)

async function authorize(companyId?: string) {
  error.value = ''
  busyId.value = companyId ?? 'auto'
  const p = params.value
  const body: AuthorizeRequest = {
    solution: p.solution,
    redirect_uri: p.redirect_uri,
    code_challenge: p.code_challenge,
    code_challenge_method: p.code_challenge_method as 'S256',
    state: p.state || undefined,
    company_id: companyId || p.company_id || undefined,
  }
  try {
    const res = await api.post<AuthorizeResponse>('/api/v1/auth/authorize', body)
    redirecting.value = true
    window.location.replace(res.data.redirect_to)
  } catch (err) {
    error.value = (err as { message?: string }).message || 'Gagal masuk ke aplikasi'
  } finally {
    busyId.value = null
  }
}

onMounted(async () => {
  if (missing.value) return
  if (!params.value.company_id && !auth.activeCompanyId) await auth.loadCompanies()
  if (!needsPicker.value) await authorize()
})
</script>

<template>
  <UCard :ui="{ body: 'p-6 sm:p-8' }">
    <div class="space-y-6">
      <div class="space-y-1 text-center">
        <h1 class="text-xl font-semibold text-highlighted">Masuk ke aplikasi</h1>
        <p class="text-sm text-muted">
          Sebagai <span class="font-medium text-highlighted">{{ auth.user?.email }}</span>
        </p>
      </div>

      <UAlert
        v-if="missing"
        color="error"
        variant="subtle"
        icon="i-lucide-circle-alert"
        title="Permintaan masuk tidak lengkap"
        description="Parameter solution, redirect_uri, atau code_challenge tidak ada. Buka aplikasi lagi dari launcher."
      />
      <UAlert v-else-if="error" color="error" variant="subtle" icon="i-lucide-circle-alert" :title="error" />

      <div v-if="!missing && needsPicker && !redirecting" class="space-y-2">
        <p class="text-sm text-muted">Pilih company untuk aplikasi ini.</p>
        <button
          v-for="c in auth.companies"
          :key="c.id"
          type="button"
          class="flex w-full items-center gap-3 rounded-lg border border-default p-3 text-left transition hover:border-primary hover:bg-primary/5 disabled:opacity-60"
          :disabled="!!busyId"
          @click="authorize(c.id)"
        >
          <UAvatar :text="initials(c.name)" class="rounded-md" />
          <span class="min-w-0 flex-1">
            <span class="block truncate font-medium text-highlighted">{{ c.name }}</span>
            <span class="block truncate text-xs text-muted">{{ c.slug }}</span>
          </span>
          <UIcon v-if="busyId === c.id" name="i-lucide-loader-circle" class="size-5 animate-spin text-muted" />
          <UIcon v-else name="i-lucide-chevron-right" class="size-5 text-dimmed" />
        </button>
      </div>
      <div v-else-if="!missing && !error" class="flex items-center justify-center gap-2 py-4 text-sm text-muted">
        <UIcon name="i-lucide-loader-circle" class="size-5 animate-spin" />
        Mengalihkan ke aplikasi…
      </div>

      <div v-if="missing || error" class="space-y-2">
        <UButton v-if="error && !missing" block icon="i-lucide-rotate-cw" @click="authorize()">Coba lagi</UButton>
        <UButton block color="neutral" variant="ghost" to="/">Kembali ke console</UButton>
      </div>
    </div>
  </UCard>
</template>
