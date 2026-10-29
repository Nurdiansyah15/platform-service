<script setup lang="ts">
import type { FormError, FormSubmitEvent } from '@nuxt/ui'
import type { CompanyResponse } from '~/types/api'

const open = defineModel<boolean>('open', { default: false })
const emit = defineEmits<{ created: [company: CompanyResponse] }>()

const api = useApi()
const { toastError, fieldErrors } = useApiError()
const form = useTemplateRef('form')

const state = reactive({ name: '', slug: '' })
const slugTouched = ref(false)
const saving = ref(false)

// Slug mengikuti nama sampai user mengubahnya sendiri.
watch(() => state.name, (name) => {
  if (!slugTouched.value) state.slug = slugify(name)
})
watch(open, (v) => {
  if (v) {
    state.name = ''
    state.slug = ''
    slugTouched.value = false
  }
})

function validate(s: typeof state): FormError[] {
  const errors: FormError[] = []
  if (!s.name.trim()) errors.push({ name: 'name', message: 'Nama wajib diisi' })
  if (!/^[a-z0-9]+(?:-[a-z0-9]+)*$/.test(s.slug)) errors.push({ name: 'slug', message: 'Hanya huruf kecil, angka, dan tanda hubung' })
  return errors
}

async function onSubmit(e: FormSubmitEvent<typeof state>) {
  saving.value = true
  try {
    const res = await api.post<CompanyResponse>('/api/v1/companies', { name: e.data.name.trim(), slug: e.data.slug })
    open.value = false
    emit('created', res.data)
  } catch (err) {
    const fields = fieldErrors(err)
    if (fields.length) form.value?.setErrors(fields)
    else toastError(err, 'Gagal membuat company')
  } finally {
    saving.value = false
  }
}
</script>

<template>
  <UModal v-model:open="open" title="Buat company" description="Company adalah organisasi pelanggan yang memakai solution.">
    <template #body>
      <UForm ref="form" :state="state" :validate="validate" class="space-y-4" @submit="onSubmit">
        <UFormField label="Nama company" name="name" required>
          <UInput v-model="state.name" placeholder="PT Contoh Jaya" class="w-full" autofocus />
        </UFormField>
        <UFormField label="Slug" name="slug" required help="Pengenal unik di URL. Dibuat otomatis dari nama.">
          <UInput v-model="state.slug" placeholder="contoh-jaya" class="w-full" @update:model-value="slugTouched = true" />
        </UFormField>
        <div class="flex justify-end gap-2 pt-2">
          <UButton color="neutral" variant="ghost" @click="open = false">Batal</UButton>
          <UButton type="submit" :loading="saving">Buat company</UButton>
        </div>
      </UForm>
    </template>
  </UModal>
</template>
