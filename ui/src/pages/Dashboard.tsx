import React, { useEffect, useMemo, useState } from 'react'
import { toast } from 'sonner'
import {
  Activity,
  AlertCircle,
  ArrowRight,
  ArrowUpRight,
  ArrowDownRight,
  Building2,
  CheckCircle2,
  Clock,
  Cpu,
  Hash,
  Layers,
  Minus,
  RefreshCw,
  TrendingUp,
  Zap,
} from 'lucide-react'
import {
  LineChart,
  Line,
  BarChart,
  Bar,
  PieChart,
  Pie,
  Cell,
  XAxis,
  YAxis,
  CartesianGrid,
  Tooltip as ReTooltip,
  ResponsiveContainer,
  Legend,
  Area,
  AreaChart,
} from 'recharts'
import { fetchDashboard, DashboardOverview } from '../api/dashboard'
import { Button } from '@/components/ui/button'
import { BentoCard } from '@/components/ui/bento-card'
import { Skeleton } from '@/components/ui/skeleton'
import {
  Table,
  TableBody,
  TableCell,
  TableHead,
  TableHeader,
  TableRow,
} from '@/components/ui/table'
import { Badge } from '@/components/ui/badge'

// ─── Types ──────────────────────────────────────────────────────────────────

type Period = 'today' | '7d' | '30d'

// ─── Constants ──────────────────────────────────────────────────────────────

const CHART_COLORS = [
  '#4361ee',
  '#10b981',
  '#f59e0b',
  '#8b5cf6',
  '#06b6d4',
  '#ec4899',
  '#6366f1',
  '#f97316',
]

const PERIOD_OPTIONS: { label: string; value: Period }[] = [
  { label: '今天', value: 'today' },
  { label: '近7天', value: '7d' },
  { label: '近30天', value: '30d' },
]

// ─── Helpers ────────────────────────────────────────────────────────────────

function formatNumber(n: number): string {
  if (n >= 1_000_000) return (n / 1_000_000).toFixed(1) + 'M'
  if (n >= 1_000) return (n / 1_000).toFixed(1) + 'K'
  return n.toLocaleString()
}

function formatTokenCount(n: number): string {
  if (n >= 1_000_000) return (n / 1_000_000).toFixed(1) + 'M'
  if (n >= 1_000) return (n / 1_000).toFixed(1) + 'K'
  return String(n)
}

function formatMs(ms: number): string {
  if (ms >= 1000) return (ms / 1000).toFixed(1) + 's'
  return Math.round(ms) + 'ms'
}

function formatTime(ts: string): string {
  try {
    const d = new Date(ts)
    return d.toLocaleTimeString('zh-CN', { hour: '2-digit', minute: '2-digit' })
  } catch {
    return ts
  }
}

// ─── Sub-components ─────────────────────────────────────────────────────────

function MiniSparkline({ data, color }: { data: number[]; color: string }) {
  if (!data || data.length < 2) return null
  return (
    <ResponsiveContainer width="100%" height={32}>
      <AreaChart data={data.map((v, i) => ({ i, v }))} margin={{ top: 2, right: 0, left: 0, bottom: 0 }}>
        <defs>
          <linearGradient id={`spark-${color.replace('#', '')}`} x1="0" y1="0" x2="0" y2="1">
            <stop offset="0%" stopColor={color} stopOpacity={0.3} />
            <stop offset="100%" stopColor={color} stopOpacity={0.02} />
          </linearGradient>
        </defs>
        <Area
          type="monotone"
          dataKey="v"
          stroke={color}
          strokeWidth={1.5}
          fill={`url(#spark-${color.replace('#', '')})`}
          dot={false}
          isAnimationActive={false}
        />
      </AreaChart>
    </ResponsiveContainer>
  )
}

// ─── Custom Tooltip ─────────────────────────────────────────────────────────

function ChartTooltip({ active, payload, label }: any) {
  if (!active || !payload?.length) return null
  return (
    <div className="rounded-lg border border-border bg-card px-3 py-2 shadow-lg">
      <p className="mb-1 text-xs font-medium text-muted-foreground">{label}</p>
      {payload.map((entry: any, i: number) => (
        <p key={i} className="text-xs" style={{ color: entry.color }}>
          <span className="font-semibold">{entry.name}:</span>{' '}
          {typeof entry.value === 'number' ? formatNumber(entry.value) : entry.value}
        </p>
      ))}
    </div>
  )
}

// ─── Loading State ──────────────────────────────────────────────────────────

function DashboardSkeleton() {
  return (
    <div className="space-y-6">
      {/* Header skeleton */}
      <div className="flex items-center justify-between">
        <Skeleton className="h-8 w-32" />
        <Skeleton className="h-9 w-24" />
      </div>
      {/* Status + Period */}
      <div className="flex items-center justify-between">
        <Skeleton className="h-8 w-28 rounded-full" />
        <Skeleton className="h-9 w-48 rounded-lg" />
      </div>
      {/* Metrics bar */}
      <div className="grid grid-cols-2 gap-3 sm:grid-cols-4 lg:grid-cols-8">
        {Array.from({ length: 8 }).map((_, i) => (
          <Skeleton key={i} className="h-[72px] rounded-xl" />
        ))}
      </div>
      {/* KPI cards */}
      <div className="grid grid-cols-1 gap-4 sm:grid-cols-2 lg:grid-cols-4">
        {Array.from({ length: 4 }).map((_, i) => (
          <Skeleton key={i} className="h-[140px] rounded-2xl" />
        ))}
      </div>
      {/* Charts */}
      <div className="grid grid-cols-1 gap-4 lg:grid-cols-2">
        {Array.from({ length: 4 }).map((_, i) => (
          <Skeleton key={i} className="h-[280px] rounded-2xl" />
        ))}
      </div>
      {/* Tables */}
      <div className="grid grid-cols-1 gap-4 lg:grid-cols-2">
        <Skeleton className="h-[320px] rounded-2xl" />
        <Skeleton className="h-[320px] rounded-2xl" />
      </div>
    </div>
  )
}

// ─── Main Dashboard ─────────────────────────────────────────────────────────

export default function Dashboard() {
  const [data, setData] = useState<DashboardOverview | null>(null)
  const [loading, setLoading] = useState(true)
  const [period, setPeriod] = useState<Period>('7d')

  const loadData = () => {
    setLoading(true)
    fetchDashboard()
      .then(setData)
      .catch(err => {
        toast.error('加载仪表盘数据失败', { description: err instanceof Error ? err.message : '未知错误' })
      })
      .finally(() => setLoading(false))
  }

  useEffect(() => { loadData() }, [])

  // Derived data
  const totalTokens = useMemo(() => {
    if (!data) return 0
    return data.overview.todayInputTokens + data.overview.todayOutputTokens
  }, [data])

  const successRate = useMemo(() => {
    if (!data || data.overview.todayRequests === 0) return 100
    return ((data.overview.todayRequests - data.overview.todayFailedRequests) / data.overview.todayRequests) * 100
  }, [data])

  const requestTrendData = useMemo(() => {
    if (!data?.trend) return []
    return data.trend.map(t => ({
      time: t.time.slice(5),
      requests: t.requests,
    }))
  }, [data])

  const tokenTrendData = useMemo(() => {
    if (!data?.trend) return []
    return data.trend.map(t => ({
      time: t.time.slice(5),
      tokens: t.tokens,
    }))
  }, [data])

  const providerPieData = useMemo(() => {
    if (!data?.providerDistribution) return []
    return data.providerDistribution.map((p, i) => ({
      name: p.name,
      value: p.tokens,
      fill: CHART_COLORS[i % CHART_COLORS.length],
    }))
  }, [data])

  const modelRankData = useMemo(() => {
    if (!data?.modelDistribution) return []
    return [...data.modelDistribution]
      .sort((a, b) => b.tokens - a.tokens)
      .slice(0, 8)
      .map((m, i) => ({ rank: i + 1, ...m }))
  }, [data])

  // Period display value
  const periodRequests = useMemo(() => {
    if (!data) return 0
    switch (period) {
      case 'today': return data.overview.todayRequests
      case '7d': return data.overview.sevenDayRequests
      case '30d': return data.overview.thirtyDayRequests
    }
  }, [data, period])

  if (loading) return <DashboardSkeleton />
  if (!data) return <div className="p-6 text-muted-foreground">加载失败</div>

  // ─── Metrics bar items ──────────────────────────────────────────────────

  const metricsBar = [
    {
      label: '今日请求',
      value: formatNumber(data.overview.todayRequests),
      icon: Activity,
      bg: 'bg-emerald-500/8',
      color: 'text-emerald-600 dark:text-emerald-400',
    },
    {
      label: '输入Token',
      value: formatTokenCount(data.overview.todayInputTokens),
      icon: ArrowDownRight,
      bg: 'bg-blue-500/8',
      color: 'text-blue-600 dark:text-blue-400',
    },
    {
      label: '输出Token',
      value: formatTokenCount(data.overview.todayOutputTokens),
      icon: ArrowUpRight,
      bg: 'bg-violet-500/8',
      color: 'text-violet-600 dark:text-violet-400',
    },
    {
      label: '总Token',
      value: formatTokenCount(totalTokens),
      icon: Zap,
      bg: 'bg-amber-500/8',
      color: 'text-amber-600 dark:text-amber-400',
    },
    {
      label: '成功率',
      value: successRate.toFixed(1) + '%',
      icon: CheckCircle2,
      bg: successRate >= 99 ? 'bg-emerald-500/8' : 'bg-amber-500/8',
      color: successRate >= 99 ? 'text-emerald-600 dark:text-emerald-400' : 'text-amber-600 dark:text-amber-400',
    },
    {
      label: '失败请求',
      value: formatNumber(data.overview.todayFailedRequests),
      icon: AlertCircle,
      bg: data.overview.todayFailedRequests > 0 ? 'bg-red-500/8' : 'bg-muted',
      color: data.overview.todayFailedRequests > 0 ? 'text-red-600 dark:text-red-400' : 'text-muted-foreground',
    },
    {
      label: '7日请求',
      value: formatNumber(data.overview.sevenDayRequests),
      icon: TrendingUp,
      bg: 'bg-cyan-500/8',
      color: 'text-cyan-600 dark:text-cyan-400',
    },
    {
      label: '30日请求',
      value: formatNumber(data.overview.thirtyDayRequests),
      icon: Layers,
      bg: 'bg-indigo-500/8',
      color: 'text-indigo-600 dark:text-indigo-400',
    },
  ]

  // ─── KPI cards ──────────────────────────────────────────────────────────

  const kpiCards = [
    {
      label: '请求数',
      value: formatNumber(periodRequests),
      sub: `今日 ${formatNumber(data.overview.todayRequests)}`,
      icon: Hash,
      accent: '#4361ee',
      sparkData: data.trend?.map(t => t.requests) ?? [],
    },
    {
      label: 'Token 消耗',
      value: formatTokenCount(totalTokens),
      sub: `输入 ${formatTokenCount(data.overview.todayInputTokens)} / 输出 ${formatTokenCount(data.overview.todayOutputTokens)}`,
      icon: Zap,
      accent: '#10b981',
      sparkData: data.trend?.map(t => t.tokens) ?? [],
    },
    {
      label: '请求成功率',
      value: successRate.toFixed(1) + '%',
      sub: `失败 ${data.overview.todayFailedRequests} 次`,
      icon: CheckCircle2,
      accent: '#06b6d4',
      sparkData: [],
    },
    {
      label: '失败率',
      value: (data.overview.failureRate * 100).toFixed(2) + '%',
      sub: `共 ${data.overview.todayFailedRequests} 次失败`,
      icon: AlertCircle,
      accent: data.overview.failureRate > 0.05 ? '#ef4444' : '#10b981',
      sparkData: [],
    },
  ]

  return (
    <div className="space-y-6">
      {/* ─── Header ─────────────────────────────────────────────────────── */}
      <div className="flex flex-col gap-4 sm:flex-row sm:items-center sm:justify-between">
        <h1 className="text-2xl font-semibold tracking-tight">仪表盘</h1>
        <div className="flex items-center gap-3">
          {/* System status */}
          <div className="flex items-center gap-2 rounded-full border border-emerald-200 bg-emerald-50 px-3 py-1.5 dark:border-emerald-800 dark:bg-emerald-950/30">
            <span className="relative flex h-2 w-2">
              <span className="absolute inline-flex h-full w-full animate-ping rounded-full bg-emerald-400 opacity-75" />
              <span className="relative inline-flex h-2 w-2 rounded-full bg-emerald-500" />
            </span>
            <span className="text-xs font-medium text-emerald-700 dark:text-emerald-300">系统正常</span>
          </div>

          {/* Period switcher */}
          <div className="flex items-center rounded-lg border border-border bg-muted/50 p-0.5">
            {PERIOD_OPTIONS.map(opt => (
              <button
                key={opt.value}
                onClick={() => setPeriod(opt.value)}
                className={`rounded-md px-3 py-1.5 text-xs font-medium transition-all ${
                  period === opt.value
                    ? 'bg-card text-foreground shadow-sm'
                    : 'text-muted-foreground hover:text-foreground'
                }`}
              >
                {opt.label}
              </button>
            ))}
          </div>

          <Button variant="outline" size="sm" onClick={loadData}>
            <RefreshCw className="size-3.5" />
            刷新
          </Button>
        </div>
      </div>

      {/* ─── Realtime Metrics Bar ───────────────────────────────────────── */}
      <div className="grid grid-cols-2 gap-2.5 sm:grid-cols-4 lg:grid-cols-8">
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

      {/* ─── KPI Cards ──────────────────────────────────────────────────── */}
      <div>
        <p className="mb-3 text-xs font-semibold uppercase tracking-wider text-muted-foreground">
          核心指标
        </p>
        <div className="grid grid-cols-1 gap-4 sm:grid-cols-2 lg:grid-cols-4">
          {kpiCards.map(card => {
            const Icon = card.icon
            return (
              <BentoCard key={card.label} className="relative overflow-hidden">
                {/* Accent deco */}
                <div
                  className="pointer-events-none absolute -right-8 -top-8 h-24 w-24 rounded-full opacity-[0.07] blur-2xl"
                  style={{ background: card.accent }}
                />
                <div className="relative z-10">
                  <div className="flex items-center justify-between">
                    <div
                      className="flex h-9 w-9 items-center justify-center rounded-xl"
                      style={{ background: `${card.accent}10` }}
                    >
                      <Icon className="size-4.5" style={{ color: card.accent }} />
                    </div>
                  </div>
                  <p className="mt-3 text-xs font-medium text-muted-foreground">{card.label}</p>
                  <p className="mt-1 text-2xl font-bold tracking-tight">{card.value}</p>
                  <p className="mt-0.5 text-[11px] text-muted-foreground">{card.sub}</p>
                  {/* Sparkline */}
                  {card.sparkData.length > 1 && (
                    <div className="mt-2 -mx-1">
                      <MiniSparkline data={card.sparkData} color={card.accent} />
                    </div>
                  )}
                </div>
              </BentoCard>
            )
          })}
        </div>
      </div>

      {/* ─── Charts 2x2 Grid ────────────────────────────────────────────── */}
      <div>
        <p className="mb-3 text-xs font-semibold uppercase tracking-wider text-muted-foreground">
          趋势分析
        </p>
        <div className="grid grid-cols-1 gap-4 lg:grid-cols-2">
          {/* Request Trend */}
          <BentoCard compact>
            <div className="mb-3 flex items-center gap-2 border-b border-border/60 pb-3">
              <TrendingUp className="size-4 text-muted-foreground" />
              <span className="text-sm font-semibold">请求量趋势</span>
            </div>
            <div className="h-[220px]">
              {requestTrendData.length > 0 ? (
                <ResponsiveContainer width="100%" height="100%">
                  <AreaChart data={requestTrendData} margin={{ top: 5, right: 10, left: -10, bottom: 0 }}>
                    <defs>
                      <linearGradient id="reqGrad" x1="0" y1="0" x2="0" y2="1">
                        <stop offset="0%" stopColor="#4361ee" stopOpacity={0.2} />
                        <stop offset="100%" stopColor="#4361ee" stopOpacity={0.01} />
                      </linearGradient>
                    </defs>
                    <CartesianGrid strokeDasharray="3 3" stroke="hsl(var(--border))" vertical={false} />
                    <XAxis
                      dataKey="time"
                      tick={{ fontSize: 11, fill: 'hsl(var(--muted-foreground))' }}
                      axisLine={{ stroke: 'hsl(var(--border))' }}
                      tickLine={false}
                    />
                    <YAxis
                      tick={{ fontSize: 11, fill: 'hsl(var(--muted-foreground))' }}
                      axisLine={false}
                      tickLine={false}
                    />
                    <ReTooltip content={<ChartTooltip />} />
                    <Area
                      type="monotone"
                      dataKey="requests"
                      name="请求数"
                      stroke="#4361ee"
                      strokeWidth={2}
                      fill="url(#reqGrad)"
                      dot={{ r: 3, fill: '#4361ee', strokeWidth: 0 }}
                      activeDot={{ r: 5, fill: '#4361ee', strokeWidth: 2, stroke: '#fff' }}
                    />
                  </AreaChart>
                </ResponsiveContainer>
              ) : (
                <div className="flex h-full items-center justify-center text-sm text-muted-foreground">暂无数据</div>
              )}
            </div>
          </BentoCard>

          {/* Token Trend */}
          <BentoCard compact>
            <div className="mb-3 flex items-center gap-2 border-b border-border/60 pb-3">
              <Zap className="size-4 text-muted-foreground" />
              <span className="text-sm font-semibold">Token 趋势</span>
            </div>
            <div className="h-[220px]">
              {tokenTrendData.length > 0 ? (
                <ResponsiveContainer width="100%" height="100%">
                  <BarChart data={tokenTrendData} margin={{ top: 5, right: 10, left: -10, bottom: 0 }}>
                    <CartesianGrid strokeDasharray="3 3" stroke="hsl(var(--border))" vertical={false} />
                    <XAxis
                      dataKey="time"
                      tick={{ fontSize: 11, fill: 'hsl(var(--muted-foreground))' }}
                      axisLine={{ stroke: 'hsl(var(--border))' }}
                      tickLine={false}
                    />
                    <YAxis
                      tick={{ fontSize: 11, fill: 'hsl(var(--muted-foreground))' }}
                      axisLine={false}
                      tickLine={false}
                    />
                    <ReTooltip content={<ChartTooltip />} />
                    <Bar
                      dataKey="tokens"
                      name="Token"
                      fill="#10b981"
                      radius={[4, 4, 0, 0]}
                      maxBarSize={32}
                    />
                  </BarChart>
                </ResponsiveContainer>
              ) : (
                <div className="flex h-full items-center justify-center text-sm text-muted-foreground">暂无数据</div>
              )}
            </div>
          </BentoCard>

          {/* Provider Distribution */}
          <BentoCard compact>
            <div className="mb-3 flex items-center gap-2 border-b border-border/60 pb-3">
              <Building2 className="size-4 text-muted-foreground" />
              <span className="text-sm font-semibold">服务商分布</span>
            </div>
            <div className="h-[220px]">
              {providerPieData.length > 0 ? (
                <ResponsiveContainer width="100%" height="100%">
                  <PieChart>
                    <Pie
                      data={providerPieData}
                      cx="40%"
                      cy="50%"
                      innerRadius={50}
                      outerRadius={80}
                      paddingAngle={3}
                      dataKey="value"
                      strokeWidth={0}
                    >
                      {providerPieData.map((entry, i) => (
                        <Cell key={i} fill={entry.fill} />
                      ))}
                    </Pie>
                    <ReTooltip
                      formatter={(value: number, name: string) => [formatTokenCount(value), name]}
                    />
                    <Legend
                      verticalAlign="middle"
                      align="right"
                      layout="vertical"
                      iconType="circle"
                      iconSize={8}
                      formatter={(value: string) => (
                        <span className="text-xs text-muted-foreground">{value}</span>
                      )}
                    />
                  </PieChart>
                </ResponsiveContainer>
              ) : (
                <div className="flex h-full items-center justify-center text-sm text-muted-foreground">暂无数据</div>
              )}
            </div>
          </BentoCard>

          {/* Model Distribution */}
          <BentoCard compact>
            <div className="mb-3 flex items-center gap-2 border-b border-border/60 pb-3">
              <Cpu className="size-4 text-muted-foreground" />
              <span className="text-sm font-semibold">模型分布</span>
            </div>
            <div className="h-[220px]">
              {data.modelDistribution && data.modelDistribution.length > 0 ? (
                <ResponsiveContainer width="100%" height="100%">
                  <BarChart
                    data={[...data.modelDistribution]
                      .sort((a, b) => b.tokens - a.tokens)
                      .slice(0, 8)
                      .map(m => ({ name: m.name.length > 12 ? m.name.slice(0, 12) + '…' : m.name, tokens: m.tokens }))}
                    layout="vertical"
                    margin={{ top: 0, right: 10, left: 0, bottom: 0 }}
                  >
                    <CartesianGrid strokeDasharray="3 3" stroke="hsl(var(--border))" horizontal={false} />
                    <XAxis
                      type="number"
                      tick={{ fontSize: 11, fill: 'hsl(var(--muted-foreground))' }}
                      axisLine={false}
                      tickLine={false}
                    />
                    <YAxis
                      type="category"
                      dataKey="name"
                      tick={{ fontSize: 11, fill: 'hsl(var(--muted-foreground))' }}
                      axisLine={false}
                      tickLine={false}
                      width={100}
                    />
                    <ReTooltip content={<ChartTooltip />} />
                    <Bar
                      dataKey="tokens"
                      name="Token"
                      fill="#8b5cf6"
                      radius={[0, 4, 4, 0]}
                      maxBarSize={18}
                    />
                  </BarChart>
                </ResponsiveContainer>
              ) : (
                <div className="flex h-full items-center justify-center text-sm text-muted-foreground">暂无数据</div>
              )}
            </div>
          </BentoCard>
        </div>
      </div>

      {/* ─── Tables Section ─────────────────────────────────────────────── */}
      <div>
        <p className="mb-3 text-xs font-semibold uppercase tracking-wider text-muted-foreground">
          数据明细
        </p>
        <div className="grid grid-cols-1 gap-4 lg:grid-cols-2">
          {/* Model Ranking Table */}
          <BentoCard compact>
            <div className="mb-3 flex items-center gap-2 border-b border-border/60 pb-3">
              <TrendingUp className="size-4 text-muted-foreground" />
              <span className="text-sm font-semibold">模型调用排行</span>
            </div>
            <Table>
              <TableHeader>
                <TableRow className="hover:bg-transparent">
                  <TableHead className="w-12 text-center">#</TableHead>
                  <TableHead>模型名称</TableHead>
                  <TableHead className="text-right">Token 消耗</TableHead>
                </TableRow>
              </TableHeader>
              <TableBody>
                {modelRankData.length > 0 ? (
                  modelRankData.map(m => (
                    <TableRow key={m.name}>
                      <TableCell className="text-center">
                        <span
                          className={`inline-flex h-6 w-6 items-center justify-center rounded-md text-xs font-bold ${
                            m.rank === 1
                              ? 'bg-amber-100 text-amber-700 dark:bg-amber-900/30 dark:text-amber-400'
                              : m.rank === 2
                                ? 'bg-slate-100 text-slate-600 dark:bg-slate-800 dark:text-slate-400'
                                : m.rank === 3
                                  ? 'bg-orange-100 text-orange-700 dark:bg-orange-900/30 dark:text-orange-400'
                                  : 'text-muted-foreground'
                          }`}
                        >
                          {m.rank}
                        </span>
                      </TableCell>
                      <TableCell className="font-medium">{m.name}</TableCell>
                      <TableCell className="text-right tabular-nums">{formatTokenCount(m.tokens)}</TableCell>
                    </TableRow>
                  ))
                ) : (
                  <TableRow>
                    <TableCell colSpan={3} className="py-8 text-center text-sm text-muted-foreground">
                      暂无数据
                    </TableCell>
                  </TableRow>
                )}
              </TableBody>
            </Table>
          </BentoCard>

          {/* Recent Requests Table */}
          <BentoCard compact>
            <div className="mb-3 flex items-center justify-between border-b border-border/60 pb-3">
              <div className="flex items-center gap-2">
                <Clock className="size-4 text-muted-foreground" />
                <span className="text-sm font-semibold">最近请求</span>
              </div>
              <button className="flex items-center gap-1 text-xs font-medium text-primary hover:underline">
                查看全部
                <ArrowRight className="size-3" />
              </button>
            </div>
            <Table>
              <TableHeader>
                <TableRow className="hover:bg-transparent">
                  <TableHead>时间</TableHead>
                  <TableHead>模型</TableHead>
                  <TableHead>服务商</TableHead>
                  <TableHead className="text-right">耗时</TableHead>
                  <TableHead className="text-center">状态</TableHead>
                </TableRow>
              </TableHeader>
              <TableBody>
                {data.recentLogs && data.recentLogs.length > 0 ? (
                  data.recentLogs.slice(0, 8).map(log => (
                    <TableRow key={log.id}>
                      <TableCell className="text-xs text-muted-foreground">
                        {formatTime(log.timestamp)}
                      </TableCell>
                      <TableCell className="max-w-[120px] truncate text-xs font-medium">
                        {log.modelName}
                      </TableCell>
                      <TableCell className="text-xs text-muted-foreground">
                        {log.providerName}
                      </TableCell>
                      <TableCell className="text-right text-xs tabular-nums">
                        <span className={log.totalLatencyMs > 5000 ? 'text-amber-600 dark:text-amber-400' : ''}>
                          {formatMs(log.totalLatencyMs)}
                        </span>
                      </TableCell>
                      <TableCell className="text-center">
                        <Badge
                          variant={log.success ? 'default' : 'destructive'}
                          className={`text-[10px] ${
                            log.success
                              ? 'border-emerald-200 bg-emerald-50 text-emerald-700 dark:border-emerald-800 dark:bg-emerald-950/30 dark:text-emerald-400'
                              : ''
                          }`}
                        >
                          {log.success ? '成功' : '失败'}
                        </Badge>
                      </TableCell>
                    </TableRow>
                  ))
                ) : (
                  <TableRow>
                    <TableCell colSpan={5} className="py-8 text-center text-sm text-muted-foreground">
                      暂无请求记录
                    </TableCell>
                  </TableRow>
                )}
              </TableBody>
            </Table>
          </BentoCard>
        </div>
      </div>
    </div>
  )
}
