import { cn } from '@/lib/utils'

interface StatusDotProps {
  healthy: boolean
  enabled: boolean
  silenced?: boolean
  partiallySilenced?: boolean
  size?: 'sm' | 'md'
}

export function StatusDot({ healthy, enabled, silenced, partiallySilenced, size = 'md' }: StatusDotProps) {
  const sizeClass = size === 'sm' ? 'h-2 w-2' : 'h-2.5 w-2.5'

  if (!enabled) {
    return <span className={cn('inline-block rounded-full bg-disabled', sizeClass)} />
  }

  if (silenced) {
    return (
      <span
        className={cn(
          'inline-block rounded-full',
          sizeClass,
          healthy ? 'bg-healthy opacity-50' : 'bg-warning opacity-70'
        )}
      />
    )
  }

  if (partiallySilenced) {
    return (
      <span
        className={cn(
          'inline-block rounded-full',
          sizeClass,
          healthy ? 'bg-healthy opacity-70' : 'bg-warning opacity-85'
        )}
      />
    )
  }

  return (
    <span
      className={cn(
        'inline-block rounded-full',
        sizeClass,
        healthy
          ? 'bg-healthy animate-pulse-healthy'
          : 'bg-unhealthy animate-pulse-unhealthy'
      )}
    />
  )
}
