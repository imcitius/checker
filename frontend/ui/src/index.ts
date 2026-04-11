/**
 * @ensafely/checker-ui — Shared React component library for the Checker monitoring tool.
 *
 * This library provides all UI components, pages, hooks, and utilities needed
 * to build both the standalone Checker frontend and the cloud tenant overlay.
 *
 * Components accept customization props so that cloud can extend them without
 * build-time patching.
 */

// ─── CSS (import in your app: import '@ensafely/checker-ui/styles.css') ───
import './globals.css'

// ─── shadcn/ui Primitives ───
export { Badge, badgeVariants, type BadgeProps } from './components/ui/badge'
export { Button, buttonVariants, type ButtonProps } from './components/ui/button'
export { Collapsible, CollapsibleTrigger, CollapsibleContent } from './components/ui/collapsible'
export { Combobox } from './components/ui/combobox'
export {
  Command, CommandDialog, CommandInput, CommandList,
  CommandEmpty, CommandGroup, CommandItem, CommandShortcut,
} from './components/ui/command'
export {
  Dialog, DialogPortal, DialogOverlay, DialogTrigger, DialogClose,
  DialogContent, DialogHeader, DialogFooter, DialogTitle, DialogDescription,
} from './components/ui/dialog'
export {
  DropdownMenu, DropdownMenuTrigger, DropdownMenuContent,
  DropdownMenuItem, DropdownMenuSeparator, DropdownMenuSub,
  DropdownMenuSubTrigger, DropdownMenuSubContent,
} from './components/ui/dropdown-menu'
export { Input } from './components/ui/input'
export { Popover, PopoverTrigger, PopoverContent, PopoverAnchor } from './components/ui/popover'
export { ScrollArea, ScrollBar } from './components/ui/scroll-area'
export {
  Select, SelectGroup, SelectValue, SelectTrigger,
  SelectContent, SelectItem,
} from './components/ui/select'
export { Separator } from './components/ui/separator'
export {
  Sheet, SheetPortal, SheetOverlay, SheetTrigger, SheetClose,
  SheetContent, SheetHeader, SheetFooter, SheetTitle, SheetDescription,
} from './components/ui/sheet'
export { Switch } from './components/ui/switch'
export { Tabs, TabsList, TabsTrigger, TabsContent } from './components/ui/tabs'
export { Tooltip, TooltipTrigger, TooltipContent, TooltipProvider } from './components/ui/tooltip'

// ─── Components (with customization props) ───
export { TopBar, type TopBarProps, type NavItem, type MenuItem } from './components/TopBar'
export { CheckRow, type CheckRowProps, type ExtraBadgeRenderer as CheckRowExtraBadgeRenderer } from './components/CheckRow'
export { CheckDetails, type CheckDetailsProps, type ExtraBadgeRenderer as CheckDetailsExtraBadgeRenderer } from './components/CheckDetails'
export {
  CheckEditDrawer, type CheckEditDrawerProps,
  type ExtraFieldRenderer, type ExtraSectionRenderer, type FieldTooltips,
} from './components/CheckEditDrawer'
export { CommandPalette, openCommandPalette } from './components/CommandPalette'

// ─── Components (no customization needed) ───
export { CheckList } from './components/CheckList'
export { EventLog } from './components/EventLog'
export { HealthMap } from './components/HealthMap'
export { ImportDialog } from './components/ImportDialog'
export { MetricsRow } from './components/MetricsRow'
export { ProjectGroup } from './components/ProjectGroup'
export { StatusBar } from './components/StatusBar'
export { StatusDot } from './components/StatusDot'
export { VersionBadge } from './components/VersionBadge'

// ─── Pages ───
export { Dashboard } from './pages/Dashboard'
export { Management, type ManagementProps } from './pages/Management'
export { Alerts } from './pages/Alerts'
export { Settings, type SettingsProps, type ConfigField } from './pages/Settings'
export { Login } from './pages/Login'

// ─── Hooks ───
export { useAlerts } from './hooks/useAlerts'
export { useChecks } from './hooks/useChecks'
export { useEventLog } from './hooks/useEventLog'
export { useFavicon } from './hooks/useFavicon'
export { useKeyboard } from './hooks/useKeyboard'

// ─── Lib (factories & utilities) ───
export {
  createApiClient, api,
  type ApiClient, type ApiClientConfig,
  type CheckDefinition, type CheckImportResult, type CheckImportResultItem,
  type CheckImportError, type CheckImportValidation,
  type AlertEvent as ApiAlertEvent, type AlertsResponse,
  type AlertSilence, type SilencesResponse, type CreateSilenceRequest,
  type AlertChannel, type AlertChannelInput,
  type CheckDefaults, type RegionResult,
} from './lib/api'
export {
  createWebSocket, WebSocketManager,
  type WebSocketConfig,
  type Check, type WSMessage,
  type AlertEvent as WSAlertEvent,
} from './lib/websocket'
export { cn, relativeTime, formatTime } from './lib/utils'
export { ThemeProvider, useTheme } from './lib/theme'
export {
  TopBarConfigProvider, useTopBarConfig,
  type TopBarConfig, type TopBarConfigProviderProps,
} from './lib/topbar-context'
export {
  CHANNEL_TYPES, getChannelMeta,
  type ChannelType, type ChannelTypeMeta,
} from './lib/channels'
