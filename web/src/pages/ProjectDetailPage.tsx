import { useState } from 'react'
import { useParams, Link } from 'react-router-dom'
import { useQuery, useQueryClient } from '@tanstack/react-query'
import { AppHeader } from '@/components/AppHeader'
import { StatusBadge } from '@/components/StatusBadge'
import { CompletionRateBar } from '@/components/CompletionRateBar'
import { RelativeTime } from '@/components/RelativeTime'
import { TableSkeleton } from '@/components/TableSkeleton'
import { EmptyState } from '@/components/EmptyState'
import { Tabs, TabsList, TabsTrigger, TabsContent } from '@/components/ui/tabs'
import { getProject } from '@/api/projects'

export default function ProjectDetailPage() {
  const { id } = useParams<{ id: string }>()
  const queryClient = useQueryClient()
  const [tab, setTab] = useState('features')

  const { data, isLoading } = useQuery({
    queryKey: ['project', id],
    queryFn: () => getProject(Number(id)),
  })

  const handleUploadSuccess = () => {
    queryClient.invalidateQueries({ queryKey: ['project', id] })
  }

  return (
    <div className="flex min-h-screen flex-col">
      <AppHeader
        projectName={data?.name}
        onUploadSuccess={handleUploadSuccess}
      />
      <main className="flex-1 p-6">
        <Link
          to="/"
          className="text-sm text-muted-foreground hover:underline"
        >
          &larr; Back to projects
        </Link>

        {isLoading ? (
          <div className="mt-4">
            <div className="h-8 w-48 animate-pulse rounded bg-muted" />
            <div className="mt-4">
              <TableSkeleton rows={5} />
            </div>
          </div>
        ) : data ? (
          <>
            <h1 className="mt-4 text-2xl font-bold">{data.name}</h1>

            <Tabs
              value={tab}
              onValueChange={setTab}
              className="mt-4"
            >
              <TabsList>
                <TabsTrigger value="features">Features</TabsTrigger>
                <TabsTrigger value="proposals">Proposals</TabsTrigger>
              </TabsList>

              <TabsContent value="features" className="mt-4">
                {data.features.length === 0 ? (
                  <EmptyState title="此项目暂无 Features" />
                ) : (
                  <table className="w-full text-sm">
                    <thead>
                      <tr className="border-b text-left text-muted-foreground">
                        <th className="pb-2 pr-4 font-medium">Feature</th>
                        <th className="pb-2 pr-4 font-medium">Status</th>
                        <th className="pb-2 pr-4 font-medium">
                          Completion
                        </th>
                        <th className="pb-2 font-medium">Updated</th>
                      </tr>
                    </thead>
                    <tbody>
                      {data.features.map((feature) => (
                        <tr key={feature.id} className="border-b last:border-0">
                          <td className="py-3 pr-4">
                            <Link
                              to={`/features/${feature.id}/tasks`}
                              className="font-medium text-primary hover:underline"
                            >
                              {feature.name}
                            </Link>
                          </td>
                          <td className="py-3 pr-4">
                            <StatusBadge status={feature.status} />
                          </td>
                          <td className="py-3 pr-4">
                            <CompletionRateBar rate={feature.completionRate} />
                          </td>
                          <td className="py-3">
                            <RelativeTime date={feature.updatedAt} />
                          </td>
                        </tr>
                      ))}
                    </tbody>
                  </table>
                )}
              </TabsContent>

              <TabsContent value="proposals" className="mt-4">
                {data.proposals.length === 0 ? (
                  <EmptyState title="此项目暂无 Proposals" />
                ) : (
                  <table className="w-full text-sm">
                    <thead>
                      <tr className="border-b text-left text-muted-foreground">
                        <th className="pb-2 pr-4 font-medium">Title</th>
                        <th className="pb-2 pr-4 font-medium">Features</th>
                        <th className="pb-2 font-medium">Created</th>
                      </tr>
                    </thead>
                    <tbody>
                      {data.proposals.map((proposal) => (
                        <tr key={proposal.id} className="border-b last:border-0">
                          <td className="py-3 pr-4">
                            <Link
                              to={`/proposals/${proposal.id}`}
                              className="font-medium text-primary hover:underline"
                            >
                              {proposal.title}
                            </Link>
                          </td>
                          <td className="py-3 pr-4">
                            {proposal.featureCount}
                          </td>
                          <td className="py-3">
                            <RelativeTime date={proposal.createdAt} />
                          </td>
                        </tr>
                      ))}
                    </tbody>
                  </table>
                )}
              </TabsContent>
            </Tabs>
          </>
        ) : null}
      </main>
    </div>
  )
}
