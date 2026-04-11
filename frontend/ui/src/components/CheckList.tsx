import { ProjectGroup } from '@/components/ProjectGroup'
import type { ProjectGroup as ProjectGroupType } from '@/hooks/useChecks'
import { ArrowUp, ArrowDown, ArrowUpDown } from 'lucide-react'

type SortColumn = 'name' | 'type' | 'status' | 'host' | 'frequency'
type SortDirection = 'asc' | 'desc'

interface CheckListProps {
  groups: ProjectGroupType[]
  collapsedGroups: Set<string>
  onToggleGroup: (name: string) => void
  selectedUUID: string | null
  expandedUUID: string | null
  onSelectCheck: (uuid: string) => void
  sortColumn: SortColumn | null
  sortDirection: SortDirection
  onSort: (column: SortColumn) => void
}

function SortIcon({ column, sortColumn, sortDirection }: { column: SortColumn; sortColumn: SortColumn | null; sortDirection: SortDirection }) {
  if (sortColumn !== column) return <ArrowUpDown className="h-3 w-3 ml-0.5 opacity-40" />
  if (sortDirection === 'asc') return <ArrowUp className="h-3 w-3 ml-0.5" />
  return <ArrowDown className="h-3 w-3 ml-0.5" />
}

export function CheckList({
  groups,
  collapsedGroups,
  onToggleGroup,
  selectedUUID,
  expandedUUID,
  onSelectCheck,
  sortColumn,
  sortDirection,
  onSort,
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
      <div className="flex items-center gap-3 px-3 py-1.5 text-xs text-muted-foreground border-b bg-muted/50">
        <span className="w-2.5" />
        <button className="w-[180px] shrink-0 inline-flex items-center cursor-pointer hover:text-foreground transition-colors select-none" onClick={() => onSort('name')}>
          Name<SortIcon column="name" sortColumn={sortColumn} sortDirection={sortDirection} />
        </button>
        <button className="w-[52px] shrink-0 inline-flex items-center cursor-pointer hover:text-foreground transition-colors select-none" onClick={() => onSort('type')}>
          Type<SortIcon column="type" sortColumn={sortColumn} sortDirection={sortDirection} />
        </button>
        <button className="w-[70px] shrink-0 inline-flex items-center cursor-pointer hover:text-foreground transition-colors select-none" onClick={() => onSort('status')}>
          Status<SortIcon column="status" sortColumn={sortColumn} sortDirection={sortDirection} />
        </button>
        <button className="flex-1 inline-flex items-center cursor-pointer hover:text-foreground transition-colors select-none" onClick={() => onSort('host')}>
          Target<SortIcon column="host" sortColumn={sortColumn} sortDirection={sortDirection} />
        </button>
        <button className="w-[40px] shrink-0 inline-flex items-center justify-end cursor-pointer hover:text-foreground transition-colors select-none" onClick={() => onSort('frequency')}>
          Freq<SortIcon column="frequency" sortColumn={sortColumn} sortDirection={sortDirection} />
        </button>
        <span className="w-[60px] shrink-0 text-right">Last Run</span>
        <span className="w-4 shrink-0" />
      </div>

      {groups.map((group) => (
        <ProjectGroup
          key={group.name}
          group={group}
          collapsedGroups={collapsedGroups}
          onToggleGroup={onToggleGroup}
          selectedUUID={selectedUUID}
          expandedUUID={expandedUUID}
          onSelectCheck={onSelectCheck}
        />
      ))}
    </div>
  )
}
