<script setup lang="ts">
const auth = useAuthStore()
const toast = useToast()
const busy = ref(false)

const items = computed(() => auth.companies.map(c => ({ label: c.name, value: c.id })))

async function onChange(id: string | number | undefined) {
  const companyId = String(id ?? '')
  if (!companyId || companyId === auth.activeCompanyId) return
  busy.value = true
  try {
    await auth.switchCompany(companyId)
    toast.add({ title: 'Company aktif diganti', color: 'success' })
    await navigateTo('/')
  } catch (err) {
    const e = err as { message?: string }
    toast.add({ title: e.message || 'Gagal mengganti company', color: 'error' })
  } finally {
    busy.value = false
  }
}
</script>

<template>
  <USelect
    v-if="auth.companies.length"
    :model-value="auth.activeCompanyId ?? undefined"
    :items="items"
    :disabled="busy"
    placeholder="Pilih company"
    class="w-56"
    @update:model-value="onChange"
  />
  <span v-else class="text-xs text-muted">Belum ada company</span>
</template>
