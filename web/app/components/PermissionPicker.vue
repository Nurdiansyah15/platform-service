<script setup lang="ts">
import type { PermissionResponse } from '~/types/api'
import type { PermissionGroupView } from '~/utils/permissions'

type Group = PermissionGroupView<PermissionResponse>
type Row = { kind: 'group'; group: Group } | { kind: 'item'; item: PermissionResponse; depth: number }

// Checklist permission role dikelompokkan per grup katalog — bertingkat mengikuti hierarki
// grup (fallback: prefix code) — dengan pencarian dan pilih-semua per grup (termasuk
// subgrup). Setiap permission terpilih punya scope tempat ia berlaku untuk role ini (Seluruh
// company / Cabang / scope custom). Yang ditampilkan nama + deskripsi; code mentah hanya bila
// "Tampilkan kode" aktif. readonly = hanya yang terpilih.
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
      || (p.group?.name ?? '').toLowerCase().includes(q)
      || (p.group?.parents ?? []).some(a => a.name.toLowerCase().includes(q))),
  )
  return groupPermissions(visible)
})

/** Isi kartu grup root: permission langsung, lalu subgrup (header + isinya) berjenjang. */
function rowsOf(g: Group): Row[] {
  return [
    ...g.items.map(item => ({ kind: 'item' as const, item, depth: g.depth })),
    ...g.children.flatMap(c => [{ kind: 'group' as const, group: c }, ...rowsOf(c)]),
  ]
}

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

function groupState(g: Group): boolean | 'indeterminate' {
  const items = groupAllItems(g)
  const n = items.filter(p => isOn(p.code)).length
  return n === 0 ? false : n === items.length ? true : 'indeterminate'
}

function countOn(g: Group) {
  return groupAllItems(g).filter(p => isOn(p.code)).length
}

function toggleGroup(g: Group) {
  const items = groupAllItems(g)
  const all = groupState(g) === true
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
          <span class="text-xs text-muted">{{ countOn(g) }}/{{ groupAllItems(g).length }}</span>
          <UCheckbox
            v-if="!readonly"
            :model-value="groupState(g)"
            :aria-label="`Pilih semua di grup ${g.label}`"
            @update:model-value="toggleGroup(g)"
          />
        </div>
      </div>
      <ul class="divide-y divide-default">
        <template v-for="row in rowsOf(g)" :key="row.kind === 'group' ? row.group.key : row.item.id">
          <li
            v-if="row.kind === 'group'"
            class="flex items-center justify-between gap-3 bg-elevated/25 py-1.5 pr-4"
            :style="{ paddingLeft: `${1 + (row.group.depth - 1) * 1.5}rem` }"
          >
            <span class="flex items-center gap-1.5 text-xs font-semibold text-toned">
              <UIcon name="i-lucide-corner-down-right" class="size-3.5 text-dimmed" />
              {{ row.group.label }}
            </span>
            <div class="flex items-center gap-3">
              <span class="text-xs text-muted">{{ countOn(row.group) }}/{{ groupAllItems(row.group).length }}</span>
              <UCheckbox
                v-if="!readonly"
                :model-value="groupState(row.group)"
                :aria-label="`Pilih semua di grup ${row.group.label}`"
                @update:model-value="toggleGroup(row.group)"
              />
            </div>
          </li>
          <li
            v-else
            class="flex flex-wrap items-start gap-3 py-2.5 pr-4"
            :style="{ paddingLeft: `${1 + row.depth * 1.5}rem` }"
          >
            <label class="flex min-w-0 flex-1 items-start gap-3" :class="readonly ? '' : 'cursor-pointer'">
              <UCheckbox
                v-if="!readonly"
                :model-value="isOn(row.item.code)"
                class="mt-0.5"
                @update:model-value="(v: boolean | 'indeterminate') => toggle(row.item.code, v === true)"
              />
              <UIcon v-else name="i-lucide-check" class="mt-0.5 size-4 text-primary" />
              <span class="min-w-0 flex-1">
                <span class="block text-sm text-highlighted">{{ row.item.name }}</span>
                <span v-if="row.item.description" class="block text-xs text-muted">{{ row.item.description }}</span>
                <span v-if="showCode" class="block font-mono text-xs text-dimmed">{{ row.item.code }}</span>
              </span>
            </label>
            <div v-if="isOn(row.item.code)" class="flex items-center gap-2">
              <span class="text-xs text-muted">Berlaku di</span>
              <USelect
                v-if="!readonly"
                :model-value="selected[row.item.code]"
                :items="scopeOptions"
                size="xs"
                class="w-40"
                @update:model-value="(v: string) => setScope(row.item.code, v)"
              />
              <UBadge v-else :label="scopeLabel(selected[row.item.code]!)" color="neutral" variant="soft" size="sm" />
            </div>
            <StatusBadge v-if="row.item.status !== 'active'" :status="row.item.status" size="xs" />
          </li>
        </template>
      </ul>
    </div>
  </div>
</template>
