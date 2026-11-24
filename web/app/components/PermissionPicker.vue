<script setup lang="ts">
import type { PermissionResponse } from '~/types/api'

// Checklist permission role dikelompokkan per grup katalog (fallback: prefix code), dengan
// pencarian dan pilih-semua per grup. Setiap permission terpilih punya scope tempat ia
// berlaku untuk role ini (Seluruh company / Cabang / scope custom). Yang ditampilkan nama +
// deskripsi; code mentah hanya bila "Tampilkan kode" aktif. readonly = hanya yang terpilih.
const props = defineProps<{
  catalog: PermissionResponse[]
  readonly?: boolean
  /** Pilihan scope (value = scope_type). Elemen pertama = default saat permission dicentang. */
  scopeOptions: { label: string; value: string }[]
}>()
/** code → scope_type. */
const selected = defineModel<Record<string, string>>({ required: true })

const search = ref('')
const showCode = useShowPermissionCode()
const defaultScope = computed(() => props.scopeOptions[0]?.value ?? 'company')

const groups = computed(() => {
  const q = search.value.trim().toLowerCase()
  const visible = props.catalog.filter(p =>
    (!props.readonly || p.code in selected.value)
    && (!q
      || p.code.toLowerCase().includes(q)
      || p.name.toLowerCase().includes(q)
      || (p.description ?? '').toLowerCase().includes(q)
      || (p.group?.name ?? '').toLowerCase().includes(q)),
  )
  return groupPermissions(visible)
})

function isOn(code: string) {
  return code in selected.value
}

function toggle(code: string, on: boolean) {
  if (props.readonly) return
  const next = { ...selected.value }
  if (on) next[code] = next[code] ?? defaultScope.value
  else delete next[code]
  selected.value = next
}

function setScope(code: string, scope: string) {
  selected.value = { ...selected.value, [code]: scope }
}

function groupState(items: PermissionResponse[]): boolean | 'indeterminate' {
  const n = items.filter(p => isOn(p.code)).length
  return n === 0 ? false : n === items.length ? true : 'indeterminate'
}

function toggleGroup(items: PermissionResponse[]) {
  const all = groupState(items) === true
  const next = { ...selected.value }
  for (const p of items) {
    if (all) delete next[p.code]
    else next[p.code] = next[p.code] ?? defaultScope.value
  }
  selected.value = next
}

function scopeLabel(value: string) {
  return props.scopeOptions.find(o => o.value === value)?.label ?? value
}
</script>

<template>
  <div class="space-y-4">
    <div class="flex flex-wrap items-center justify-between gap-3">
      <UInput v-model="search" icon="i-lucide-search" placeholder="Cari permission..." class="w-full sm:w-80" />
      <USwitch v-model="showCode" label="Tampilkan kode" size="sm" />
    </div>

    <UEmpty
      v-if="!groups.length"
      icon="i-lucide-key-round"
      :title="readonly ? 'Belum ada permission' : 'Tidak ada permission'"
      :description="search ? 'Tidak ada yang cocok dengan pencarian.' : readonly ? 'Role ini belum diberi permission.' : 'Katalog permission solution masih kosong.'"
      variant="naked"
    />

    <div v-for="g in groups" :key="g.key" class="rounded-lg border border-default">
      <div class="flex items-center justify-between gap-3 border-b border-default bg-elevated/50 px-4 py-2">
        <span class="text-sm font-semibold text-highlighted">{{ g.label }}</span>
        <div class="flex items-center gap-3">
          <span class="text-xs text-muted">{{ g.items.filter(p => isOn(p.code)).length }}/{{ g.items.length }}</span>
          <UCheckbox
            v-if="!readonly"
            :model-value="groupState(g.items)"
            :aria-label="`Pilih semua di grup ${g.label}`"
            @update:model-value="toggleGroup(g.items)"
          />
        </div>
      </div>
      <ul class="divide-y divide-default">
        <li v-for="p in g.items" :key="p.id" class="flex flex-wrap items-start gap-3 px-4 py-2.5">
          <label class="flex min-w-0 flex-1 items-start gap-3" :class="readonly ? '' : 'cursor-pointer'">
            <UCheckbox
              v-if="!readonly"
              :model-value="isOn(p.code)"
              class="mt-0.5"
              @update:model-value="(v: boolean | 'indeterminate') => toggle(p.code, v === true)"
            />
            <UIcon v-else name="i-lucide-check" class="mt-0.5 size-4 text-primary" />
            <span class="min-w-0 flex-1">
              <span class="block text-sm text-highlighted">{{ p.name }}</span>
              <span v-if="p.description" class="block text-xs text-muted">{{ p.description }}</span>
              <span v-if="showCode" class="block font-mono text-xs text-dimmed">{{ p.code }}</span>
            </span>
          </label>
          <div v-if="isOn(p.code)" class="flex items-center gap-2">
            <span class="text-xs text-muted">Berlaku di</span>
            <USelect
              v-if="!readonly"
              :model-value="selected[p.code]"
              :items="scopeOptions"
              size="xs"
              class="w-40"
              @update:model-value="(v: string) => setScope(p.code, v)"
            />
            <UBadge v-else :label="scopeLabel(selected[p.code]!)" color="neutral" variant="soft" size="sm" />
          </div>
          <StatusBadge v-if="p.status !== 'active'" :status="p.status" size="xs" />
        </li>
      </ul>
    </div>
  </div>
</template>
