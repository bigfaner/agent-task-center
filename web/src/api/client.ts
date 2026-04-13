import type { ApiError } from '@/types'

const BASE_URL: string = import.meta.env.VITE_API_URL ?? ''

export class ApiClientError extends Error {
  constructor(
    public status: number,
    public body: ApiError,
  ) {
    super(body.message || `API error ${status}`)
    this.name = 'ApiClientError'
  }
}

export async function request<T>(
  path: string,
  options?: RequestInit,
): Promise<T> {
  const url = `${BASE_URL}${path}`
  const res = await fetch(url, {
    ...options,
    headers: {
      'Content-Type': 'application/json',
      ...options?.headers,
    },
  })

  if (!res.ok) {
    const body: ApiError = await res.json().catch(() => ({
      error: 'unknown',
      message: res.statusText,
    }))
    throw new ApiClientError(res.status, body)
  }

  return res.json() as Promise<T>
}
