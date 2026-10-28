<script setup lang="ts">
const auth = useAuthStore()
</script>

<template>
  <div>
    <PageHeader
      :title="`Halo, ${auth.user?.name ?? 'Pengguna'}`"
      subtitle="Ringkasan sesi dan konteks platform."
    />

    <UAlert
      v-if="!auth.hasCompany && auth.isPlatformAdmin"
      class="mb-4"
      color="info"
      variant="subtle"
      icon="i-lucide-info"
      title="Belum ada company aktif"
      description="Sebagai platform admin kamu bisa tetap mengelola company mana pun dari menu Companies."
    >
      <template #actions>
        <UButton to="/companies" size="sm" color="neutral" variant="outline">Ke Companies</UButton>
      </template>
    </UAlert>

    <UAlert
      v-else-if="!auth.hasCompany"
      class="mb-4"
      color="warning"
      variant="subtle"
      icon="i-lucide-triangle-alert"
      title="Belum memilih company"
      description="Pilih company aktif untuk mengakses data per-company."
    >
      <template #actions>
        <UButton to="/onboarding" size="sm" color="neutral" variant="outline">Pilih company</UButton>
      </template>
    </UAlert>

    <div class="grid gap-4 sm:grid-cols-2 lg:grid-cols-3">
      <UCard>
        <p class="text-xs uppercase tracking-wide text-dimmed">User</p>
        <p class="mt-2 font-medium text-highlighted">{{ auth.user?.name }}</p>
        <p class="text-sm text-muted">{{ auth.user?.email }}</p>
        <UBadge
          v-if="auth.isPlatformAdmin"
          class="mt-3"
          color="primary"
          variant="subtle"
          size="sm"
        >
          Platform Admin
        </UBadge>
      </UCard>

      <UCard>
        <p class="text-xs uppercase tracking-wide text-dimmed">Solution (audience)</p>
        <p class="mt-2 font-medium text-highlighted">{{ auth.solution?.name ?? '-' }}</p>
        <p class="text-sm text-muted">{{ auth.solution?.code ?? '-' }}</p>
      </UCard>

      <UCard>
        <p class="text-xs uppercase tracking-wide text-dimmed">Company aktif</p>
        <p class="mt-2 font-medium text-highlighted">{{ auth.activeCompany?.name ?? '-' }}</p>
        <p class="text-sm text-muted">{{ auth.activeCompany?.slug ?? auth.activeCompanyId ?? '-' }}</p>
      </UCard>
    </div>

    <div class="mt-6 grid gap-4 sm:grid-cols-2 lg:grid-cols-3">
      <NuxtLink to="/companies" class="rounded-lg border border-default bg-default p-4 transition hover:border-primary/40">
        <div class="flex items-center gap-3">
          <UIcon name="i-lucide-building-2" class="size-5 text-primary" />
          <span class="font-medium text-highlighted">Companies</span>
        </div>
        <p class="mt-2 text-sm text-muted">Kelola company, member, dan branch.</p>
      </NuxtLink>
      <NuxtLink to="/solutions" class="rounded-lg border border-default bg-default p-4 transition hover:border-primary/40">
        <div class="flex items-center gap-3">
          <UIcon name="i-lucide-boxes" class="size-5 text-primary" />
          <span class="font-medium text-highlighted">Solutions</span>
        </div>
        <p class="mt-2 text-sm text-muted">Katalog solution dan role global.</p>
      </NuxtLink>
      <NuxtLink to="/authorization/check" class="rounded-lg border border-default bg-default p-4 transition hover:border-primary/40">
        <div class="flex items-center gap-3">
          <UIcon name="i-lucide-shield-check" class="size-5 text-primary" />
          <span class="font-medium text-highlighted">Authorization</span>
        </div>
        <p class="mt-2 text-sm text-muted">Uji permission dan assignment role.</p>
      </NuxtLink>
    </div>
  </div>
</template>
