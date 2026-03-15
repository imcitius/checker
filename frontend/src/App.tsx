import { BrowserRouter, Routes, Route } from 'react-router-dom'
import { Toaster } from 'sonner'
import { Dashboard } from '@/pages/Dashboard'
import { Management } from '@/pages/Management'
import { Alerts } from '@/pages/Alerts'
import { Login } from '@/pages/Login'
import { Settings } from '@/pages/Settings'
import { VersionBadge } from '@/components/VersionBadge'
import { CommandPalette } from '@/components/CommandPalette'
import { useTheme } from '@/lib/theme'

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
