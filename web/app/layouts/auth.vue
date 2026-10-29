<template>
  <div class="flex min-h-dvh items-center justify-center bg-elevated/40 p-4">
    <div class="w-full max-w-md">
      <div class="mb-8 flex items-center justify-center gap-2.5">
        <div class="flex size-9 items-center justify-center rounded-lg bg-primary text-inverted">
          <UIcon name="i-lucide-hexagon" class="size-5" />
        </div>
        <span class="text-lg font-semibold text-highlighted">Platform Console</span>
      </div>
      <slot />
    </div>
  </div>
</template>
