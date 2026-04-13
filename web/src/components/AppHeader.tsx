import { useState } from 'react'
import { UploadDialog } from './UploadDialog'

interface AppHeaderProps {
  projectName?: string
  showUpload?: boolean
  onUploadSuccess?: () => void
}

export function AppHeader({
  projectName,
  showUpload = true,
  onUploadSuccess,
}: AppHeaderProps) {
  const [uploadOpen, setUploadOpen] = useState(false)

  return (
    <>
      <header className="flex items-center justify-between border-b px-6 py-3">
        <div className="text-lg font-semibold">Agent Task Center</div>
        {showUpload && (
          <button
            type="button"
            onClick={() => setUploadOpen(true)}
            className="rounded-lg bg-primary px-4 py-2 text-sm font-medium text-primary-foreground hover:bg-primary/80"
          >
            Upload
          </button>
        )}
      </header>
      <UploadDialog
        open={uploadOpen}
        onOpenChange={setUploadOpen}
        projectName={projectName}
        onUploadSuccess={onUploadSuccess}
      />
    </>
  )
}
