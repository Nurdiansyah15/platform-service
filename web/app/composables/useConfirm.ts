import { ConfirmModal } from '#components'

export interface ConfirmOptions {
  title: string
  description?: string
  confirmLabel?: string
  /** danger = tombol merah untuk aksi destruktif. */
  danger?: boolean
  /** Bila diisi, user wajib mengetik teks ini sebelum bisa konfirmasi (aksi tak bisa dibatalkan). */
  typeToConfirm?: string
}

/** Pengganti window.confirm berbasis UModal. `await confirm({...})` → true bila disetujui. */
export function useConfirm() {
  const overlay = useOverlay()
  const modal = overlay.create(ConfirmModal)

  async function confirm(opts: ConfirmOptions): Promise<boolean> {
    const instance = modal.open(opts)
    return (await instance.result) === true
  }

  return { confirm }
}
