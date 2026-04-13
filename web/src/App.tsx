import { Routes, Route } from 'react-router-dom'
import ProjectListPage from '@/pages/ProjectListPage'
import ProjectDetailPage from '@/pages/ProjectDetailPage'
import FeatureKanbanPage from '@/pages/FeatureKanbanPage'
import TaskDetailPage from '@/pages/TaskDetailPage'
import DocViewerPage from '@/pages/DocViewerPage'

function App() {
  return (
    <Routes>
      <Route path="/" element={<ProjectListPage />} />
      <Route path="/projects/:id" element={<ProjectDetailPage />} />
      <Route path="/features/:id/tasks" element={<FeatureKanbanPage />} />
      <Route path="/tasks/:id" element={<TaskDetailPage />} />
      <Route path="/proposals/:id" element={<DocViewerPage />} />
    </Routes>
  )
}

export default App
