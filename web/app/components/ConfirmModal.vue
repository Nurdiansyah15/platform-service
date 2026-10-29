<script setup lang="ts">
const props = defineProps<{
  title: string
  description?: string
  confirmLabel?: string
  danger?: boolean
  typeToConfirm?: string
}>()
const emit = defineEmits<{ close: [value: boolean] }>()

const typed = ref('')
const canConfirm = computed(() => !props.typeToConfirm || typed.value.trim() === props.typeToConfirm)
</script>

<template>
  <UModal :title="title" :description="description" :close="false" @update:open="(v: boolean) => !v && emit('close', false)">
    <template v-if="typeToConfirm" #body>
      <UFormField :label="`Ketik &quot;${typeToConfirm}&quot; untuk melanjutkan`">
        <UInput v-model="typed" class="w-full" autofocus @keydown.enter="canConfirm && emit('close', true)" />
      </UFormField>
    </template>
    <template #footer>
      <div class="flex w-full justify-end gap-2">
        <UButton color="neutral" variant="ghost" @click="emit('close', false)">Batal</UButton>
        <UButton :color="danger ? 'error' : 'primary'" :disabled="!canConfirm" @click="emit('close', true)">
          {{ confirmLabel ?? 'Lanjutkan' }}
        </UButton>
      </div>
    </template>
  </UModal>
</template>
