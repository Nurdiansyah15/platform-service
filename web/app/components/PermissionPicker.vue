<script setup lang="ts">
import type { PermissionResponse } from '~/types/api'

// Checklist permission dikelompokkan per modul (prefix sebelum titik pertama), dengan
// pencarian dan pilih-semua per grup. readonly = hanya menampilkan permission terpilih.
const props = defineProps<{ catalog: PermissionResponse[]; readonly?: boolean }>()
const selected = defineModel<string[]>({ required: true })

const search = ref('')

const groups = computed(() => {
  const q = search.value.trim().toLowerCase()
  const visible = props.catalog.filter(p =>
    (!props.readonly || selected.value.includes(p.code))
    && (!q || p.code.toLowerCase().includes(q) || p.name.toLowerCase().includes(q)),
  )
  const map = new Map<string, PermissionResponse[]>()
  for (const p of visible) {
    const key = p.code.includes('.') ? p.code.split('.')[0]! : 'lainnya'
    map.set(key, [...(map.get(key) ?? []), p])
  }
  return [...map.entries()].sort(([a], [b]) => a.localeCompare(b)).map(([name, items]) => ({ name, items }))
})

function isOn(code: string) {
  return selected.value.includes(code)
}

function toggle(code: string, on: boolean) {
  if (props.readonly) return
  selected.value = on ? [...new Set([...selected.value, code])] : selected.value.filter(c => c !== code)
}

function groupState(items: PermissionResponse[]): boolean | 'indeterminate' {
  const n = items.filter(p => isOn(p.code)).length
  return n === 0 ? false : n === items.length ? true : 'indeterminate'
}

function toggleGroup(items: PermissionResponse[]) {
  const all = groupState(items) === true
  const codes = items.map(p => p.code)
  selected.value = all
    ? selected.value.filter(c => !codes.includes(c))
    : [...new Set([...selected.value, ...codes])]
}
</script>

<template>
  <div class="space-y-4">
    <UInput v-model="search" icon="i-lucide-search" placeholder="Cari permission..." class="w-full sm:w-80" />

    <UEmpty
      v-if="!groups.length"
      icon="i-lucide-key-round"
      :title="readonly ? 'Belum ada permission' : 'Tidak ada permission'"
      :description="search ? 'Tidak ada yang cocok dengan pencarian.' : readonly ? 'Role ini belum diberi permission.' : 'Katalog permission solution masih kosong.'"
      variant="naked"
    />

    <div v-for="g in groups" :key="g.name" class="rounded-lg border border-default">
      <div class="flex items-center justify-between gap-3 border-b border-default bg-elevated/50 px-4 py-2">
        <span class="text-sm font-semibold capitalize text-highlighted">{{ g.name }}</span>
        <div class="flex items-center gap-3">
          <span class="text-xs text-muted">{{ g.items.filter(p => isOn(p.code)).length }}/{{ g.items.length }}</span>
          <UCheckbox
            v-if="!readonly"
            :model-value="groupState(g.items)"
            aria-label="Pilih semua di grup"
            @update:model-value="toggleGroup(g.items)"
          />
        </div>
      </div>
      <ul class="divide-y divide-default">
        <li v-for="p in g.items" :key="p.id">
          <label
            class="flex items-start gap-3 px-4 py-2.5"
            :class="readonly ? '' : 'cursor-pointer hover:bg-elevated/40'"
          >
            <UCheckbox
              v-if="!readonly"
              :model-value="isOn(p.code)"
              class="mt-0.5"
              @update:model-value="(v: boolean | 'indeterminate') => toggle(p.code, v === true)"
            />
            <UIcon v-else name="i-lucide-check" class="mt-0.5 size-4 text-primary" />
            <span class="min-w-0 flex-1">
              <span class="block text-sm text-highlighted">{{ p.name }}</span>
              <span class="block font-mono text-xs text-muted">{{ p.code }}</span>
            </span>
            <StatusBadge v-if="p.status !== 'active'" :status="p.status" size="xs" />
          </label>
        </li>
      </ul>
    </div>
  </div>
</template>
