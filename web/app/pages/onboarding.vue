<script setup lang="ts">
const auth = useAuthStore()
const api = useApi()
const toast = useToast()

const loading = ref(false)
const busyId = ref<string | null>(null)

const form = reactive({ name: '', slug: '' })

async function selectCompany(id: string) {
  busyId.value = id
  try {
    await auth.switchCompany(id)
    await navigateTo('/')
  } catch (err) {
    toast.add({ title: (err as { message?: string }).message || 'Gagal memilih company', color: 'error' })
  } finally {
    busyId.value = null
  }
}

async function createCompany() {
  loading.value = true
  try {
    const res = await api.post<{ id: string; name: string }>('/api/v1/companies', {
      name: form.name,
      slug: form.slug,
    })
    const company = res.data

    if (auth.isPlatformAdmin) {
      toast.add({ title: 'Company dibuat', color: 'success' })
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
    await auth.switchCompany(company.id)
    await navigateTo('/')
  } catch (err) {
    toast.add({ title: (err as { message?: string }).message || 'Gagal membuat company', color: 'error' })
  } finally {
    loading.value = false
  }
}
</script>

<template>
  <div class="mx-auto max-w-3xl">
    <PageHeader title="Pilih company" subtitle="Aktifkan company untuk sesi ini atau buat company baru." />

    <UCard class="mb-6">
      <p class="mb-4 text-sm font-medium text-highlighted">Company yang bisa dipilih</p>
      <div v-if="auth.companies.length" class="space-y-2">
        <div
          v-for="c in auth.companies"
          :key="c.id"
          class="flex items-center justify-between rounded-md border border-default px-4 py-3"
        >
          <div>
            <p class="font-medium text-highlighted">{{ c.name }}</p>
            <p class="text-xs text-muted">{{ c.slug }} · {{ c.status }}</p>
          </div>
          <UButton
            size="sm"
            :disabled="c.is_active"
            :loading="busyId === c.id"
            @click="selectCompany(c.id)"
          >
            {{ c.is_active ? 'Aktif' : 'Aktifkan' }}
          </UButton>
        </div>
      </div>
      <p v-else class="text-sm text-muted">
        Belum ada company yang bisa dipilih untuk solution ini.
      </p>

      <div v-if="auth.isPlatformAdmin" class="mt-4">
        <UButton to="/companies" color="neutral" variant="outline" icon="i-lucide-arrow-right">
          Lanjut ke Companies (admin)
        </UButton>
      </div>
    </UCard>

    <UCard>
      <p class="mb-4 text-sm font-medium text-highlighted">Buat company baru</p>
      <form class="grid gap-4 sm:grid-cols-2" @submit.prevent="createCompany">
        <UFormField label="Nama">
          <UInput v-model="form.name" class="w-full" placeholder="PT Contoh Jaya" />
        </UFormField>
        <UFormField label="Slug">
          <UInput v-model="form.slug" class="w-full" placeholder="contoh-jaya" />
        </UFormField>
        <div class="sm:col-span-2">
          <UButton type="submit" :loading="loading" color="primary">Buat company</UButton>
        </div>
      </form>
    </UCard>
  </div>
</template>
