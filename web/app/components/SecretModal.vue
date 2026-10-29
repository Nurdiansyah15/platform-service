<script setup lang="ts">
// Menampilkan kredensial yang hanya terlihat sekali; tidak bisa ditutup sebelum user
// menyatakan sudah menyimpannya.
defineProps<{ title: string; clientId: string; secret: string }>()
const emit = defineEmits<{ close: [] }>()
const saved = ref(false)
</script>

<template>
  <UModal :title="title" :dismissible="false" :close="false">
    <template #body>
      <div class="space-y-4">
        <UAlert
          color="warning"
          variant="subtle"
          icon="i-lucide-key-round"
          title="Simpan secret sekarang"
          description="Secret hanya ditampilkan sekali. Setelah ditutup, secret tidak bisa dilihat lagi (rotate untuk membuat yang baru)."
        />
        <UFormField label="client_id">
          <div class="flex items-center rounded-md border border-default px-3 py-2">
            <CopyText :value="clientId" mono />
          </div>
        </UFormField>
        <UFormField label="client_secret">
          <div class="flex items-center rounded-md border border-default px-3 py-2">
            <CopyText :value="secret" mono />
          </div>
        </UFormField>
        <UCheckbox v-model="saved" label="Saya sudah menyimpan secret ini di tempat aman" />
      </div>
    </template>
    <template #footer>
      <div class="flex w-full justify-end">
        <UButton :disabled="!saved" @click="emit('close')">Selesai</UButton>
      </div>
    </template>
  </UModal>
</template>
