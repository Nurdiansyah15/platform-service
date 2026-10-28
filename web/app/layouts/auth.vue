<template>
  <div class="flex min-h-screen items-center justify-center bg-elevated/40 p-4">
    <div class="w-full max-w-md">
      <div class="mb-6 flex items-center justify-center gap-2">
        <UIcon name="i-lucide-hexagon" class="size-6 text-primary" />
        <span class="text-lg font-semibold text-highlighted">Platform Console</span>
      </div>
      <slot />
    </div>
  </div>
</template>
