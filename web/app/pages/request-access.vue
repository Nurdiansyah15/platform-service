<script setup lang="ts">
definePageMeta({ layout: 'auth' })

const name = ref('')
const email = ref('')
const company = ref('')
const message = ref('')
const submitted = ref(false)

function onSubmit() {
  submitted.value = true
}
</script>

<template>
  <UCard>
    <div class="space-y-5">
      <div>
        <h1 class="text-lg font-semibold text-highlighted">Ajukan akses</h1>
        <p class="mt-1 text-sm text-muted">
          Akun platform dibuat oleh admin. Kirim permintaan, tim platform akan menghubungimu.
        </p>
      </div>

      <div v-if="submitted">
        <UAlert
          color="success"
          variant="subtle"
          icon="i-lucide-circle-check"
          title="Permintaan tercatat"
          description="Ini demo tanpa backend: hubungi admin platform secara langsung untuk pembuatan akun."
        />
        <UButton class="mt-4" variant="outline" color="neutral" block @click="submitted = false">
          Kirim lagi
        </UButton>
      </div>

      <form v-else class="space-y-4" @submit.prevent="onSubmit">
        <UFormField label="Nama" name="name">
          <UInput v-model="name" class="w-full" placeholder="Nama lengkap" />
        </UFormField>
        <UFormField label="Email" name="email">
          <UInput v-model="email" type="email" class="w-full" placeholder="nama@perusahaan.com" />
        </UFormField>
        <UFormField label="Perusahaan" name="company">
          <UInput v-model="company" class="w-full" placeholder="Nama perusahaan" />
        </UFormField>
        <UFormField label="Pesan" name="message">
          <UTextarea v-model="message" :rows="3" class="w-full" placeholder="Kebutuhan akses..." />
        </UFormField>
        <UButton type="submit" block color="primary">Kirim permintaan</UButton>
      </form>

      <p class="text-center text-sm text-muted">
        Sudah punya akun?
        <NuxtLink to="/login" class="text-primary hover:underline">Masuk</NuxtLink>
      </p>
    </div>
  </UCard>
</template>
