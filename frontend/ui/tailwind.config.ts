import type { Config } from 'tailwindcss'
import tailwindcssAnimate from 'tailwindcss-animate'

const config: Config = {
  darkMode: 'class',
  content: ['./src/**/*.{ts,tsx}'],
  theme: {
    extend: {
      colors: {
        border: 'hsl(var(--border))',
        input: 'hsl(var(--input))',
        ring: 'hsl(var(--ring))',
        background: 'hsl(var(--background))',
        foreground: 'hsl(var(--foreground))',
        primary: {
          DEFAULT: 'hsl(var(--primary))',
          foreground: 'hsl(var(--primary-foreground))',
        },
        secondary: {
          DEFAULT: 'hsl(var(--secondary))',
          foreground: 'hsl(var(--secondary-foreground))',
        },
        destructive: {
          DEFAULT: 'hsl(var(--destructive))',
          foreground: 'hsl(var(--destructive-foreground))',
        },
        muted: {
          DEFAULT: 'hsl(var(--muted))',
          foreground: 'hsl(var(--muted-foreground))',
        },
        accent: {
          DEFAULT: 'hsl(var(--accent))',
          foreground: 'hsl(var(--accent-foreground))',
        },
        popover: {
          DEFAULT: 'hsl(var(--popover))',
          foreground: 'hsl(var(--popover-foreground))',
        },
        card: {
          DEFAULT: 'hsl(var(--card))',
          foreground: 'hsl(var(--card-foreground))',
        },
        // Status colors
        healthy: '#3fb950',
        unhealthy: '#f85149',
        warning: '#d29922',
        disabled: '#6e7681',
        info: '#58a6ff',
        database: '#bc8cff',
      },
      fontFamily: {
        sans: ['Inter', 'system-ui', 'sans-serif'],
        mono: ['JetBrains Mono', 'Fira Code', 'monospace'],
      },
      borderRadius: {
        lg: 'var(--radius)',
        md: 'calc(var(--radius) - 2px)',
        sm: 'calc(var(--radius) - 4px)',
      },
      keyframes: {
        'pulse-healthy': {
          '0%, 100%': { boxShadow: '0 0 0 0 rgba(63, 185, 80, 0.4)' },
          '50%': { boxShadow: '0 0 0 4px rgba(63, 185, 80, 0)' },
        },
        'pulse-unhealthy': {
          '0%, 100%': { boxShadow: '0 0 0 0 rgba(248, 81, 73, 0.6)' },
          '50%': { boxShadow: '0 0 0 6px rgba(248, 81, 73, 0)' },
        },
        'count-up': {
          '0%': { opacity: '0', transform: 'translateY(10px)' },
          '100%': { opacity: '1', transform: 'translateY(0)' },
        },
        'slide-in': {
          '0%': { opacity: '0', transform: 'translateX(-10px)' },
          '100%': { opacity: '1', transform: 'translateX(0)' },
        },
      },
      animation: {
        'pulse-healthy': 'pulse-healthy 2s ease-in-out infinite',
        'pulse-unhealthy': 'pulse-unhealthy 1.5s ease-in-out infinite',
        'count-up': 'count-up 0.4s ease-out forwards',
        'slide-in': 'slide-in 0.2s ease-out forwards',
      },
    },
  },
  plugins: [tailwindcssAnimate],
}

export default config
