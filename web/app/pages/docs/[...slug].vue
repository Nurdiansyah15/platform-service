<script setup lang="ts">
definePageMeta({ layout: 'docs' })

const route = useRoute()

const { data: page } = await useAsyncData(`docs-${route.path}`, () =>
  queryCollection('docs').path(route.path).first(),
)
if (!page.value) {
  throw createError({ statusCode: 404, statusMessage: 'Halaman dokumentasi tidak ditemukan', fatal: true })
}

const { data: surround } = await useAsyncData(`docs-surround-${route.path}`, () =>
  queryCollectionItemSurroundings('docs', route.path, { fields: ['description'] }),
)

useHead({ title: () => `${page.value?.title ?? 'Dokumentasi'} · Platform Docs` })
</script>

<template>
  <div v-if="page" class="flex gap-8">
    <article class="min-w-0 flex-1">
      <UPageHeader :title="page.title" :description="page.description" class="mb-6" />
      <ContentRenderer :value="page" />
      <USeparator class="my-8" />
      <UContentSurround :surround="surround ?? []" />
    </article>
    <aside v-if="page.body?.toc?.links?.length" class="hidden w-56 shrink-0 xl:block">
      <UContentToc :links="page.body.toc.links" title="Di halaman ini" class="sticky top-20" />
    </aside>
  </div>
</template>
