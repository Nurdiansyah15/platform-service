<script setup lang="ts">
import type { MemberCandidate } from '~/types/api'

// Autocomplete user terdaftar (email / nama, min. 3 karakter) yang belum member company.
const props = defineProps<{ companyId: string }>()
const model = defineModel<MemberCandidate | undefined>()

const api = useApi()
const term = ref('')
const items = ref<MemberCandidate[]>([])
const loading = ref(false)

let timer: ReturnType<typeof setTimeout> | undefined
let seq = 0
watch(term, (value) => {
  clearTimeout(timer)
  const q = value.trim()
  if (q.length < 3) {
    items.value = []
    return
  }
  timer = setTimeout(async () => {
    const mySeq = ++seq
    loading.value = true
    try {
      const res = await api.get<MemberCandidate[]>(`/api/v1/companies/${props.companyId}/member-candidates`, {
        query: { search: q },
      })
      if (mySeq === seq) items.value = res.data
    } catch {
      if (mySeq === seq) items.value = []
    } finally {
      if (mySeq === seq) loading.value = false
    }
  }, 250)
})
</script>

<template>
  <UInputMenu
    v-model="model"
    v-model:search-term="term"
    :items="items"
    label-key="name"
    description-key="email"
    :loading="loading"
    ignore-filter
    icon="i-lucide-search"
    placeholder="Ketik nama atau email (min. 3 huruf)"
    class="w-full"
  >
    <template #empty>
      <span class="text-sm text-muted">
        {{ term.trim().length < 3 ? 'Ketik minimal 3 huruf' : 'Tidak ada user yang cocok' }}
      </span>
    </template>
  </UInputMenu>
</template>
