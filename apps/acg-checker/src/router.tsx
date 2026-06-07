import { createBrowserRouter, Navigate } from 'react-router-dom'
import { CheckPage } from './pages/CheckPage'
import { FailedPage } from './pages/FailedPage'

export const router = createBrowserRouter([
  { path: '/check', element: <CheckPage /> },
  { path: '/failed', element: <FailedPage /> },
  { path: '*', element: <Navigate to="/check" replace /> },
])
