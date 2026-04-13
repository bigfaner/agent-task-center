import { useState, useRef, useEffect, type DragEvent, type ChangeEvent } from 'react'
import { useQuery } from '@tanstack/react-query'
import { uploadFile } from '@/api/upload'
import { listProjects } from '@/api/projects'

// ── Types ──

type UploadState = 'idle' | 'validating' | 'ready' | 'invalid' | 'uploading' | 'success' | 'error'

interface UploadDialogProps {
  open: boolean
  onOpenChange: (open: boolean) => void
  projectName?: string
  onUploadSuccess?: () => void
}

const MAX_FILE_SIZE = 5 * 1024 * 1024 // 5MB
const VALID_EXTENSIONS = ['.json', '.md']

function validateFile(file: File): string | null {
  const ext = file.name.substring(file.name.lastIndexOf('.')).toLowerCase()
  if (!VALID_EXTENSIONS.includes(ext)) {
    return 'Only .json and .md files are supported'
  }
  if (file.size > MAX_FILE_SIZE) {
    return 'File size cannot exceed 5MB'
  }
  return null
}

// ── Component ──

export function UploadDialog({
  open,
  onOpenChange,
  projectName,
  onUploadSuccess,
}: UploadDialogProps) {
  const [file, setFile] = useState<File | null>(null)
  const [selectedProject, setSelectedProject] = useState(projectName ?? '')
  const [state, setState] = useState<UploadState>('idle')
  const [validationError, setValidationError] = useState<string | null>(null)
  const [uploadError, setUploadError] = useState<string | null>(null)
  const [successMessage, setSuccessMessage] = useState<string | null>(null)
  const [isDragOver, setIsDragOver] = useState(false)
  const fileInputRef = useRef<HTMLInputElement>(null)

  // Fetch projects for selector
  const { data: projectsData } = useQuery({
    queryKey: ['projects-list'],
    queryFn: () => listProjects({ pageSize: 100 }),
    enabled: open && !projectName,
  })

  // Reset state when dialog opens/closes
  useEffect(() => {
    if (open) {
      setFile(null)
      setSelectedProject(projectName ?? '')
      setState('idle')
      setValidationError(null)
      setUploadError(null)
      setSuccessMessage(null)
      setIsDragOver(false)
    }
  }, [open, projectName])

  const handleFileSelect = (selectedFile: File) => {
    const error = validateFile(selectedFile)
    if (error) {
      setFile(selectedFile)
      setValidationError(error)
      setUploadError(null)
      setState('invalid')
    } else {
      setFile(selectedFile)
      setValidationError(null)
      setUploadError(null)
      setState('ready')
    }
  }

  const handleInputChange = (e: ChangeEvent<HTMLInputElement>) => {
    const selected = e.target.files?.[0]
    if (selected) {
      handleFileSelect(selected)
    }
  }

  const handleDragOver = (e: DragEvent) => {
    e.preventDefault()
    setIsDragOver(true)
  }

  const handleDragLeave = (e: DragEvent) => {
    e.preventDefault()
    setIsDragOver(false)
  }

  const handleDrop = (e: DragEvent) => {
    e.preventDefault()
    setIsDragOver(false)
    const droppedFile = e.dataTransfer.files[0]
    if (droppedFile) {
      handleFileSelect(droppedFile)
    }
  }

  const handleDropZoneClick = () => {
    fileInputRef.current?.click()
  }

  const handleUpload = async () => {
    if (!file) return
    const projectToUse = selectedProject || projectName
    if (!projectToUse) return

    setState('uploading')
    setUploadError(null)

    try {
      const result = await uploadFile(projectToUse, undefined, file)
      setSuccessMessage(result.message)
      setState('success')
      onUploadSuccess?.()
    } catch (err) {
      setUploadError(err instanceof Error ? err.message : 'Upload failed')
      setState('error')
    }
  }

  const handleClose = () => {
    onOpenChange(false)
  }

  if (!open) return null

  const canUpload = state === 'ready' && file && (selectedProject || projectName)
  const errorMessage = validationError || uploadError

  return (
    <div className="fixed inset-0 z-50 flex items-center justify-center bg-black/50">
      <div
        data-testid="upload-dialog"
        className="w-full max-w-md rounded-lg bg-background p-6 shadow-lg"
      >
        {/* Header */}
        <div className="flex items-center justify-between">
          <h2 className="text-lg font-semibold">Upload File</h2>
          <button
            type="button"
            onClick={handleClose}
            className="text-muted-foreground hover:text-foreground"
            aria-label="Close"
          >
            &#x2715;
          </button>
        </div>

        {/* Project Selector */}
        {!projectName && (
          <div className="mt-4">
            <label className="mb-1 block text-sm font-medium">Project</label>
            <select
              data-testid="project-selector"
              value={selectedProject}
              onChange={(e) => setSelectedProject(e.target.value)}
              className="w-full rounded-md border bg-background px-3 py-2 text-sm"
            >
              <option value="">Select project</option>
              {projectsData?.items.map((p) => (
                <option key={p.id} value={p.name}>
                  {p.name}
                </option>
              ))}
            </select>
          </div>
        )}

        {/* Drop Zone */}
        <div className="mt-4">
          <div
            data-testid="drop-zone"
            role="button"
            tabIndex={0}
            onClick={handleDropZoneClick}
            onKeyDown={(e) => {
              if (e.key === 'Enter' || e.key === ' ') handleDropZoneClick()
            }}
            onDragOver={handleDragOver}
            onDragLeave={handleDragLeave}
            onDrop={handleDrop}
            className={`flex cursor-pointer flex-col items-center justify-center rounded-lg border-2 border-dashed p-8 transition-colors ${
              isDragOver
                ? 'border-primary bg-primary/5'
                : 'border-muted-foreground/25 hover:border-muted-foreground/50'
            }`}
          >
            <p className="text-sm text-muted-foreground">
              Drag &amp; drop files here, or click to select
            </p>
            <p className="mt-1 text-xs text-muted-foreground">
              Supports .json and .md, max 5MB
            </p>
          </div>

          <input
            ref={fileInputRef}
            type="file"
            accept=".json,.md"
            onChange={handleInputChange}
            className="hidden"
          />
        </div>

        {/* Selected file preview */}
        {file && state !== 'success' && (
          <div data-testid="selected-file" className="mt-3 text-sm">
            <span className="font-medium">{file.name}</span>
            <span className="ml-2 text-muted-foreground">
              ({(file.size / 1024).toFixed(1)} KB)
            </span>
          </div>
        )}

        {/* Validation / Error message */}
        {errorMessage && (
          <p data-testid="validation-message" className="mt-2 text-sm text-destructive">
            {errorMessage}
          </p>
        )}

        {/* Upload progress */}
        {state === 'uploading' && (
          <div className="mt-3">
            <div className="h-2 w-full overflow-hidden rounded-full bg-muted">
              <div className="h-full animate-pulse rounded-full bg-primary" style={{ width: '100%' }} />
            </div>
            <p className="mt-1 text-xs text-muted-foreground">Uploading...</p>
          </div>
        )}

        {/* Upload result */}
        {state === 'success' && successMessage && (
          <div data-testid="upload-result" className="mt-3 rounded-md bg-green-50 p-3 text-sm text-green-700">
            <span className="font-medium">Success:</span> {successMessage}
          </div>
        )}

        {/* Footer */}
        <div className="mt-6 flex justify-end gap-2">
          <button
            type="button"
            onClick={handleClose}
            className="rounded-lg border px-4 py-2 text-sm hover:bg-accent"
          >
            {state === 'success' ? 'Close' : 'Cancel'}
          </button>
          {state !== 'success' && (
            <button
              type="button"
              onClick={state === 'error' ? handleUpload : handleUpload}
              disabled={!canUpload}
              className="rounded-lg bg-primary px-4 py-2 text-sm font-medium text-primary-foreground hover:bg-primary/80 disabled:opacity-50"
            >
              {state === 'uploading' ? 'Uploading...' : state === 'error' ? 'Retry' : 'Upload'}
            </button>
          )}
        </div>
      </div>
    </div>
  )
}
