type BadgeColor = 'success' | 'warning' | 'error' | 'neutral' | 'info' | 'primary'

interface StatusMeta {
  label: string
  color: BadgeColor
}

/** Satu sumber label & warna status untuk semua entitas (user, company, member, branch, solution, client). */
const STATUS: Record<string, StatusMeta> = {
  active: { label: 'Aktif', color: 'success' },
  invited: { label: 'Diundang', color: 'info' },
  inactive: { label: 'Nonaktif', color: 'neutral' },
  suspended: { label: 'Ditangguhkan', color: 'warning' },
  closed: { label: 'Ditutup', color: 'error' },
  revoked: { label: 'Dicabut', color: 'error' },
  deprecated: { label: 'Usang', color: 'warning' },
}

export function statusMeta(status?: string | null): StatusMeta {
  return STATUS[status ?? ''] ?? { label: status || '—', color: 'neutral' }
}

export function statusOptions(statuses: string[], allLabel = 'Semua status') {
  return [
    { label: allLabel, value: undefined as string | undefined },
    ...statuses.map(s => ({ label: statusMeta(s).label, value: s as string | undefined })),
  ]
}
