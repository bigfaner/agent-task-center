import { useParams, Link } from 'react-router-dom'

export default function DocViewerPage() {
  const { id } = useParams<{ id: string }>()

  return (
    <div className="flex min-h-screen flex-col">
      <main className="flex-1 p-6">
        <Link to=".." className="text-sm text-muted-foreground hover:underline">
          &larr; Back
        </Link>
        <h1 className="mt-4 text-2xl font-bold">Proposal {id}</h1>
        <p className="mt-2 text-muted-foreground">
          Document content will appear here.
        </p>
      </main>
    </div>
  )
}
