<script setup lang="ts">
const props = defineProps<{ value: string; label?: string; mono?: boolean }>()
const toast = useToast()
const copied = ref(false)

async function copy() {
  try {
    await navigator.clipboard.writeText(props.value)
    copied.value = true
    setTimeout(() => (copied.value = false), 1500)
  } catch {
    toast.add({ title: 'Gagal menyalin', color: 'error' })
  }
}
</script>

<template>
  <span class="inline-flex min-w-0 items-center gap-1">
    <span :class="mono ? 'font-mono text-xs' : ''" class="truncate">{{ label ?? value }}</span>
    <UTooltip :text="copied ? 'Tersalin' : 'Salin'">
      <UButton
        :icon="copied ? 'i-lucide-check' : 'i-lucide-copy'"
        color="neutral"
        variant="ghost"
        size="xs"
        :aria-label="`Salin ${label ?? value}`"
        @click.stop="copy"
      />
    </UTooltip>
  </span>
</template>
