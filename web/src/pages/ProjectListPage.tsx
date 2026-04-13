import { useState } from 'react'
import { useQuery, useQueryClient } from '@tanstack/react-query'
import { AppHeader } from '@/components/AppHeader'
import { SearchInput } from '@/components/SearchInput'
import { ProjectTable } from '@/components/ProjectTable'
import { TableSkeleton } from '@/components/TableSkeleton'
import { EmptyState } from '@/components/EmptyState'
import { ErrorState } from '@/components/ErrorState'
import { useDebounce } from '@/hooks/useDebounce'
import { listProjects } from '@/api/projects'

export default function ProjectListPage() {
  const [search, setSearch] = useState('')
  const debouncedSearch = useDebounce(search, 300)
  const queryClient = useQueryClient()

  const { data, isLoading, error, refetch } = useQuery({
    queryKey: ['projects', debouncedSearch],
    queryFn: () =>
      listProjects({ search: debouncedSearch, page: 1, pageSize: 20 }),
  })

  const handleUploadSuccess = () => {
    queryClient.invalidateQueries({ queryKey: ['projects'] })
  }

  return (
    <div className="flex min-h-screen flex-col">
      <AppHeader onUploadSuccess={handleUploadSuccess} />
      <main className="flex-1 p-6">
        <h1 className="text-2xl font-bold">Projects</h1>
        <div className="mt-4">
          <SearchInput value={search} onChange={setSearch} />
        </div>
        <div className="mt-4">
          {isLoading ? (
            <TableSkeleton rows={5} />
          ) : error ? (
            <ErrorState
              message="Failed to load projects"
              onRetry={() => refetch()}
            />
          ) : data && data.items.length > 0 ? (
            <ProjectTable projects={data.items} />
          ) : data && data.items.length === 0 ? (
            <EmptyState
              title={
                debouncedSearch
                  ? '未找到匹配项目'
                  : '暂无项目，点击上传开始'
              }
            />
          ) : null}
        </div>
      </main>
    </div>
  )
}
