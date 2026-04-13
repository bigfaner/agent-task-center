import { request } from './client'
import type { TaskDetail, ListRecordsResponse } from '@/types'

export function getTask(id: number): Promise<TaskDetail> {
  return request(`/api/tasks/${id}`)
}

export function listTaskRecords(
  id: number,
  params?: { page?: number; pageSize?: number },
): Promise<ListRecordsResponse> {
  const searchParams = new URLSearchParams()
  if (params?.page) searchParams.set('page', String(params.page))
  if (params?.pageSize) searchParams.set('pageSize', String(params.pageSize))
  const qs = searchParams.toString()
  return request(`/api/tasks/${id}/records${qs ? `?${qs}` : ''}`)
}
