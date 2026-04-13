import { AppHeader } from '@/components/AppHeader'

export default function ProjectListPage() {
  return (
    <div className="flex min-h-screen flex-col">
      <AppHeader />
      <main className="flex-1 p-6">
        <h1 className="text-2xl font-bold">Projects</h1>
        <p className="mt-2 text-muted-foreground">
          Project list will appear here.
        </p>
      </main>
    </div>
  )
}
