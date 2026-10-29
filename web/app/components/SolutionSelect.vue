<script setup lang="ts">
import type { SolutionResponse } from '~/types/api'

// Pilihan solution; tersimpan di query URL (?solution=) agar tetap terpilih saat kembali.
// Default: solution pertama selain console (role produk yang biasanya dicari).
const model = defineModel<string | undefined>()
const emit = defineEmits<{ loaded: [solutions: SolutionResponse[]] }>()

const api = useApi()
const auth = useAuthStore()
const route = useRoute()
const router = useRouter()
const solutions = ref<SolutionResponse[]>([])
const loading = ref(false)

const items = computed(() => solutions.value.map(s => ({ label: s.name, value: s.id, description: s.code })))

onMounted(async () => {
  loading.value = true
  try {
    const res = await api.get<SolutionResponse[]>('/api/v1/solutions', { query: { limit: 100 } })
    solutions.value = res.data
    emit('loaded', res.data)
    const fromQuery = typeof route.query.solution === 'string' ? route.query.solution : undefined
    const valid = (id?: string) => !!id && res.data.some(s => s.id === id)
    if (valid(fromQuery)) model.value = fromQuery
    else if (!valid(model.value)) model.value = (res.data.find(s => s.id !== auth.solution?.id) ?? res.data[0])?.id
  } finally {
    loading.value = false
  }
})

watch(model, (id) => {
  if (id && route.query.solution !== id) router.replace({ query: { ...route.query, solution: id } })
})
</script>

<template>
  <USelectMenu
    v-model="model"
    :items="items"
    value-key="value"
    :loading="loading"
    :search-input="{ placeholder: 'Cari solution...' }"
    icon="i-lucide-boxes"
    placeholder="Pilih solution"
    class="w-full sm:w-72"
  />
</template>
