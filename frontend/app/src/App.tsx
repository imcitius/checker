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
  TestCooldownProvider,
  TopBarConfigProvider,
  useTheme,
} from '@ensafely/checker-ui'

/**
 * Standalone logout: cookie-based auth — navigate to server-side logout
 * endpoint which clears the session cookie and redirects to login.
 */
function handleLogout() {
  window.location.href = '/auth/logout'
}

function AppContent() {
  const { resolved } = useTheme()

  return (
    <TopBarConfigProvider brandName="Checker" onLogout={handleLogout}>
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
    </TopBarConfigProvider>
  )
}

export function App() {
  return (
    <BrowserRouter>
      <TestCooldownProvider>
        <AppContent />
      </TestCooldownProvider>
    </BrowserRouter>
  )
}
