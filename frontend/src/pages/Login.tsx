import { useEffect, useState } from 'react'

type AuthMode = 'password' | 'oidc' | 'none' | null

export function Login() {
  const [mode, setMode] = useState<AuthMode>(null)
  const [password, setPassword] = useState('')
  const [error, setError] = useState('')
  const [loading, setLoading] = useState(false)

  useEffect(() => {
    fetch('/api/auth/mode')
      .then(r => r.json())
      .then(d => setMode(d.mode))
      .catch(() => setMode('none'))
  }, [])

  const handlePasswordSubmit = async (e: React.FormEvent) => {
    e.preventDefault()
    setError('')
    setLoading(true)

    try {
      const res = await fetch('/auth/login', {
        method: 'POST',
        headers: { 'Content-Type': 'application/json' },
        body: JSON.stringify({ password }),
      })

      if (res.ok) {
        window.location.href = '/'
      } else {
        const data = await res.json()
        setError(data.error || 'Login failed')
      }
    } catch {
      setError('Connection error')
    } finally {
      setLoading(false)
    }
  }

  return (
    <div className="min-h-screen bg-background flex items-center justify-center">
      <div className="w-full max-w-sm mx-auto px-4">
        {/* Logo and title */}
        <div className="text-center mb-8">
          <div className="inline-flex h-14 w-14 rounded-lg bg-healthy/20 items-center justify-center mb-4">
            <span className="text-healthy font-mono font-bold text-2xl">C</span>
          </div>
          <h1 className="text-2xl font-semibold text-foreground">Checker</h1>
          <p className="text-muted-foreground text-sm mt-1">
            Sign in to access your dashboard
          </p>
        </div>

        {/* Login card */}
        <div className="rounded-lg border bg-card p-6 space-y-4">
          {mode === null && (
            <div className="text-center text-muted-foreground text-sm">Loading...</div>
          )}

          {mode === 'password' && (
            <form onSubmit={handlePasswordSubmit} className="space-y-4">
              <div>
                <input
                  type="password"
                  value={password}
                  onChange={e => setPassword(e.target.value)}
                  placeholder="Password"
                  autoFocus
                  className="w-full h-10 px-3 rounded-md border bg-background text-foreground text-sm placeholder:text-muted-foreground focus:outline-none focus:ring-2 focus:ring-ring"
                />
              </div>
              {error && (
                <p className="text-sm text-destructive">{error}</p>
              )}
              <button
                type="submit"
                disabled={loading || !password}
                className="inline-flex items-center justify-center w-full h-10 px-4 rounded-md text-sm font-medium transition-colors bg-primary text-primary-foreground shadow hover:bg-primary/90 disabled:opacity-50"
              >
                {loading ? 'Signing in...' : 'Sign in'}
              </button>
            </form>
          )}

          {mode === 'oidc' && (
            <a
              href="/auth/login"
              className="inline-flex items-center justify-center w-full h-10 px-4 rounded-md text-sm font-medium transition-colors bg-primary text-primary-foreground shadow hover:bg-primary/90 gap-2"
            >
              Sign in with SSO
            </a>
          )}

          {mode === 'none' && (
            <p className="text-center text-sm text-muted-foreground">
              Authentication is not configured.
            </p>
          )}
        </div>

        <p className="text-center text-xs text-muted-foreground mt-6">
          {mode === 'oidc'
            ? 'Authentication powered by your organization\u2019s identity provider.'
            : mode === 'password'
              ? 'Enter the password configured for this instance.'
              : '\u00A0'}
        </p>
      </div>
    </div>
  )
}
