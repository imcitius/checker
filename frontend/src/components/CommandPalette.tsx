import { useState, useEffect, useCallback } from 'react'
import { useNavigate } from 'react-router-dom'
import {
  CommandDialog,
  CommandInput,
  CommandList,
  CommandEmpty,
  CommandGroup,
  CommandItem,
  CommandShortcut,
} from '@/components/ui/command'
import { useTheme } from '@/lib/theme'
import { api, type CheckDefinition } from '@/lib/api'
import {
  Settings,
  Bell,
  Sun,
  Moon,
  LayoutDashboard,
  Plus,
  Upload,
  Search,
} from 'lucide-react'

/**
 * Custom event name used to open the command palette from anywhere
 * (e.g. the TopBar button). Dispatch via:
 *   window.dispatchEvent(new Event('open-command-palette'))
 */
const OPEN_EVENT = 'open-command-palette'

export function CommandPalette() {
  const [open, setOpen] = useState(false)
  const [checks, setChecks] = useState<CheckDefinition[]>([])
  const navigate = useNavigate()
  const { theme, setTheme } = useTheme()

  // ---------- keyboard shortcut: Cmd+K / Ctrl+K ----------
  useEffect(() => {
    function handleKeyDown(e: KeyboardEvent) {
      if ((e.metaKey || e.ctrlKey) && e.key === 'k') {
        e.preventDefault()
        setOpen((prev) => !prev)
      }
    }
    window.addEventListener('keydown', handleKeyDown)
    return () => window.removeEventListener('keydown', handleKeyDown)
  }, [])

  // ---------- listen for custom "open" event from TopBar ----------
  useEffect(() => {
    function handleOpen() {
      setOpen(true)
    }
    window.addEventListener(OPEN_EVENT, handleOpen)
    return () => window.removeEventListener(OPEN_EVENT, handleOpen)
  }, [])

  // ---------- fetch checks when palette opens ----------
  useEffect(() => {
    if (!open) return
    api.getChecks().then(setChecks).catch(() => setChecks([]))
  }, [open])

  const close = useCallback(() => setOpen(false), [])

  const go = useCallback(
    (path: string) => {
      navigate(path)
      close()
    },
    [navigate, close]
  )

  return (
    <CommandDialog open={open} onOpenChange={setOpen}>
      <CommandInput placeholder="Type a command or search..." />
      <CommandList>
        <CommandEmpty>No results found.</CommandEmpty>

        {/* ---------- Check search ---------- */}
        {checks.length > 0 && (
          <CommandGroup heading="Checks">
            {checks.slice(0, 15).map((check) => (
              <CommandItem
                key={check.uuid}
                value={`check-${check.name}-${check.uuid}`}
                onSelect={() => {
                  navigate(`/?check=${check.uuid}`)
                  close()
                }}
              >
                <span
                  className={`inline-block h-2 w-2 rounded-full mr-2 shrink-0 ${
                    !check.enabled ? 'bg-disabled' : 'bg-healthy'
                  }`}
                />
                <span className="truncate">{check.name}</span>
                <span className="ml-auto text-xs text-muted-foreground">
                  {check.type}
                </span>
              </CommandItem>
            ))}
          </CommandGroup>
        )}

        {/* ---------- Navigation ---------- */}
        <CommandGroup heading="Navigation">
          <CommandItem onSelect={() => go('/')}>
            <LayoutDashboard className="mr-2 h-4 w-4" />
            Go to Dashboard
          </CommandItem>
          <CommandItem onSelect={() => go('/manage')}>
            <Settings className="mr-2 h-4 w-4" />
            Go to Management
          </CommandItem>
          <CommandItem onSelect={() => go('/alerts')}>
            <Bell className="mr-2 h-4 w-4" />
            Go to Alerts
          </CommandItem>
        </CommandGroup>

        {/* ---------- Actions ---------- */}
        <CommandGroup heading="Actions">
          <CommandItem
            onSelect={() => {
              navigate('/manage?action=create')
              close()
            }}
          >
            <Plus className="mr-2 h-4 w-4" />
            Create New Check
          </CommandItem>
          <CommandItem
            onSelect={() => {
              navigate('/manage?action=import')
              close()
            }}
          >
            <Upload className="mr-2 h-4 w-4" />
            Import Checks (YAML)
          </CommandItem>
          <CommandItem
            onSelect={() => {
              setTheme(theme === 'dark' ? 'light' : 'dark')
              close()
            }}
          >
            {theme === 'dark' ? (
              <Sun className="mr-2 h-4 w-4" />
            ) : (
              <Moon className="mr-2 h-4 w-4" />
            )}
            Toggle Theme
          </CommandItem>
        </CommandGroup>

        {/* ---------- Keyboard shortcuts reference ---------- */}
        <CommandGroup heading="Keyboard Shortcuts">
          <CommandItem disabled>
            <Search className="mr-2 h-4 w-4" />
            Command Palette
            <CommandShortcut>⌘K</CommandShortcut>
          </CommandItem>
          <CommandItem disabled>
            Select
            <CommandShortcut>↵</CommandShortcut>
          </CommandItem>
          <CommandItem disabled>
            Close
            <CommandShortcut>Esc</CommandShortcut>
          </CommandItem>
        </CommandGroup>
      </CommandList>
    </CommandDialog>
  )
}

/** Helper to open the command palette from anywhere */
export function openCommandPalette() {
  window.dispatchEvent(new Event(OPEN_EVENT))
}
