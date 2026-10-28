import type { ApiEnvelope, ApiErrorBody, AppError } from '~/types/api'

export function isApiErrorBody(value: unknown): value is ApiErrorBody {
  return !!value && typeof value === 'object' && 'error_code' in (value as Record<string, unknown>)
}

/** Ubah error fetch/ofetch menjadi AppError yang seragam untuk UI. */
export function toAppError(err: unknown): AppError {
  const e = err as {
    response?: { status?: number }
    data?: ApiErrorBody
    message?: string
  }
  const status = e?.response?.status ?? 0
  const body = e?.data
  const fields: Record<string, string> = {}
  let message = e?.message || 'Terjadi kesalahan'

  if (body) {
    if (typeof body.errors === 'string') {
      message = body.errors
    } else if (body.errors && typeof body.errors === 'object') {
      for (const [key, val] of Object.entries(body.errors)) {
        fields[key] = String(val)
      }
      const first = Object.values(fields)[0]
      if (first) message = first
    } else if (body.error_code) {
      message = body.error_code
    }
  }

  return { status, code: body?.error_code ?? 'UNKNOWN', message, fields }
}

export interface ApiRequestOptions {
  method?: 'GET' | 'POST' | 'PUT' | 'PATCH' | 'DELETE'
  body?: unknown
  query?: Record<string, unknown>
  token?: string | null
  headers?: Record<string, string>
}

/** Panggilan API dasar (tanpa retry). Melempar AppError saat gagal. */
export async function apiFetch<T = unknown>(
  path: string,
  opts: ApiRequestOptions = {},
): Promise<ApiEnvelope<T>> {
  const config = useRuntimeConfig()
  try {
    return await $fetch<ApiEnvelope<T>>(path, {
      baseURL: config.public.apiBase as string,
      method: opts.method ?? 'GET',
      body: opts.body as Record<string, unknown> | undefined,
      query: opts.query,
      headers: {
        ...(opts.token ? { Authorization: `Bearer ${opts.token}` } : {}),
        ...(opts.headers ?? {}),
      },
    })
  } catch (err) {
    throw toAppError(err)
  }
}
