import type { Config } from 'tailwindcss'
import uiConfig from '../ui/tailwind.config'

/**
 * Standalone UI extends the shared UI tailwind config.
 * The content path includes the ui library source so that all
 * utility classes used by shared components are included.
 */
const config: Config = {
  ...uiConfig,
  content: [
    './index.html',
    './src/**/*.{ts,tsx}',
    '../ui/src/**/*.{ts,tsx}',
  ],
}

export default config
