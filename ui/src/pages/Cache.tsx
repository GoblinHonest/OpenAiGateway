import React, { useEffect, useState } from 'react'
import { toast } from 'sonner'
import {
  Database,
  RefreshCw,
  Trash2,
  Settings,
  HardDrive,
  Layers,
  Info,
  Zap,
  CheckCircle2,
  XCircle,
} from 'lucide-react'
import {
  fetchCacheConfig,
  fetchCacheStats,
  fetchCacheEntries,
  clearCache,
  type CacheConfig,
  type CacheStats,
} from '../api/cache'
import { Button } from '@/components/ui/button'
import { BentoCard } from '@/components/ui/bento-card'
import { PageHeader } from '@/components/ui/page-header'
import { Badge } from '@/components/ui/badge'
import { Skeleton } from '@/components/ui/skeleton'
import {
  AlertDialog,
  AlertDialogAction,
  AlertDialogCancel,
  AlertDialogContent,
  AlertDialogDescription,
  AlertDialogFooter,
  AlertDialogHeader,
  AlertDialogTitle,
} from '@/components/ui/alert-dialog'

// ─── Helpers ────────────────────────────────────────────────────────────────

function formatNumber(n: number): string {
  if (n >= 1_000_000) return (n / 1_000_000).toFixed(1) + 'M'
  if (n >= 1_000) return (n / 1_000).toFixed(1) + 'K'
  return n.toLocaleString()
}

// ─── Loading State ──────────────────────────────────────────────────────────

function CacheSkeleton() {
  return (
    <div className="space-y-6">
      <div className="flex items-center justify-between">
        <Skeleton className="h-8 w-32" />
        <div className="flex gap-2">
          <Skeleton className="h-9 w-20" />
          <Skeleton className="h-9 w-24" />
        </div>
      </div>
      <div className="grid grid-cols-2 gap-2.5 sm:grid-cols-3">
        {Array.from({ length: 3 }).map((_, i) => (
          <Skeleton key={i} className="h-[68px] rounded-xl" />
        ))}
      </div>
      <div className="grid grid-cols-1 gap-4 md:grid-cols-2">
        {Array.from({ length: 2 }).map((_, i) => (
          <Skeleton key={i} className="h-[180px] rounded-2xl" />
        ))}
      </div>
    </div>
  )
}

// ─── Main Component ─────────────────────────────────────────────────────────

export default function Cache() {
  const [config, setConfig] = useState<CacheConfig | null>(null)
  const [stats, setStats] = useState<CacheStats | null>(null)
  const [entryCount, setEntryCount] = useState(0)
  const [loading, setLoading] = useState(true)
  const [clearing, setClearing] = useState(false)
  const [showClearDialog, setShowClearDialog] = useState(false)

  const loadData = () => {
    setLoading(true)
    Promise.all([fetchCacheConfig(), fetchCacheStats(), fetchCacheEntries()])
      .then(([cfg, st, entries]) => {
        setConfig(cfg)
        setStats(st)
        setEntryCount(entries.total)
      })
      .catch(err => {
        toast.error('加载缓存数据失败', {
          description: err instanceof Error ? err.message : '未知错误',
        })
      })
      .finally(() => setLoading(false))
  }

  useEffect(() => { loadData() }, [])

  const handleClear = async () => {
    setClearing(true)
    try {
      const result = await clearCache()
      toast.success('缓存已清除', {
        description: result.message,
      })
      loadData()
    } catch (err) {
      toast.error('清除缓存失败', {
        description: err instanceof Error ? err.message : '未知错误',
      })
    } finally {
      setClearing(false)
      setShowClearDialog(false)
    }
  }

  // ─── Metrics Bar ────────────────────────────────────────────────────────

  const metricsBar = [
    {
      label: '缓存状态',
      value: config?.enabled ? '已启用' : '未启用',
      icon: config?.enabled ? CheckCircle2 : XCircle,
      bg: config?.enabled ? 'bg-emerald-500/8' : 'bg-muted',
      color: config?.enabled ? 'text-emerald-600 dark:text-emerald-400' : 'text-muted-foreground',
    },
    {
      label: '缓存条目',
      value: formatNumber(stats?.total_entries ?? 0),
      icon: Layers,
      bg: 'bg-blue-500/8',
      color: 'text-blue-600 dark:text-blue-400',
    },
    {
      label: '缓存类型',
      value: config?.type === 'llm_response_cache' ? 'LLM 响应' : (config?.type || '-'),
      icon: Database,
      bg: 'bg-violet-500/8',
      color: 'text-violet-600 dark:text-violet-400',
    },
  ]

  // ─── Loading ────────────────────────────────────────────────────────────

  if (loading) return <CacheSkeleton />

  // ─── Render ─────────────────────────────────────────────────────────────

  return (
    <div className="space-y-6">
      {/* ─── Header ─────────────────────────────────────────────────────── */}
      <PageHeader title="缓存管理" description="管理 LLM 响应缓存配置和数据">
        <Button variant="outline" size="sm" onClick={loadData}>
          <RefreshCw className="size-3.5" />
          刷新
        </Button>
        <Button
          variant="destructive"
          size="sm"
          onClick={() => setShowClearDialog(true)}
          disabled={!config?.enabled || clearing}
        >
          <Trash2 className="size-3.5" />
          清除缓存
        </Button>
      </PageHeader>

      {/* ─── Metrics Bar ────────────────────────────────────────────────── */}
      <div className="grid grid-cols-2 gap-2.5 sm:grid-cols-3">
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
                <p className="text-sm font-bold text-foreground">
                  {m.value}
                </p>
              </div>
            </div>
          )
        })}
      </div>

      {/* ─── Config & Info Cards ────────────────────────────────────────── */}
      <div className="grid grid-cols-1 gap-4 md:grid-cols-2">
        {/* Cache Configuration */}
        <BentoCard>
          <div className="mb-4 flex items-center gap-3 border-b border-border/60 pb-4">
            <div className="flex h-9 w-9 items-center justify-center rounded-xl bg-cyan-500/8">
              <Settings className="size-4.5 text-cyan-600 dark:text-cyan-400" />
            </div>
            <div>
              <h3 className="text-sm font-semibold">缓存配置</h3>
              <p className="text-xs text-muted-foreground">当前缓存运行参数</p>
            </div>
          </div>

          <div className="space-y-0">
            <div className="flex items-center justify-between rounded-lg px-2 py-3">
              <span className="text-sm text-muted-foreground">启用状态</span>
              <Badge className={config?.enabled
                ? 'border-emerald-200 bg-emerald-50 text-emerald-700 dark:border-emerald-800 dark:bg-emerald-950/30 dark:text-emerald-400'
                : 'border-slate-200 bg-slate-50 text-slate-600 dark:border-slate-700 dark:bg-slate-800/50 dark:text-slate-400'
              }>
                {config?.enabled ? '已启用' : '未启用'}
              </Badge>
            </div>
            <div className="flex items-center justify-between rounded-lg px-2 py-3">
              <span className="text-sm text-muted-foreground">缓存类型</span>
              <span className="font-mono text-xs text-foreground">{config?.type || '-'}</span>
            </div>
            <div className="flex items-center justify-between rounded-lg px-2 py-3">
              <span className="text-sm text-muted-foreground">条目数量</span>
              <span className="text-sm font-semibold tabular-nums text-foreground">
                {formatNumber(stats?.total_entries ?? 0)}
              </span>
            </div>
          </div>
        </BentoCard>

        {/* Cache Description / Info */}
        <BentoCard>
          <div className="mb-4 flex items-center gap-3 border-b border-border/60 pb-4">
            <div className="flex h-9 w-9 items-center justify-center rounded-xl bg-blue-500/8">
              <Info className="size-4.5 text-blue-600 dark:text-blue-400" />
            </div>
            <div>
              <h3 className="text-sm font-semibold">说明</h3>
              <p className="text-xs text-muted-foreground">缓存工作原理</p>
            </div>
          </div>

          <div className="space-y-4">
            <p className="text-sm leading-relaxed text-muted-foreground">
              {config?.description || '网关侧 LLM 响应缓存，相同请求命中时直接返回缓存结果'}
            </p>

            <div className="rounded-lg border border-border/60 bg-muted/30 p-3">
              <div className="flex items-start gap-2">
                <Zap className="mt-0.5 size-4 shrink-0 text-amber-500" />
                <div>
                  <p className="text-xs font-semibold">工作方式</p>
                  <p className="mt-1 text-xs leading-relaxed text-muted-foreground">
                    相同的请求参数（模型、消息、温度等）会生成缓存键。命中缓存时直接返回结果，跳过服务商调用，降低延迟和成本。
                  </p>
                </div>
              </div>
            </div>

            <div className="rounded-lg border border-border/60 bg-muted/30 p-3">
              <div className="flex items-start gap-2">
                <HardDrive className="mt-0.5 size-4 shrink-0 text-violet-500" />
                <div>
                  <p className="text-xs font-semibold">存储后端</p>
                  <p className="mt-1 text-xs leading-relaxed text-muted-foreground">
                    缓存数据存储在 Redis 中。清除操作将删除所有缓存条目，不影响原始配置。
                  </p>
                </div>
              </div>
            </div>
          </div>
        </BentoCard>
      </div>

      {/* ─── Prompt Caching Info (from Config page) ─────────────────────── */}
      <div>
        <p className="mb-3 text-xs font-semibold uppercase tracking-wider text-muted-foreground">
          服务商侧 Prompt 缓存
        </p>
        <BentoCard compact>
          <div className="flex items-start gap-3">
            <div className="flex h-9 w-9 shrink-0 items-center justify-center rounded-xl bg-emerald-500/8">
              <Database className="size-4.5 text-emerald-600 dark:text-emerald-400" />
            </div>
            <div className="space-y-2">
              <h3 className="text-sm font-semibold">各服务商 Prompt 缓存策略</h3>
              <div className="space-y-1.5 text-xs text-muted-foreground">
                <p>
                  <span className="font-semibold text-foreground">OpenAI</span> — 自动缓存超过 1024 tokens 的 prompt
                </p>
                <p>
                  <span className="font-semibold text-foreground">Anthropic</span> — 需在请求中指定 cache_control 参数
                </p>
                <p>
                  <span className="font-semibold text-foreground">DeepSeek</span> — 自动缓存
                </p>
              </div>
            </div>
          </div>
        </BentoCard>
      </div>

      {/* ─── Clear Confirmation Dialog ──────────────────────────────────── */}
      <AlertDialog open={showClearDialog} onOpenChange={setShowClearDialog}>
        <AlertDialogContent>
          <AlertDialogHeader>
            <AlertDialogTitle>确认清除缓存？</AlertDialogTitle>
            <AlertDialogDescription>
              此操作将删除所有 {formatNumber(stats?.total_entries ?? 0)} 条缓存条目。清除后，后续请求将重新调用服务商。
            </AlertDialogDescription>
          </AlertDialogHeader>
          <AlertDialogFooter>
            <AlertDialogCancel disabled={clearing}>取消</AlertDialogCancel>
            <AlertDialogAction
              onClick={handleClear}
              disabled={clearing}
              className="bg-destructive text-destructive-foreground hover:bg-destructive/90"
            >
              {clearing ? '清除中...' : '确认清除'}
            </AlertDialogAction>
          </AlertDialogFooter>
        </AlertDialogContent>
      </AlertDialog>
    </div>
  )
}
