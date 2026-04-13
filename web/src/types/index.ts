// ── Pagination ──

export interface PaginatedResponse<T> {
  items: T[]
  total: number
  page: number
  pageSize: number
}

// ── Project ──

export interface ProjectSummary {
  id: number
  name: string
  featureCount: number
  taskTotal: number
  completionRate: number
  updatedAt: string
}

export type ListProjectsResponse = PaginatedResponse<ProjectSummary>

export interface ProjectProposal {
  id: number
  slug: string
  title: string
  createdAt: string
  featureCount: number
}

export interface ProjectFeature {
  id: number
  slug: string
  name: string
  status: FeatureStatus
  completionRate: number
  updatedAt: string
}

export interface ProjectDetail {
  id: number
  name: string
  proposals: ProjectProposal[]
  features: ProjectFeature[]
}

// ── Feature ──

export type FeatureStatus =
  | 'prd'
  | 'design'
  | 'tasks'
  | 'in-progress'
  | 'done'

// ── Task ──

export type TaskStatus =
  | 'pending'
  | 'in_progress'
  | 'completed'
  | 'blocked'

export type TaskPriority = 'P0' | 'P1' | 'P2'

export interface TaskSummary {
  id: number
  taskId: string
  title: string
  status: TaskStatus
  priority: TaskPriority
  tags: string[]
  claimedBy: string | null
  dependencies: string[]
}

export interface FeatureTasksResponse {
  featureId: number
  featureName: string
  tasks: TaskSummary[]
}

export interface TaskFilter {
  priority?: string
  tag?: string
  status?: string
}

export interface TaskDetail {
  id: number
  taskId: string
  title: string
  description: string
  status: TaskStatus
  priority: TaskPriority
  tags: string[]
  claimedBy: string | null
  dependencies: string[]
  createdAt: string
  updatedAt: string
}

// ── Execution Record ──

export interface AcceptanceCriterion {
  criterion: string
  met: boolean
}

export interface ExecutionRecord {
  id: number
  agentId: string
  summary: string
  filesCreated: string[]
  filesModified: string[]
  keyDecisions: string[]
  testsPassed: number
  testsFailed: number
  coverage: number
  acceptanceCriteria: AcceptanceCriterion[]
  createdAt: string
}

export type ListRecordsResponse = PaginatedResponse<ExecutionRecord>

// ── Document ──

export interface RelatedFeature {
  id: number
  name: string
  slug: string
}

export interface RelatedTask {
  id: number
  taskId: string
  title: string
}

export interface DocumentContent {
  title: string
  content: string
  relatedFeatures: RelatedFeature[]
  relatedTasks: RelatedTask[]
}

// ── Upload ──

export interface UpsertSummary {
  filename: string
  created: number
  updated: number
  skipped: number
  message: string
}

// ── API Error ──

export interface ApiError {
  error: string
  message: string
  hint?: string
}
