import { BrowserRouter, Routes, Route } from 'react-router-dom'
import { Dashboard } from '@/pages/Dashboard'
import { Management } from '@/pages/Management'
import { Alerts } from '@/pages/Alerts'
import { Login } from '@/pages/Login'
import { VersionBadge } from '@/components/VersionBadge'

export function App() {
  return (
    <BrowserRouter>
      <Routes>
        <Route path="/" element={<Dashboard />} />
        <Route path="/manage" element={<Management />} />
        <Route path="/alerts" element={<Alerts />} />
        <Route path="/login" element={<Login />} />
      </Routes>
      <VersionBadge />
    </BrowserRouter>
  )
}
