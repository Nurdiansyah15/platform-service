<script setup lang="ts">
import type { CompanyResponse } from '~/types/api'

definePageMeta({ layout: 'auth' })

const auth = useAuthStore()
const route = useRoute()
const { toastError } = useApiError()

const busyId = ref<string | null>(null)
const showCreate = ref(route.query.create === '1')

async function selectCompany(id: string) {
  busyId.value = id
  try {
    await auth.switchCompany(id)
    await navigateTo(`/companies/${id}`)
  } catch (err) {
    toastError(err, 'Gagal memilih company')
  } finally {
    busyId.value = null
  }
}

async function onCreated(company: CompanyResponse) {
  await auth.loadCompanies()
  if (auth.isPlatformAdmin) {
    await navigateTo(`/companies/${company.id}`)
    return
  }
  // GATE-SUBSCRIPTION (nonaktif sementara): company baru langsung bisa dipilih tanpa
  // subscription, jadi fallback "butuh subscription" tidak dipakai.
  // try {
  //   await auth.switchCompany(company.id)
  //   await navigateTo('/')
  // } catch {
  //   toast.add({
  //     title: 'Company dibuat',
  //     description: 'Company belum bisa diaktifkan (butuh subscription). Hubungi admin platform.',
  //     color: 'warning',
  //   })
  //   await auth.loadCompanies()
  // }
  await selectCompany(company.id)
}

async function logout() {
  await auth.logout()
  await navigateTo('/login')
}

onMounted(() => auth.loadCompanies())
</script>

<template>
  <UCard :ui="{ body: 'p-6 sm:p-8' }">
    <div class="space-y-6">
      <div class="space-y-1 text-center">
        <h1 class="text-xl font-semibold text-highlighted">Halo, {{ auth.user?.name }}</h1>
        <p class="text-sm text-muted">Pilih company yang ingin kamu kelola.</p>
      </div>

      <div v-if="auth.companies.length" class="space-y-2">
        <button
          v-for="c in auth.companies"
          :key="c.id"
          type="button"
          class="flex w-full items-center gap-3 rounded-lg border border-default p-3 text-left transition hover:border-primary hover:bg-primary/5 disabled:opacity-60"
          :disabled="!!busyId"
          @click="selectCompany(c.id)"
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
      <UEmpty
        v-else
        icon="i-lucide-building-2"
        title="Belum ada company"
        description="Buat company baru, atau minta admin company-mu menambahkanmu sebagai anggota."
        variant="naked"
      />

      <div class="space-y-2">
        <UButton block icon="i-lucide-plus" :variant="auth.companies.length ? 'outline' : 'solid'" @click="showCreate = true">
          Buat company baru
        </UButton>
        <UButton v-if="auth.isPlatformAdmin" block color="neutral" variant="ghost" to="/">Ke dashboard admin</UButton>
        <UButton block color="neutral" variant="ghost" icon="i-lucide-log-out" @click="logout">Keluar</UButton>
      </div>
    </div>

    <CompanyCreateModal v-model:open="showCreate" @created="onCreated" />
  </UCard>
</template>
