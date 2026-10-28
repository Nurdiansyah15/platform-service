import type { ApiEnvelope, AppError } from '~/types/api'
import type { ApiRequestOptions } from '~/utils/http'

/**
 * Klien API terautentikasi: menyisipkan access token dan otomatis refresh
 * sekali saat 401 sebelum mengulang request.
 */
export function useApi() {
  const auth = useAuthStore()

  async function request<T = unknown>(
    path: string,
    opts: ApiRequestOptions = {},
  ): Promise<ApiEnvelope<T>> {
    try {
      return await apiFetch<T>(path, { ...opts, token: auth.accessToken })
    } catch (err) {
      const appErr = err as AppError
      if (appErr.status === 401 && auth.refreshToken) {
        const refreshed = await auth.tryRefresh()
        if (refreshed) {
          return await apiFetch<T>(path, { ...opts, token: auth.accessToken })
        }
      }
      throw appErr
    }
  }

  return {
    request,
    get: <T = unknown>(path: string, opts: ApiRequestOptions = {}) =>
      request<T>(path, { ...opts, method: 'GET' }),
    post: <T = unknown>(path: string, body?: unknown, opts: ApiRequestOptions = {}) =>
      request<T>(path, { ...opts, method: 'POST', body }),
    put: <T = unknown>(path: string, body?: unknown, opts: ApiRequestOptions = {}) =>
      request<T>(path, { ...opts, method: 'PUT', body }),
    patch: <T = unknown>(path: string, body?: unknown, opts: ApiRequestOptions = {}) =>
      request<T>(path, { ...opts, method: 'PATCH', body }),
    del: <T = unknown>(path: string, opts: ApiRequestOptions = {}) =>
      request<T>(path, { ...opts, method: 'DELETE' }),
  }
}
