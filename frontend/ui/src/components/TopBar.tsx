import { useState, useCallback, useEffect, useRef, type ReactNode } from 'react'
import { Search, Settings, LogOut, User, Keyboard, Bell, Sun, Moon, Monitor, Menu, X, Cog } from 'lucide-react'
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

/** Navigation item for extending the TopBar nav */
export interface NavItem {
  /** Route path */
  path: string
  /** Display label */
  label: string
  /** Optional icon element */
  icon?: ReactNode
}

/** Menu item for extending the user dropdown menu */
export interface MenuItem {
  /** Unique key */
  key: string
  /** Display label */
  label: string
  /** Optional icon element */
  icon?: ReactNode
  /** Click handler */
  onClick?: () => void
  /** If set, renders as a link */
  href?: string
  /** If true, renders a separator before this item */
  separatorBefore?: boolean
}

export interface TopBarProps {
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

  // --- Customization props ---
  /** Brand name displayed next to the logo. Default: "Checker" */
  brandName?: string
  /** Extra navigation items added after the default nav links */
  extraNavItems?: NavItem[]
  /** Extra items added to the user dropdown menu */
  userMenuItems?: MenuItem[]
  /** Custom logout handler. Default: navigates to /auth/logout */
  onLogout?: () => void
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
  brandName = 'Checker',
  extraNavItems = [],
  userMenuItems = [],
  onLogout,
}: TopBarProps) {
  const location = useLocation()
  const { theme, setTheme } = useTheme()
  const [mobileMenuOpen, setMobileMenuOpen] = useState(false)
  const drawerRef = useRef<HTMLDivElement>(null)

  // Close mobile menu on route change
  useEffect(() => {
    setMobileMenuOpen(false)
  }, [location.pathname])

  // Close mobile menu on backdrop click or click outside
  const handleBackdropClick = useCallback((e: React.MouseEvent) => {
    if (drawerRef.current && !drawerRef.current.contains(e.target as Node)) {
      setMobileMenuOpen(false)
    }
  }, [])

  // Close on Escape key
  useEffect(() => {
    if (!mobileMenuOpen) return
    const handleEscape = (e: KeyboardEvent) => {
      if (e.key === 'Escape') setMobileMenuOpen(false)
    }
    document.addEventListener('keydown', handleEscape)
    return () => document.removeEventListener('keydown', handleEscape)
  }, [mobileMenuOpen])

  // Prevent body scroll when mobile menu is open
  useEffect(() => {
    if (mobileMenuOpen) {
      document.body.style.overflow = 'hidden'
    } else {
      document.body.style.overflow = ''
    }
    return () => { document.body.style.overflow = '' }
  }, [mobileMenuOpen])

  const handleLogout = onLogout
    ? () => { onLogout() }
    : undefined

  const logoutElement = handleLogout ? (
    <button className="flex items-center w-full" onClick={handleLogout}>
      <LogOut className="mr-2 h-4 w-4" />
      Logout
    </button>
  ) : (
    <a href="/auth/logout" className="flex items-center">
      <LogOut className="mr-2 h-4 w-4" />
      Logout
    </a>
  )

  const defaultNavItems: NavItem[] = [
    { path: '/', label: 'Dashboard' },
    { path: '/manage', label: 'Manage', icon: <Settings className="h-4 w-4 mr-1" /> },
    { path: '/alerts', label: 'Alerts', icon: <Bell className="h-4 w-4 mr-1" /> },
    { path: '/settings', label: 'Settings', icon: <Cog className="h-4 w-4 mr-1" /> },
  ]

  const allNavItems = [...defaultNavItems, ...extraNavItems]

  return (
    <>
      <header className="sticky top-0 z-40 border-b bg-card scanline-bg">
        <div className="mx-auto max-w-[1600px] px-4 py-2">
          <div className="flex items-center gap-2 sm:gap-4">
            {/* Hamburger menu button -- visible on small screens */}
            <Button
              variant="ghost"
              size="icon"
              className="sm:hidden h-9 w-9 min-h-[44px] min-w-[44px] shrink-0"
              onClick={() => setMobileMenuOpen(true)}
              aria-label="Open navigation menu"
            >
              <Menu className="h-5 w-5" />
            </Button>

            {/* Logo */}
            <Link to="/" className="flex items-center gap-2 shrink-0">
              <div className="h-7 w-7 rounded bg-healthy/20 flex items-center justify-center">
                <span className="text-healthy font-mono font-bold text-sm">{brandName.charAt(0)}</span>
              </div>
              <span className="font-semibold text-foreground hidden sm:inline">{brandName}</span>
            </Link>

            {/* Nav -- hidden on mobile, shown on sm+ */}
            <nav className="hidden sm:flex items-center gap-1 shrink-0">
              {allNavItems.map((item) => (
                <Link key={item.path} to={item.path}>
                  <Button
                    variant={location.pathname === item.path ? 'secondary' : 'ghost'}
                    size="sm"
                    className="min-h-[44px]"
                  >
                    {item.icon}
                    {item.label}
                  </Button>
                </Link>
              ))}
            </nav>

            {/* Search */}
            <div className="flex-1 min-w-0 max-w-md relative">
              <Search className="absolute left-2.5 top-1/2 -translate-y-1/2 h-4 w-4 text-muted-foreground" />
              <Input
                ref={searchRef}
                placeholder="Search...  (/)"
                value={search}
                onChange={(e) => onSearchChange(e.target.value)}
                className="pl-9 h-9 min-h-[44px] text-sm"
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
                <kbd className="text-[10px]">&#8984;K</kbd>
              </Button>

              {/* Theme toggle */}
              <DropdownMenu>
                <DropdownMenuTrigger asChild>
                  <Button variant="ghost" size="icon" className="h-9 w-9 min-h-[44px] min-w-[44px]">
                    <Sun className="h-4 w-4 rotate-0 scale-100 transition-transform dark:-rotate-90 dark:scale-0" />
                    <Moon className="absolute h-4 w-4 rotate-90 scale-0 transition-transform dark:rotate-0 dark:scale-100" />
                    <span className="sr-only">Toggle theme</span>
                  </Button>
                </DropdownMenuTrigger>
                <DropdownMenuContent align="end">
                  <DropdownMenuItem onClick={() => setTheme('light')}>
                    <Sun className="mr-2 h-4 w-4" />
                    Light
                    {theme === 'light' && <span className="ml-auto text-xs text-muted-foreground">&#10003;</span>}
                  </DropdownMenuItem>
                  <DropdownMenuItem onClick={() => setTheme('dark')}>
                    <Moon className="mr-2 h-4 w-4" />
                    Dark
                    {theme === 'dark' && <span className="ml-auto text-xs text-muted-foreground">&#10003;</span>}
                  </DropdownMenuItem>
                  <DropdownMenuItem onClick={() => setTheme('system')}>
                    <Monitor className="mr-2 h-4 w-4" />
                    System
                    {theme === 'system' && <span className="ml-auto text-xs text-muted-foreground">&#10003;</span>}
                  </DropdownMenuItem>
                </DropdownMenuContent>
              </DropdownMenu>

              {/* User menu -- hidden on mobile (available in drawer) */}
              <DropdownMenu>
                <DropdownMenuTrigger asChild>
                  <Button variant="ghost" size="icon" className="hidden sm:flex h-9 w-9 min-h-[44px] min-w-[44px]">
                    <User className="h-4 w-4" />
                  </Button>
                </DropdownMenuTrigger>
                <DropdownMenuContent align="end">
                  <DropdownMenuItem onClick={onOpenCommandPalette ?? openCommandPalette}>
                    <Keyboard className="mr-2 h-4 w-4" />
                    Shortcuts
                  </DropdownMenuItem>
                  {userMenuItems.map((item) => (
                    <span key={item.key}>
                      {item.separatorBefore && <DropdownMenuSeparator />}
                      <DropdownMenuItem
                        onClick={item.onClick}
                        asChild={!!item.href}
                      >
                        {item.href ? (
                          <a href={item.href}>
                            {item.icon}
                            {item.label}
                          </a>
                        ) : (
                          <span>{item.icon}{item.label}</span>
                        )}
                      </DropdownMenuItem>
                    </span>
                  ))}
                  <DropdownMenuSeparator />
                  <DropdownMenuItem asChild={!handleLogout} onClick={handleLogout}>
                    {logoutElement}
                  </DropdownMenuItem>
                </DropdownMenuContent>
              </DropdownMenu>
            </div>
          </div>
        </div>
      </header>

      {/* Mobile slide-in drawer */}
      {mobileMenuOpen && (
        <div
          className="fixed inset-0 z-50 sm:hidden"
          onClick={handleBackdropClick}
        >
          {/* Backdrop */}
          <div className="absolute inset-0 bg-black/50 animate-in fade-in duration-200" />

          {/* Drawer */}
          <div
            ref={drawerRef}
            className="absolute top-0 left-0 bottom-0 w-[280px] max-w-[80vw] bg-card border-r shadow-xl animate-in slide-in-from-left duration-200 flex flex-col"
          >
            {/* Drawer header */}
            <div className="flex items-center justify-between px-4 py-3 border-b">
              <Link to="/" className="flex items-center gap-2" onClick={() => setMobileMenuOpen(false)}>
                <div className="h-7 w-7 rounded bg-healthy/20 flex items-center justify-center">
                  <span className="text-healthy font-mono font-bold text-sm">{brandName.charAt(0)}</span>
                </div>
                <span className="font-semibold text-foreground">{brandName}</span>
              </Link>
              <Button
                variant="ghost"
                size="icon"
                className="h-9 w-9 min-h-[44px] min-w-[44px]"
                onClick={() => setMobileMenuOpen(false)}
                aria-label="Close navigation menu"
              >
                <X className="h-5 w-5" />
              </Button>
            </div>

            {/* Nav links */}
            <nav className="flex flex-col gap-1 p-4">
              {allNavItems.map((item) => (
                <Link key={item.path} to={item.path} onClick={() => setMobileMenuOpen(false)}>
                  <Button
                    variant={location.pathname === item.path ? 'secondary' : 'ghost'}
                    className="w-full justify-start min-h-[44px]"
                  >
                    {item.icon && <span className="mr-2">{item.icon}</span>}
                    {item.label}
                  </Button>
                </Link>
              ))}
            </nav>

            {/* Filters (visible in drawer on mobile) */}
            <div className="px-4 py-3 border-t space-y-3 lg:hidden">
              <p className="text-xs font-medium text-muted-foreground uppercase tracking-wide">Filters</p>
              <Select value={statusFilter} onValueChange={(v) => { onStatusFilterChange(v) }}>
                <SelectTrigger className="h-9 min-h-[44px] text-xs">
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

              <Select value={projectFilter} onValueChange={(v) => { onProjectFilterChange(v) }}>
                <SelectTrigger className="h-9 min-h-[44px] text-xs">
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

              <Select value={typeFilter} onValueChange={(v) => { onTypeFilterChange(v) }}>
                <SelectTrigger className="h-9 min-h-[44px] text-xs">
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

            {/* Bottom actions */}
            <div className="mt-auto border-t p-4 space-y-1">
              <Button
                variant="ghost"
                className="w-full justify-start min-h-[44px]"
                onClick={() => {
                  setMobileMenuOpen(false)
                  ;(onOpenCommandPalette ?? openCommandPalette)()
                }}
              >
                <Keyboard className="h-4 w-4 mr-2" />
                Shortcuts
              </Button>
              {userMenuItems.map((item) => (
                <Button
                  key={item.key}
                  variant="ghost"
                  className="w-full justify-start min-h-[44px]"
                  onClick={() => {
                    setMobileMenuOpen(false)
                    item.onClick?.()
                  }}
                  asChild={!!item.href}
                >
                  {item.href ? (
                    <a href={item.href}>
                      {item.icon && <span className="mr-2">{item.icon}</span>}
                      {item.label}
                    </a>
                  ) : (
                    <span>
                      {item.icon && <span className="mr-2">{item.icon}</span>}
                      {item.label}
                    </span>
                  )}
                </Button>
              ))}
              <Button
                variant="ghost"
                className="w-full justify-start min-h-[44px]"
                onClick={handleLogout}
                asChild={!handleLogout}
              >
                {handleLogout ? (
                  <span>
                    <LogOut className="h-4 w-4 mr-2" />
                    Logout
                  </span>
                ) : (
                  <a href="/auth/logout">
                    <LogOut className="h-4 w-4 mr-2" />
                    Logout
                  </a>
                )}
              </Button>
            </div>
          </div>
        </div>
      )}
    </>
  )
}
