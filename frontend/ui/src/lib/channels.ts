import {
  MessageSquare,
  Hash,
  Mail,
  Megaphone,
  Users,
  AlertTriangle,
  Eye,
  Bell,
  Globe,
} from 'lucide-react'

export interface ChannelTypeMeta {
  value: string
  label: string
  icon: typeof MessageSquare
  color: string
}

export const CHANNEL_TYPES: readonly ChannelTypeMeta[] = [
  { value: 'telegram', label: 'Telegram', icon: MessageSquare, color: 'bg-blue-500' },
  { value: 'slack', label: 'Slack App', icon: Hash, color: 'bg-purple-500' },
  { value: 'slack_webhook', label: 'Slack Webhook', icon: Hash, color: 'bg-purple-400' },
  { value: 'email', label: 'Email', icon: Mail, color: 'bg-green-500' },
  { value: 'discord', label: 'Discord', icon: Megaphone, color: 'bg-indigo-500' },
  { value: 'teams', label: 'Teams', icon: Users, color: 'bg-blue-600' },
  { value: 'pagerduty', label: 'PagerDuty', icon: AlertTriangle, color: 'bg-emerald-500' },
  { value: 'opsgenie', label: 'Opsgenie', icon: Eye, color: 'bg-cyan-500' },
  { value: 'ntfy', label: 'ntfy', icon: Bell, color: 'bg-amber-500' },
  { value: 'webhook', label: 'Webhook', icon: Globe, color: 'bg-orange-500' },
] as const

export type ChannelType = string

export function getChannelMeta(type: string, extraTypes?: ChannelTypeMeta[]): ChannelTypeMeta {
  const allTypes = extraTypes ? [...CHANNEL_TYPES, ...extraTypes] : CHANNEL_TYPES
  return allTypes.find((ct) => ct.value === type) || CHANNEL_TYPES[0]
}
