import { useState } from 'react'
import { uploadFile } from '@/api/upload'

interface UploadDialogProps {
  open: boolean
  onOpenChange: (open: boolean) => void
  projectName?: string
  onUploadSuccess?: () => void
}

export function UploadDialog({
  open,
  onOpenChange,
  projectName,
  onUploadSuccess,
}: UploadDialogProps) {
  const [file, setFile] = useState<File | null>(null)
  const [uploading, setUploading] = useState(false)
  const [error, setError] = useState<string | null>(null)
  const [success, setSuccess] = useState<string | null>(null)

  if (!open) return null

  const handleUpload = async () => {
    if (!file || !projectName) return
    setUploading(true)
    setError(null)
    try {
      const result = await uploadFile(projectName, undefined, file)
      setSuccess(result.message)
      onUploadSuccess?.()
    } catch (err) {
      setError(err instanceof Error ? err.message : 'Upload failed')
    } finally {
      setUploading(false)
    }
  }

  const handleClose = () => {
    setFile(null)
    setError(null)
    setSuccess(null)
    onOpenChange(false)
  }

  return (
    <div className="fixed inset-0 z-50 flex items-center justify-center bg-black/50">
      <div className="w-full max-w-md rounded-lg bg-background p-6 shadow-lg">
        <h2 className="text-lg font-semibold">Upload File</h2>

        <div className="mt-4">
          <input
            type="file"
            accept=".json,.md"
            onChange={(e) => {
              const f = e.target.files?.[0]
              if (f) {
                setFile(f)
                setError(null)
                setSuccess(null)
              }
            }}
          />
        </div>

        {file && (
          <p className="mt-2 text-sm text-muted-foreground">
            {file.name} ({(file.size / 1024).toFixed(1)} KB)
          </p>
        )}

        {error && <p className="mt-2 text-sm text-destructive">{error}</p>}
        {success && <p className="mt-2 text-sm text-green-600">{success}</p>}

        <div className="mt-6 flex justify-end gap-2">
          <button
            type="button"
            onClick={handleClose}
            className="rounded-lg border px-4 py-2 text-sm"
          >
            Cancel
          </button>
          <button
            type="button"
            onClick={handleUpload}
            disabled={!file || uploading || !projectName}
            className="rounded-lg bg-primary px-4 py-2 text-sm font-medium text-primary-foreground disabled:opacity-50"
          >
            {uploading ? 'Uploading...' : 'Upload'}
          </button>
        </div>
      </div>
    </div>
  )
}
