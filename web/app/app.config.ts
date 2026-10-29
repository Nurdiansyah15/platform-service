export default defineAppConfig({
  ui: {
    colors: {
      primary: 'indigo',
      neutral: 'zinc',
    },
    button: {
      defaultVariants: { size: 'md' },
    },
    card: {
      slots: { root: 'rounded-xl' },
    },
  },
})
