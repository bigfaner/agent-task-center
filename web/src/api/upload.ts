import { ApiClientError } from './client'
import type { UpsertSummary } from '@/types'

const BASE_URL: string = import.meta.env.VITE_API_URL ?? ''

export async function uploadFile(
  projectName: string,
  featureSlug: string | undefined,
  file: File,
): Promise<UpsertSummary> {
  const searchParams = new URLSearchParams()
  searchParams.set('project', projectName)
  if (featureSlug) searchParams.set('feature', featureSlug)

  const formData = new FormData()
  formData.append('file', file)

  const url = `${BASE_URL}/api/upload?${searchParams.toString()}`
  const res = await fetch(url, {
    method: 'POST',
    body: formData,
  })

  if (!res.ok) {
    const body = await res.json().catch(() => ({
      error: 'unknown',
      message: res.statusText,
    }))
    throw new ApiClientError(res.status, body)
  }

  return res.json() as Promise<UpsertSummary>
}
