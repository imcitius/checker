import { Search, Settings, LogOut, User, Keyboard, Bell, Sun, Moon, Monitor } from 'lucide-react'
import { Link, useLocation } from 'react-router-dom'
import { Input } from '@/components/ui/input'
import {
  Select,
  SelectContent,
  SelectItem,
  SelectTrigger,
  SelectValue,
} from '@/components/ui/select'
import {
  DropdownMenu,
  DropdownMenuContent,
  DropdownMenuItem,
  DropdownMenuSeparator,
  DropdownMenuTrigger,
} from '@/components/ui/dropdown-menu'
import { Button } from '@/components/ui/button'
import { useTheme } from '@/lib/theme'
import { openCommandPalette } from '@/components/CommandPalette'

interface TopBarProps {
  search: string
  onSearchChange: (value: string) => void
  statusFilter: string
  onStatusFilterChange: (value: string) => void
  projectFilter: string
  onProjectFilterChange: (value: string) => void
  typeFilter: string
  onTypeFilterChange: (value: string) => void
  projects: string[]
  checkTypes: string[]
  searchRef: React.RefObject<HTMLInputElement | null>
  onOpenCommandPalette?: () => void
}

export function TopBar({
  search,
  onSearchChange,
  statusFilter,
  onStatusFilterChange,
  projectFilter,
  onProjectFilterChange,
  typeFilter,
  onTypeFilterChange,
  projects,
  checkTypes,
  searchRef,
  onOpenCommandPalette,
}: TopBarProps) {
  const location = useLocation()
  const { theme, setTheme } = useTheme()

  return (
    <header className="sticky top-0 z-40 border-b bg-card scanline-bg">
      <div className="mx-auto max-w-[1600px] px-4 py-2">
        <div className="flex items-center gap-4">
          {/* Logo */}
          <Link to="/" className="flex items-center gap-2 shrink-0">
            <div className="h-7 w-7 rounded bg-healthy/20 flex items-center justify-center">
              <span className="text-healthy font-mono font-bold text-sm">C</span>
            </div>
            <span className="font-semibold text-foreground hidden sm:inline">Checker</span>
          </Link>

          {/* Nav */}
          <nav className="flex items-center gap-1 shrink-0">
            <Link to="/">
              <Button
                variant={location.pathname === '/' ? 'secondary' : 'ghost'}
                size="sm"
              >
                Dashboard
              </Button>
            </Link>
            <Link to="/manage">
              <Button
                variant={location.pathname === '/manage' ? 'secondary' : 'ghost'}
                size="sm"
              >
                <Settings className="h-4 w-4 mr-1" />
                Manage
              </Button>
            </Link>
            <Link to="/alerts">
              <Button
                variant={location.pathname === '/alerts' ? 'secondary' : 'ghost'}
                size="sm"
              >
                <Bell className="h-4 w-4 mr-1" />
                Alerts
              </Button>
            </Link>
          </nav>

          {/* Search */}
          <div className="flex-1 max-w-md relative">
            <Search className="absolute left-2.5 top-1/2 -translate-y-1/2 h-4 w-4 text-muted-foreground" />
            <Input
              ref={searchRef}
              placeholder="Search checks...  (/)"
              value={search}
              onChange={(e) => onSearchChange(e.target.value)}
              className="pl-9 h-8 text-sm"
            />
          </div>

          {/* Filters */}
          <div className="hidden lg:flex items-center gap-2">
            <Select value={statusFilter} onValueChange={onStatusFilterChange}>
              <SelectTrigger className="h-8 w-[120px] text-xs">
                <SelectValue placeholder="Status" />
              </SelectTrigger>
              <SelectContent>
                <SelectItem value="all">All Status</SelectItem>
                <SelectItem value="healthy">Healthy</SelectItem>
                <SelectItem value="unhealthy">Unhealthy</SelectItem>
                <SelectItem value="disabled">Disabled</SelectItem>
                <SelectItem value="silenced">Silenced</SelectItem>
              </SelectContent>
            </Select>

            <Select value={projectFilter} onValueChange={onProjectFilterChange}>
              <SelectTrigger className="h-8 w-[130px] text-xs">
                <SelectValue placeholder="Project" />
              </SelectTrigger>
              <SelectContent>
                <SelectItem value="all">All Projects</SelectItem>
                {projects.map((p) => (
                  <SelectItem key={p} value={p}>
                    {p}
                  </SelectItem>
                ))}
              </SelectContent>
            </Select>

            <Select value={typeFilter} onValueChange={onTypeFilterChange}>
              <SelectTrigger className="h-8 w-[110px] text-xs">
                <SelectValue placeholder="Type" />
              </SelectTrigger>
              <SelectContent>
                <SelectItem value="all">All Types</SelectItem>
                {checkTypes.map((t) => (
                  <SelectItem key={t} value={t}>
                    {t}
                  </SelectItem>
                ))}
              </SelectContent>
            </Select>
          </div>

          {/* Command palette hint + Theme toggle + User menu */}
          <div className="flex items-center gap-1 shrink-0">
            <Button
              variant="ghost"
              size="sm"
              className="hidden sm:flex text-xs text-muted-foreground gap-1"
              onClick={onOpenCommandPalette ?? openCommandPalette}
            >
              <Keyboard className="h-3.5 w-3.5" />
              <kbd className="text-[10px]">⌘K</kbd>
            </Button>

            {/* Theme toggle */}
            <DropdownMenu>
              <DropdownMenuTrigger asChild>
                <Button variant="ghost" size="icon" className="h-8 w-8">
                  <Sun className="h-4 w-4 rotate-0 scale-100 transition-transform dark:-rotate-90 dark:scale-0" />
                  <Moon className="absolute h-4 w-4 rotate-90 scale-0 transition-transform dark:rotate-0 dark:scale-100" />
                  <span className="sr-only">Toggle theme</span>
                </Button>
              </DropdownMenuTrigger>
              <DropdownMenuContent align="end">
                <DropdownMenuItem onClick={() => setTheme('light')}>
                  <Sun className="mr-2 h-4 w-4" />
                  Light
                  {theme === 'light' && <span className="ml-auto text-xs text-muted-foreground">✓</span>}
                </DropdownMenuItem>
                <DropdownMenuItem onClick={() => setTheme('dark')}>
                  <Moon className="mr-2 h-4 w-4" />
                  Dark
                  {theme === 'dark' && <span className="ml-auto text-xs text-muted-foreground">✓</span>}
                </DropdownMenuItem>
                <DropdownMenuItem onClick={() => setTheme('system')}>
                  <Monitor className="mr-2 h-4 w-4" />
                  System
                  {theme === 'system' && <span className="ml-auto text-xs text-muted-foreground">✓</span>}
                </DropdownMenuItem>
              </DropdownMenuContent>
            </DropdownMenu>

            {/* User menu */}
            <DropdownMenu>
              <DropdownMenuTrigger asChild>
                <Button variant="ghost" size="icon" className="h-8 w-8">
                  <User className="h-4 w-4" />
                </Button>
              </DropdownMenuTrigger>
              <DropdownMenuContent align="end">
                <DropdownMenuItem onClick={onOpenCommandPalette ?? openCommandPalette}>
                  <Keyboard className="mr-2 h-4 w-4" />
                  Shortcuts
                </DropdownMenuItem>
                <DropdownMenuSeparator />
                <DropdownMenuItem asChild>
                  <a href="/auth/logout">
                    <LogOut className="mr-2 h-4 w-4" />
                    Logout
                  </a>
                </DropdownMenuItem>
              </DropdownMenuContent>
            </DropdownMenu>
          </div>
        </div>
      </div>
    </header>
  )
}
