import { ChevronRight } from 'lucide-react'
import { Collapsible, CollapsibleContent, CollapsibleTrigger } from '@/components/ui/collapsible'
import { Badge } from '@/components/ui/badge'
import { CheckRow } from '@/components/CheckRow'
import { CheckDetails } from '@/components/CheckDetails'
import type { Check } from '@/lib/websocket'
import { cn } from '@/lib/utils'

interface ProjectGroupProps {
  name: string
  checks: Check[]
  healthyCount: number
  failingCount: number
  isOpen: boolean
  onToggle: () => void
  selectedUUID: string | null
  expandedUUID: string | null
  onSelectCheck: (uuid: string) => void
  onToggleCheck: (uuid: string, enabled: boolean) => void
}

export function ProjectGroup({
  name,
  checks,
  healthyCount,
  failingCount,
  isOpen,
  onToggle,
  selectedUUID,
  expandedUUID,
  onSelectCheck,
  onToggleCheck,
}: ProjectGroupProps) {
  return (
    <Collapsible open={isOpen} onOpenChange={onToggle}>
      <CollapsibleTrigger className="flex items-center gap-2 w-full px-3 py-1.5 hover:bg-muted/50 transition-colors text-sm group">
        <ChevronRight
          className={cn('h-3.5 w-3.5 text-muted-foreground transition-transform', isOpen && 'rotate-90')}
        />
        <span className="font-semibold text-foreground">{name}</span>
        <span className="text-muted-foreground text-xs">
          {checks.length} check{checks.length !== 1 ? 's' : ''}
        </span>
        {failingCount > 0 && (
          <Badge variant="unhealthy" className="text-[10px] px-1.5 py-0">
            {failingCount} failing
          </Badge>
        )}
        {failingCount === 0 && checks.length > 0 && (
          <Badge variant="healthy" className="text-[10px] px-1.5 py-0 opacity-60">
            all ok
          </Badge>
        )}
      </CollapsibleTrigger>
      <CollapsibleContent>
        {checks.map((check) => (
          <div key={check.UUID}>
            <CheckRow
              check={check}
              isSelected={selectedUUID === check.UUID}
              isExpanded={expandedUUID === check.UUID}
              onSelect={() => onSelectCheck(check.UUID)}
              onToggle={onToggleCheck}
            />
            {expandedUUID === check.UUID && <CheckDetails check={check} />}
          </div>
        ))}
      </CollapsibleContent>
    </Collapsible>
  )
}
