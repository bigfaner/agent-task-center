import { request } from './client'
import type { DocumentContent } from '@/types'

export function getProposalContent(id: number): Promise<DocumentContent> {
  return request(`/api/proposals/${id}/content`)
}

export function getFeatureContent(id: number): Promise<DocumentContent> {
  return request(`/api/features/${id}/content`)
}
