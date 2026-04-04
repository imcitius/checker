import { render, screen, waitFor } from '@testing-library/react'
import userEvent from '@testing-library/user-event'
import { describe, it, expect, vi, beforeEach } from 'vitest'
import { Login } from './Login'

describe('Login', () => {
  beforeEach(() => {
    vi.restoreAllMocks()
  })

  it('shows loading state initially', () => {
    vi.stubGlobal(
      'fetch',
      vi.fn().mockReturnValue(new Promise(() => {}))
    )

    render(<Login />)
    expect(screen.getByText('Loading...')).toBeInTheDocument()
  })

  it('renders password form when auth mode is password', async () => {
    vi.stubGlobal(
      'fetch',
      vi.fn().mockResolvedValue({
        ok: true,
        json: () => Promise.resolve({ mode: 'password' }),
      })
    )

    render(<Login />)

    await waitFor(() => {
      expect(screen.getByPlaceholderText('Password')).toBeInTheDocument()
    })
    expect(screen.getByRole('button', { name: 'Sign in' })).toBeInTheDocument()
  })

  it('renders SSO button when auth mode is oidc', async () => {
    vi.stubGlobal(
      'fetch',
      vi.fn().mockResolvedValue({
        ok: true,
        json: () => Promise.resolve({ mode: 'oidc' }),
      })
    )

    render(<Login />)

    await waitFor(() => {
      expect(screen.getByText('Sign in with SSO')).toBeInTheDocument()
    })
  })

  it('shows message when auth is not configured', async () => {
    vi.stubGlobal(
      'fetch',
      vi.fn().mockResolvedValue({
        ok: true,
        json: () => Promise.resolve({ mode: 'none' }),
      })
    )

    render(<Login />)

    await waitFor(() => {
      expect(screen.getByText('Authentication is not configured.')).toBeInTheDocument()
    })
  })

  it('disables sign in button when password is empty', async () => {
    vi.stubGlobal(
      'fetch',
      vi.fn().mockResolvedValue({
        ok: true,
        json: () => Promise.resolve({ mode: 'password' }),
      })
    )

    render(<Login />)

    await waitFor(() => {
      expect(screen.getByRole('button', { name: 'Sign in' })).toBeDisabled()
    })
  })

  it('shows error on failed login', async () => {
    const user = userEvent.setup()

    const fetchMock = vi.fn()
    // First call: auth mode
    fetchMock.mockResolvedValueOnce({
      ok: true,
      json: () => Promise.resolve({ mode: 'password' }),
    })
    // Second call: login attempt
    fetchMock.mockResolvedValueOnce({
      ok: false,
      json: () => Promise.resolve({ error: 'Invalid password' }),
    })
    vi.stubGlobal('fetch', fetchMock)

    render(<Login />)

    await waitFor(() => {
      expect(screen.getByPlaceholderText('Password')).toBeInTheDocument()
    })

    await user.type(screen.getByPlaceholderText('Password'), 'wrong')
    await user.click(screen.getByRole('button', { name: 'Sign in' }))

    await waitFor(() => {
      expect(screen.getByText('Invalid password')).toBeInTheDocument()
    })
  })

  it('shows Checker title and sign-in text', () => {
    vi.stubGlobal(
      'fetch',
      vi.fn().mockReturnValue(new Promise(() => {}))
    )

    render(<Login />)
    expect(screen.getByText('Checker')).toBeInTheDocument()
    expect(screen.getByText('Sign in to access your dashboard')).toBeInTheDocument()
  })
})
