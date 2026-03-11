import { useEffect, useState } from 'react'
import type { CheckStats } from '@/hooks/useChecks'

interface MetricsRowProps {
  stats: CheckStats
}

function AnimatedCount({ value, delay }: { value: number; delay: number }) {
  const [display, setDisplay] = useState(0)
  const [animate, setAnimate] = useState(false)

  useEffect(() => {
    const timer = setTimeout(() => {
      setDisplay(value)
      setAnimate(true)
    }, delay)
    return () => clearTimeout(timer)
  }, [value, delay])

  useEffect(() => {
    if (!animate) return
    const t = setTimeout(() => setAnimate(false), 400)
    return () => clearTimeout(t)
  }, [animate])

  return (
    <span className={animate ? 'animate-count-up' : ''}>
      {display}
    </span>
  )
}

export function MetricsRow({ stats }: MetricsRowProps) {
  const cards = [
    { label: 'Total', value: stats.total, color: 'border-info/40', glow: '' },
    { label: 'Healthy', value: stats.healthy, color: 'border-healthy/40', glow: 'glow-healthy' },
    { label: 'Failing', value: stats.unhealthy, color: 'border-unhealthy/40', glow: stats.unhealthy > 0 ? 'glow-unhealthy' : '' },
    { label: 'Disabled', value: stats.disabled, color: 'border-disabled/40', glow: '' },
  ]

  return (
    <div className="grid grid-cols-2 sm:grid-cols-4 gap-3">
      {cards.map((card, i) => (
        <div
          key={card.label}
          className={`rounded-lg border bg-card p-3 text-center ${card.color} ${card.glow}`}
        >
          <div className="text-2xl font-bold font-mono text-foreground">
            <AnimatedCount value={card.value} delay={i * 80} />
          </div>
          <div className="text-xs text-muted-foreground mt-0.5">{card.label}</div>
        </div>
      ))}
    </div>
  )
}
