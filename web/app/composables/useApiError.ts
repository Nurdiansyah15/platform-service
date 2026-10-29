import type { AppError } from '~/types/api'

/** Penanganan error API yang seragam: toast pesan terjemahan + akses error per field. */
export function useApiError() {
  const toast = useToast()

  function messageOf(err: unknown, fallback = 'Terjadi kesalahan. Coba lagi.'): string {
    return (err as Partial<AppError>)?.message || fallback
  }

  function toastError(err: unknown, title?: string) {
    const msg = messageOf(err)
    toast.add({
      title: title ?? msg,
      description: title ? msg : undefined,
      color: 'error',
      icon: 'i-lucide-circle-alert',
    })
  }

  /** Error per field untuk UForm (`form.setErrors`); kosong bila error bukan validasi. */
  function fieldErrors(err: unknown): { name: string; message: string }[] {
    const fields = (err as Partial<AppError>)?.fields ?? {}
    return Object.entries(fields).map(([name, message]) => ({ name, message }))
  }

  return { messageOf, toastError, fieldErrors }
}
