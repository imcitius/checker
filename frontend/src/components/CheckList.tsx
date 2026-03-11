import { ProjectGroup } from '@/components/ProjectGroup'
import type { ProjectGroup as ProjectGroupType } from '@/hooks/useChecks'

interface CheckListProps {
  groups: ProjectGroupType[]
  collapsedGroups: Set<string>
  onToggleGroup: (name: string) => void
  selectedUUID: string | null
  expandedUUID: string | null
  onSelectCheck: (uuid: string) => void
  onToggleCheck: (uuid: string, enabled: boolean) => void
}

export function CheckList({
  groups,
  collapsedGroups,
  onToggleGroup,
  selectedUUID,
  expandedUUID,
  onSelectCheck,
  onToggleCheck,
}: CheckListProps) {
  if (groups.length === 0) {
    return (
      <div className="text-center py-12 text-muted-foreground text-sm">
        No checks found
      </div>
    )
  }

  return (
    <div className="rounded-lg border bg-card overflow-hidden">
      {/* Header */}
      <div className="flex items-center gap-3 px-3 py-1.5 text-xs text-muted-foreground border-b bg-[hsl(215_14%_10%)]">
        <span className="w-2.5" />
        <span className="w-[180px] shrink-0">Name</span>
        <span className="w-[52px] shrink-0">Type</span>
        <span className="w-[70px] shrink-0">Status</span>
        <span className="flex-1">Target</span>
        <span className="w-[40px] shrink-0 text-right">Freq</span>
        <span className="w-[60px] shrink-0 text-right">Last Run</span>
        <span className="w-9 shrink-0" />
        <span className="w-3 shrink-0" />
      </div>

      {groups.map((group) => (
        <ProjectGroup
          key={group.name}
          name={group.name}
          checks={group.checks}
          healthyCount={group.healthyCount}
          failingCount={group.failingCount}
          isOpen={!collapsedGroups.has(group.name)}
          onToggle={() => onToggleGroup(group.name)}
          selectedUUID={selectedUUID}
          expandedUUID={expandedUUID}
          onSelectCheck={onSelectCheck}
          onToggleCheck={onToggleCheck}
        />
      ))}
    </div>
  )
}
