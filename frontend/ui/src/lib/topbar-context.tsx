import { createContext, useContext, type ReactNode } from 'react'
import type { NavItem, MenuItem } from '@/components/TopBar'

/** Configuration props that customize the TopBar across all pages. */
export interface TopBarConfig {
  /** Brand name displayed next to the logo. Default: "Checker" */
  brandName?: string
  /** Extra navigation items added after the default nav links */
  extraNavItems?: NavItem[]
  /** Extra items added to the user dropdown menu */
  userMenuItems?: MenuItem[]
  /** Custom logout handler. Default: navigates to /auth/logout */
  onLogout?: () => void
}

const TopBarConfigContext = createContext<TopBarConfig>({})

export interface TopBarConfigProviderProps extends TopBarConfig {
  children: ReactNode
}

/**
 * Provides TopBar customization to all page components.
 *
 * Wrap your app (or a subtree) with this provider so that pages like
 * Dashboard, Management, Alerts, and Settings automatically pick up
 * branding, extra nav items, user menu items, and logout behavior.
 *
 * ```tsx
 * <TopBarConfigProvider brandName="Ensafely" onLogout={handleLogout}>
 *   <Routes>...</Routes>
 * </TopBarConfigProvider>
 * ```
 */
export function TopBarConfigProvider({
  children,
  brandName,
  extraNavItems,
  userMenuItems,
  onLogout,
}: TopBarConfigProviderProps) {
  return (
    <TopBarConfigContext.Provider
      value={{ brandName, extraNavItems, userMenuItems, onLogout }}
    >
      {children}
    </TopBarConfigContext.Provider>
  )
}

/** Read TopBar customization config from the nearest TopBarConfigProvider. */
export function useTopBarConfig(): TopBarConfig {
  return useContext(TopBarConfigContext)
}
