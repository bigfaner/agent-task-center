import { request } from './client'
import type { FeatureTasksResponse, TaskFilter } from '@/types'

export function getFeatureTasks(
  id: number,
  filter?: TaskFilter,
): Promise<FeatureTasksResponse> {
  const searchParams = new URLSearchParams()
  if (filter?.priority) searchParams.set('priority', filter.priority)
  if (filter?.tag) searchParams.set('tag', filter.tag)
  if (filter?.status) searchParams.set('status', filter.status)
  const qs = searchParams.toString()
  return request(`/api/features/${id}/tasks${qs ? `?${qs}` : ''}`)
}
