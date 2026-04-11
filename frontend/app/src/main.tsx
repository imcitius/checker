import { StrictMode } from 'react'
import { createRoot } from 'react-dom/client'
import * as Sentry from '@sentry/react'
import { ThemeProvider } from '@ensafely/checker-ui'
import { App } from './App'

// Import styles from the shared UI library
import '@ensafely/checker-ui/styles.css'

const sentryDsn = import.meta.env.VITE_SENTRY_DSN
if (sentryDsn) {
  Sentry.init({
    dsn: sentryDsn,
    environment: import.meta.env.VITE_SENTRY_ENVIRONMENT || 'production',
    integrations: [
      Sentry.browserTracingIntegration(),
      Sentry.replayIntegration(),
    ],
    tracesSampleRate: parseFloat(import.meta.env.VITE_SENTRY_TRACES_SAMPLE_RATE || '0.1'),
    tracePropagationTargets: [/^\/api\//],
    replaysSessionSampleRate: 0,
    replaysOnErrorSampleRate: 1.0,
  })
}

createRoot(document.getElementById('root')!).render(
  <StrictMode>
    <Sentry.ErrorBoundary fallback={<div className="p-8 text-center text-red-500">Something went wrong. Please refresh the page.</div>}>
      <ThemeProvider>
        <App />
      </ThemeProvider>
    </Sentry.ErrorBoundary>
  </StrictMode>
)
