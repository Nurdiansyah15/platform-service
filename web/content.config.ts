import { defineCollection, defineContentConfig } from '@nuxt/content'

// Dokumentasi integrasi solution: content/docs/**/*.md → /docs/...
export default defineContentConfig({
  collections: {
    docs: defineCollection({
      type: 'page',
      source: 'docs/**/*.md',
    }),
  },
})
