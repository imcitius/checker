import { ChevronRight, Layers } from 'lucide-react'
import { Collapsible, CollapsibleContent, CollapsibleTrigger } from '@/components/ui/collapsible'
import { Badge } from '@/components/ui/badge'
import { CheckRow } from '@/components/CheckRow'
import { CheckDetails } from '@/components/CheckDetails'
import type { ProjectGroup as ProjectGroupType } from '@/hooks/useChecks'
import { cn } from '@/lib/utils'

interface ProjectGroupProps {
  group: ProjectGroupType
  collapsedGroups: Set<string>
  onToggleGroup: (key: string) => void
  selectedUUID: string | null
  expandedUUID: string | null
  onSelectCheck: (uuid: string) => void
  onToggleCheck: (uuid: string, enabled: boolean) => void
}

export function ProjectGroup({
  group,
  collapsedGroups,
  onToggleGroup,
  selectedUUID,
  expandedUUID,
  onSelectCheck,
  onToggleCheck,
}: ProjectGroupProps) {
  const projectKey = `p:${group.name}`
  const isProjectOpen = !collapsedGroups.has(projectKey)
  const hasSubGroups = group.subGroups.length > 1 || (group.subGroups.length === 1 && group.subGroups[0].name !== '')

  return (
    <Collapsible open={isProjectOpen} onOpenChange={() => onToggleGroup(projectKey)}>
      <CollapsibleTrigger className="flex items-center gap-2 w-full px-3 py-1.5 hover:bg-muted/50 transition-colors text-sm group">
        <ChevronRight
          className={cn('h-3.5 w-3.5 text-muted-foreground transition-transform', isProjectOpen && 'rotate-90')}
        />
        <span className="font-semibold text-foreground">{group.name}</span>
        <span className="text-muted-foreground text-xs">
          {group.checks.length} check{group.checks.length !== 1 ? 's' : ''}
        </span>
        {group.failingCount > 0 && (
          <Badge variant="unhealthy" className="text-[10px] px-1.5 py-0">
            {group.failingCount} failing
          </Badge>
        )}
        {group.failingCount === 0 && group.checks.length > 0 && (
          <Badge variant="healthy" className="text-[10px] px-1.5 py-0 opacity-60">
            all ok
          </Badge>
        )}
      </CollapsibleTrigger>
      <CollapsibleContent>
        {hasSubGroups ? (
          group.subGroups.map((sg) => {
            const sgKey = `g:${group.name}/${sg.name}`
            const isSgOpen = !collapsedGroups.has(sgKey)

            return (
              <Collapsible key={sg.name} open={isSgOpen} onOpenChange={() => onToggleGroup(sgKey)}>
                <CollapsibleTrigger className="flex items-center gap-2 w-full px-3 py-1 pl-7 hover:bg-muted/30 transition-colors text-xs group border-t border-border/30">
                  <ChevronRight
                    className={cn('h-3 w-3 text-muted-foreground transition-transform', isSgOpen && 'rotate-90')}
                  />
                  <Layers className="h-3 w-3 text-muted-foreground" />
                  <span className="font-medium text-muted-foreground">{sg.name || '(no group)'}</span>
                  <span className="text-muted-foreground/60 text-[11px]">
                    {sg.checks.length}
                  </span>
                  {sg.failingCount > 0 && (
                    <Badge variant="unhealthy" className="text-[9px] px-1 py-0">
                      {sg.failingCount}
                    </Badge>
                  )}
                </CollapsibleTrigger>
                <CollapsibleContent>
                  {sg.checks.map((check) => (
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
          })
        ) : (
          // Single group or no group — render checks directly
          group.subGroups.flatMap((sg) =>
            sg.checks.map((check) => (
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
            ))
          )
        )}
      </CollapsibleContent>
    </Collapsible>
  )
}
