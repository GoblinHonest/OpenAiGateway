import React, { useEffect, useMemo, useState } from 'react'
import { toast } from 'sonner'
import {
  Activity,
  AlertTriangle,
  CheckCircle2,
  Clock,
  RefreshCw,
  Shield,
  XCircle,
  Zap,
  TrendingDown,
  TrendingUp,
  Wifi,
  WifiOff,
} from 'lucide-react'
import { fetchProviderHealth, ProviderHealth } from '../api/health'
import { Button } from '@/components/ui/button'
import { BentoCard } from '@/components/ui/bento-card'
import { PageHeader } from '@/components/ui/page-header'
import { Badge } from '@/components/ui/badge'
import { Skeleton } from '@/components/ui/skeleton'

// ─── Helpers ────────────────────────────────────────────────────────────────

function formatLatency(ms: number): string {
  if (ms >= 1000) return (ms / 1000).toFixed(2) + 's'
  return Math.round(ms) + 'ms'
}

function formatPercent(value: number): string {
  return (value * 100).toFixed(1) + '%'
}

function formatRelativeTime(dateStr: string): string {
  if (!dateStr) return '从未'
  const date = new Date(dateStr)
  const now = new Date()
  const diffMs = now.getTime() - date.getTime()
  const diffSec = Math.floor(diffMs / 1000)
  const diffMin = Math.floor(diffSec / 60)
  const diffHour = Math.floor(diffMin / 60)

  if (diffSec < 60) return `${diffSec}秒前`
  if (diffMin < 60) return `${diffMin}分钟前`
  if (diffHour < 24) return `${diffHour}小时前`
  return date.toLocaleDateString('zh-CN')
}

function getStatusConfig(status: string) {
  switch (status) {
    case 'healthy':
      return {
        label: '健康',
        icon: CheckCircle2,
        badgeClass: 'border-emerald-200 bg-emerald-50 text-emerald-700 dark:border-emerald-800 dark:bg-emerald-950/30 dark:text-emerald-400',
        dotClass: 'bg-emerald-500',
        cardBg: 'bg-emerald-500/8',
        cardColor: 'text-emerald-600 dark:text-emerald-400',
        ringColor: 'ring-emerald-500/20',
      }
    case 'degraded':
      return {
        label: '降级',
        icon: AlertTriangle,
        badgeClass: 'border-amber-200 bg-amber-50 text-amber-700 dark:border-amber-800 dark:bg-amber-950/30 dark:text-amber-400',
        dotClass: 'bg-amber-500',
        cardBg: 'bg-amber-500/8',
        cardColor: 'text-amber-600 dark:text-amber-400',
        ringColor: 'ring-amber-500/20',
      }
    case 'unhealthy':
      return {
        label: '异常',
        icon: XCircle,
        badgeClass: 'border-red-200 bg-red-50 text-red-700 dark:border-red-800 dark:bg-red-950/30 dark:text-red-400',
        dotClass: 'bg-red-500',
        cardBg: 'bg-red-500/8',
        cardColor: 'text-red-600 dark:text-red-400',
        ringColor: 'ring-red-500/20',
      }
    default:
      return {
        label: status,
        icon: Activity,
        badgeClass: 'border-slate-200 bg-slate-50 text-slate-600 dark:border-slate-700 dark:bg-slate-800/50 dark:text-slate-400',
        dotClass: 'bg-slate-400',
        cardBg: 'bg-slate-500/8',
        cardColor: 'text-slate-600 dark:text-slate-400',
        ringColor: 'ring-slate-500/20',
      }
  }
}

function getLatencyColor(ms: number): string {
  if (ms < 200) return 'text-emerald-600 dark:text-emerald-400'
  if (ms < 500) return 'text-amber-600 dark:text-amber-400'
  return 'text-red-600 dark:text-red-400'
}

function getAvailabilityColor(value: number): string {
  if (value >= 0.99) return 'text-emerald-600 dark:text-emerald-400'
  if (value >= 0.95) return 'text-amber-600 dark:text-amber-400'
  return 'text-red-600 dark:text-red-400'
}

// ─── Loading State ──────────────────────────────────────────────────────────

function HealthSkeleton() {
  return (
    <div className="space-y-6">
      <div className="flex items-center justify-between">
        <Skeleton className="h-8 w-40" />
        <Skeleton className="h-9 w-24" />
      </div>
      <div className="grid grid-cols-2 gap-2.5 sm:grid-cols-4">
        {Array.from({ length: 4 }).map((_, i) => (
          <Skeleton key={i} className="h-[68px] rounded-xl" />
        ))}
      </div>
      <div className="grid grid-cols-1 gap-4 md:grid-cols-2 lg:grid-cols-3">
        {Array.from({ length: 3 }).map((_, i) => (
          <Skeleton key={i} className="h-[220px] rounded-2xl" />
        ))}
      </div>
    </div>
  )
}

// ─── Main Component ─────────────────────────────────────────────────────────

export default function Health() {
  const [providers, setProviders] = useState<ProviderHealth[]>([])
  const [loading, setLoading] = useState(true)

  const loadHealth = () => {
    setLoading(true)
    fetchProviderHealth()
      .then(res => setProviders(res.providers || []))
      .catch(err => {
        toast.error('加载健康状态失败', {
          description: err instanceof Error ? err.message : '未知错误',
        })
      })
      .finally(() => setLoading(false))
  }

  useEffect(() => { loadHealth() }, [])

  // ─── Derived Stats ──────────────────────────────────────────────────────

  const stats = useMemo(() => {
    const healthyCount = providers.filter(p => p.status === 'healthy').length
    const degradedCount = providers.filter(p => p.status === 'degraded').length
    const unhealthyCount = providers.filter(p => p.status === 'unhealthy').length
    const avgLatency = providers.length > 0
      ? providers.reduce((sum, p) => sum + p.avg_latency_ms, 0) / providers.length
      : 0
    return { healthyCount, degradedCount, unhealthyCount, avgLatency }
  }, [providers])

  // ─── Metrics Bar ────────────────────────────────────────────────────────

  const metricsBar = [
    {
      label: '服务商总数',
      value: String(providers.length),
      icon: Shield,
      bg: 'bg-blue-500/8',
      color: 'text-blue-600 dark:text-blue-400',
    },
    {
      label: '健康',
      value: String(stats.healthyCount),
      icon: CheckCircle2,
      bg: 'bg-emerald-500/8',
      color: 'text-emerald-600 dark:text-emerald-400',
    },
    {
      label: '降级/异常',
      value: String(stats.degradedCount + stats.unhealthyCount),
      icon: AlertTriangle,
      bg: (stats.degradedCount + stats.unhealthyCount) > 0 ? 'bg-amber-500/8' : 'bg-muted',
      color: (stats.degradedCount + stats.unhealthyCount) > 0 ? 'text-amber-600 dark:text-amber-400' : 'text-muted-foreground',
    },
    {
      label: '平均延迟',
      value: formatLatency(stats.avgLatency),
      icon: Zap,
      bg: 'bg-violet-500/8',
      color: 'text-violet-600 dark:text-violet-400',
    },
  ]

  // ─── Loading ────────────────────────────────────────────────────────────

  if (loading) return <HealthSkeleton />

  // ─── Render ─────────────────────────────────────────────────────────────

  return (
    <div className="space-y-6">
      {/* ─── Header ─────────────────────────────────────────────────────── */}
      <PageHeader title="健康监控" description="服务商健康状态与可用性指标">
        <Button variant="outline" size="sm" onClick={loadHealth}>
          <RefreshCw className="size-3.5" />
          刷新
        </Button>
      </PageHeader>

      {/* ─── Metrics Bar ────────────────────────────────────────────────── */}
      <div className="grid grid-cols-2 gap-2.5 sm:grid-cols-4">
        {metricsBar.map(m => {
          const Icon = m.icon
          return (
            <div
              key={m.label}
              className="group flex items-center gap-2.5 rounded-xl border border-border/60 bg-card p-3 transition-all hover:border-border hover:shadow-sm"
            >
              <div className={`flex h-9 w-9 shrink-0 items-center justify-center rounded-lg ${m.bg}`}>
                <Icon className={`size-4 ${m.color}`} />
              </div>
              <div className="min-w-0">
                <p className="truncate text-[10px] font-medium uppercase tracking-wider text-muted-foreground">
                  {m.label}
                </p>
                <p className="text-sm font-bold tabular-nums text-foreground">
                  {m.value}
                </p>
              </div>
            </div>
          )
        })}
      </div>

      {/* ─── Provider Health Cards ──────────────────────────────────────── */}
      {providers.length === 0 ? (
        <BentoCard>
          <div className="flex flex-col items-center justify-center py-12 text-center">
            <WifiOff className="mb-4 size-12 text-muted-foreground/40" />
            <p className="text-sm font-medium text-muted-foreground">暂无服务商健康数据</p>
            <p className="mt-1 text-xs text-muted-foreground/60">添加服务商后将自动开始健康检查</p>
          </div>
        </BentoCard>
      ) : (
        <div className="grid grid-cols-1 gap-4 md:grid-cols-2 lg:grid-cols-3">
          {providers.map((provider, index) => {
            const statusConf = getStatusConfig(provider.status)
            const StatusIcon = statusConf.icon

            return (
              <BentoCard key={index} className="relative">
                {/* Status indicator dot */}
                <div className="absolute right-4 top-4">
                  <span className={`inline-block h-2.5 w-2.5 rounded-full ${statusConf.dotClass} ring-4 ${statusConf.ringColor}`} />
                </div>

                {/* Card Header */}
                <div className="mb-4 flex items-center gap-3">
                  <div className={`flex h-10 w-10 items-center justify-center rounded-xl ${statusConf.cardBg}`}>
                    <StatusIcon className={`size-5 ${statusConf.cardColor}`} />
                  </div>
                  <div>
                    <p className="text-sm font-semibold">服务商 #{index + 1}</p>
                    <Badge className={`mt-1 text-[10px] ${statusConf.badgeClass}`}>
                      {statusConf.label}
                    </Badge>
                  </div>
                </div>

                {/* Metrics Grid */}
                <div className="grid grid-cols-2 gap-3">
                  {/* Latency */}
                  <div className="rounded-lg bg-muted/40 p-2.5">
                    <div className="flex items-center gap-1.5 text-[10px] font-medium uppercase tracking-wider text-muted-foreground">
                      <Clock className="size-3" />
                      平均延迟
                    </div>
                    <p className={`mt-1 text-lg font-bold tabular-nums ${getLatencyColor(provider.avg_latency_ms)}`}>
                      {formatLatency(provider.avg_latency_ms)}
                    </p>
                  </div>

                  {/* Availability */}
                  <div className="rounded-lg bg-muted/40 p-2.5">
                    <div className="flex items-center gap-1.5 text-[10px] font-medium uppercase tracking-wider text-muted-foreground">
                      <Wifi className="size-3" />
                      可用性
                    </div>
                    <p className={`mt-1 text-lg font-bold tabular-nums ${getAvailabilityColor(provider.availability)}`}>
                      {formatPercent(provider.availability)}
                    </p>
                  </div>

                  {/* Error Rate */}
                  <div className="rounded-lg bg-muted/40 p-2.5">
                    <div className="flex items-center gap-1.5 text-[10px] font-medium uppercase tracking-wider text-muted-foreground">
                      <TrendingDown className="size-3" />
                      错误率
                    </div>
                    <p className={`mt-1 text-lg font-bold tabular-nums ${provider.error_rate > 0.05 ? 'text-red-600 dark:text-red-400' : 'text-foreground'}`}>
                      {formatPercent(provider.error_rate)}
                    </p>
                  </div>

                  {/* Consecutive */}
                  <div className="rounded-lg bg-muted/40 p-2.5">
                    <div className="flex items-center gap-1.5 text-[10px] font-medium uppercase tracking-wider text-muted-foreground">
                      <TrendingUp className="size-3" />
                      连续通过
                    </div>
                    <p className="mt-1 text-lg font-bold tabular-nums text-foreground">
                      {provider.consecutive_passes}
                    </p>
                  </div>
                </div>

                {/* Footer */}
                <div className="mt-3 flex items-center justify-between border-t border-border/60 pt-3">
                  <span className="text-[10px] text-muted-foreground">
                    连续失败: <span className="font-semibold tabular-nums">{provider.consecutive_failures}</span>
                  </span>
                  <span className="text-[10px] text-muted-foreground">
                    最后检查: {formatRelativeTime(provider.last_check_at)}
                  </span>
                </div>
              </BentoCard>
            )
          })}
        </div>
      )}
    </div>
  )
}
