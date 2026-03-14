import { BrowserRouter, Routes, Route } from 'react-router-dom'
import { Toaster } from 'sonner'
import { Dashboard } from '@/pages/Dashboard'
import { Management } from '@/pages/Management'
import { Alerts } from '@/pages/Alerts'
import { Login } from '@/pages/Login'
import { VersionBadge } from '@/components/VersionBadge'
import { useTheme } from '@/lib/theme'

export function App() {
  const { resolved } = useTheme()

  return (
    <BrowserRouter>
      <Routes>
        <Route path="/" element={<Dashboard />} />
        <Route path="/manage" element={<Management />} />
        <Route path="/alerts" element={<Alerts />} />
        <Route path="/login" element={<Login />} />
      </Routes>
      <VersionBadge />
      <Toaster theme={resolved} richColors position="bottom-right" />
    </BrowserRouter>
  )
}
