import { request } from './client'
import type { ListProjectsResponse, ProjectDetail } from '@/types'

export function listProjects(params?: {
  search?: string
  page?: number
  pageSize?: number
}): Promise<ListProjectsResponse> {
  const searchParams = new URLSearchParams()
  if (params?.search) searchParams.set('search', params.search)
  if (params?.page) searchParams.set('page', String(params.page))
  if (params?.pageSize) searchParams.set('pageSize', String(params.pageSize))
  const qs = searchParams.toString()
  return request(`/api/projects${qs ? `?${qs}` : ''}`)
}

export function getProject(id: number): Promise<ProjectDetail> {
  return request(`/api/projects/${id}`)
}
