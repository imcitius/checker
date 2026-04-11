/**
 * Standalone Checker UI — wraps @ensafely/checker-ui components with
 * standalone defaults: cookie auth, all channel types, no extra nav items.
 *
 * This produces IDENTICAL output to the original checker-core/frontend/src/App.tsx.
 */
import { BrowserRouter, Routes, Route } from 'react-router-dom'
import { Toaster } from 'sonner'
import {
  Dashboard,
  Management,
  Alerts,
  Settings,
  Login,
  VersionBadge,
  CommandPalette,
  useTheme,
} from '@ensafely/checker-ui'

function AppContent() {
  const { resolved } = useTheme()

  return (
    <>
      <Routes>
        <Route path="/" element={<Dashboard />} />
        <Route path="/manage" element={<Management />} />
        <Route path="/alerts" element={<Alerts />} />
        <Route path="/settings" element={<Settings />} />
        <Route path="/login" element={<Login />} />
      </Routes>
      <CommandPalette />
      <VersionBadge />
      <Toaster theme={resolved} richColors position="bottom-right" />
    </>
  )
}

export function App() {
  return (
    <BrowserRouter>
      <AppContent />
    </BrowserRouter>
  )
}
