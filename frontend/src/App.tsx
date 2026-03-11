import { BrowserRouter, Routes, Route } from 'react-router-dom'
import { Dashboard } from '@/pages/Dashboard'
import { Management } from '@/pages/Management'

export function App() {
  return (
    <BrowserRouter>
      <Routes>
        <Route path="/" element={<Dashboard />} />
        <Route path="/manage" element={<Management />} />
      </Routes>
    </BrowserRouter>
  )
}
