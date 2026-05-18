import React, { useEffect, useMemo, useState } from 'react'
import { toast } from 'sonner'
import {
  Activity,
  AlertCircle,
  CheckCircle2,
  Clock,
  FileText,
  Hash,
  RefreshCw,
  Search,
  Timer,
  XCircle,
  Zap,
  ChevronLeft,
  ChevronRight,
  Calendar,
} from 'lucide-react'
import { fetchLogs, LogEntry } from '../api/log'
import { Button } from '@/components/ui/button'
import { BentoCard } from '@/components/ui/bento-card'
import { PageHeader } from '@/components/ui/page-header'
import { Input } from '@/components/ui/input'
import { Badge } from '@/components/ui/badge'
import { Dialog, DialogContent, DialogHeader, DialogTitle } from '@/components/ui/dialog'
import { Skeleton } from '@/components/ui/skeleton'
import {
  Table,
  TableBody,
  TableCell,
  TableHead,
  TableHeader,
  TableRow,
} from '@/components/ui/table'

// ─── Helpers ────────────────────────────────────────────────────────────────

function formatLatency(ms: number): string {
  if (ms >= 1000) return (ms / 1000).toFixed(2) + 's'
  return Math.round(ms) + 'ms'
}

function formatTokenCount(n: number): string {
  if (n >= 1_000_000) return (n / 1_000_000).toFixed(1) + 'M'
  if (n >= 1_000) return (n / 1_000).toFixed(1) + 'K'
  return String(n)
}

// ─── Loading State ──────────────────────────────────────────────────────────

function LogsSkeleton() {
  return (
    <div className="space-y-6">
      <div className="flex items-center justify-between">
        <Skeleton className="h-8 w-32" />
        <Skeleton className="h-9 w-24" />
      </div>
      {/* Metrics bar skeleton */}
      <div className="grid grid-cols-2 gap-2.5 sm:grid-cols-4">
        {Array.from({ length: 4 }).map((_, i) => (
          <Skeleton key={i} className="h-[68px] rounded-xl" />
        ))}
      </div>
      {/* Filter skeleton */}
      <Skeleton className="h-12 rounded-xl" />
      {/* Table skeleton */}
      <Skeleton className="h-96 rounded-2xl" />
    </div>
  )
}

// ─── Main Component ─────────────────────────────────────────────────────────

export default function Logs() {
  const [logs, setLogs] = useState<LogEntry[]>([])
  const [total, setTotal] = useState(0)
  const [loading, setLoading] = useState(true)
  const [search, setSearch] = useState('')
  const [statusFilter, setStatusFilter] = useState<'all' | 'success' | 'failed'>('all')
  const [selectedLog, setSelectedLog] = useState<LogEntry | null>(null)

  // Pagination
  const [page, setPage] = useState(1)
  const [pageSize] = useState(20)

  // Date filtering
  const [startTime, setStartTime] = useState('')
  const [endTime, setEndTime] = useState('')
  const [activeStartTime, setActiveStartTime] = useState('')
  const [activeEndTime, setActiveEndTime] = useState('')

  const totalPages = Math.max(1, Math.ceil(total / pageSize))

  const loadLogs = () => {
    setLoading(true)
    const params: Record<string, string> = { limit: String(pageSize), offset: String((page - 1) * pageSize) }
    if (activeStartTime) params.start_time = activeStartTime
    if (activeEndTime) params.end_time = activeEndTime

    fetchLogs(params)
      .then(res => {
        setLogs(res.items || [])
        setTotal(res.total)
      })
      .catch(err => {
        toast.error('加载日志失败', { description: err instanceof Error ? err.message : '未知错误' })
      })
      .finally(() => setLoading(false))
  }

  useEffect(() => { loadLogs() }, [page, activeStartTime, activeEndTime])

  const handleApplyDateFilter = () => {
    setPage(1)
    setActiveStartTime(startTime)
    setActiveEndTime(endTime)
  }

  const handleClearDateFilter = () => {
    setStartTime('')
    setEndTime('')
    setActiveStartTime('')
    setActiveEndTime('')
    setPage(1)
  }

  // ─── Derived Stats ──────────────────────────────────────────────────────

  const stats = useMemo(() => {
    const successCount = logs.filter(l => l.success).length
    const failedCount = logs.length - successCount
    const avgLatency = logs.length > 0
      ? logs.reduce((sum, l) => sum + l.totalLatencyMs, 0) / logs.length
      : 0
    const totalTokens = logs.reduce((sum, l) => sum + l.inputTokens + l.outputTokens, 0)
    return { successCount, failedCount, avgLatency, totalTokens }
  }, [logs])

  const filteredLogs = useMemo(() => {
    return logs.filter(log => {
      const matchesSearch = search === '' ||
        log.modelName?.toLowerCase().includes(search.toLowerCase()) ||
        log.providerName?.toLowerCase().includes(search.toLowerCase())
      const matchesStatus = statusFilter === 'all' ||
        (statusFilter === 'success' && log.success) ||
        (statusFilter === 'failed' && !log.success)
      return matchesSearch && matchesStatus
    })
  }, [logs, search, statusFilter])

  // ─── Metrics Bar ────────────────────────────────────────────────────────

  const metricsBar = [
    {
      label: '总请求数',
      value: String(total),
      icon: Hash,
      bg: 'bg-blue-500/8',
      color: 'text-blue-600 dark:text-blue-400',
    },
    {
      label: '成功请求',
      value: String(stats.successCount),
      icon: CheckCircle2,
      bg: 'bg-emerald-500/8',
      color: 'text-emerald-600 dark:text-emerald-400',
    },
    {
      label: '失败请求',
      value: String(stats.failedCount),
      icon: XCircle,
      bg: stats.failedCount > 0 ? 'bg-red-500/8' : 'bg-muted',
      color: stats.failedCount > 0 ? 'text-red-600 dark:text-red-400' : 'text-muted-foreground',
    },
    {
      label: '平均延迟',
      value: formatLatency(stats.avgLatency),
      icon: Timer,
      bg: 'bg-amber-500/8',
      color: 'text-amber-600 dark:text-amber-400',
    },
  ]

  // ─── Filter Options ─────────────────────────────────────────────────────

  const filterOptions: { label: string; value: 'all' | 'success' | 'failed' }[] = [
    { label: '全部', value: 'all' },
    { label: '成功', value: 'success' },
    { label: '失败', value: 'failed' },
  ]

  if (loading) return <LogsSkeleton />

  return (
    <div className="space-y-6">
      {/* ─── Header ─────────────────────────────────────────────────────── */}
      <PageHeader title="请求日志" description="查看 API 请求历史记录">
        <Button variant="outline" size="sm" onClick={() => { setPage(1); loadLogs() }}>
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

      {/* ─── Filters ────────────────────────────────────────────────────── */}
      <div className="flex flex-col gap-3 sm:flex-row sm:items-center">
        <div className="relative flex-1">
          <Search className="absolute left-3 top-1/2 size-4 -translate-y-1/2 text-muted-foreground" />
          <Input
            placeholder="搜索模型、服务商..."
            value={search}
            onChange={e => setSearch(e.target.value)}
            className="pl-9"
          />
        </div>
        <div className="flex items-center rounded-lg border border-border bg-muted/50 p-0.5">
          {filterOptions.map(opt => (
            <button
              key={opt.value}
              onClick={() => setStatusFilter(opt.value)}
              className={`rounded-md px-3 py-1.5 text-xs font-medium transition-all ${
                statusFilter === opt.value
                  ? 'bg-card text-foreground shadow-sm'
                  : 'text-muted-foreground hover:text-foreground'
              }`}
            >
              {opt.label}
            </button>
          ))}
        </div>
      </div>

      {/* ─── Date Range Filter ──────────────────────────────────────────── */}
      <div className="flex flex-col gap-2.5 sm:flex-row sm:items-end">
        <div className="flex items-center gap-2">
          <Calendar className="size-4 text-muted-foreground" />
          <span className="text-xs font-semibold uppercase tracking-wider text-muted-foreground">时间范围</span>
        </div>
        <div className="flex flex-1 items-center gap-2">
          <Input
            type="datetime-local"
            value={startTime}
            onChange={e => setStartTime(e.target.value)}
            className="flex-1 text-xs"
            placeholder="开始时间"
          />
          <span className="text-xs text-muted-foreground">至</span>
          <Input
            type="datetime-local"
            value={endTime}
            onChange={e => setEndTime(e.target.value)}
            className="flex-1 text-xs"
            placeholder="结束时间"
          />
        </div>
        <div className="flex items-center gap-2">
          <Button size="sm" variant="outline" onClick={handleApplyDateFilter}>
            应用筛选
          </Button>
          {(activeStartTime || activeEndTime) && (
            <Button size="sm" variant="ghost" onClick={handleClearDateFilter}>
              清除
            </Button>
          )}
        </div>
      </div>

      {/* ─── Logs Table ─────────────────────────────────────────────────── */}
      <div>
        <p className="mb-3 text-xs font-semibold uppercase tracking-wider text-muted-foreground">
          请求记录
        </p>
        <BentoCard compact>
          <div className="mb-3 flex items-center gap-2 border-b border-border/60 pb-3">
            <FileText className="size-4 text-muted-foreground" />
            <span className="text-sm font-semibold">最近请求</span>
            <span className="ml-auto text-xs text-muted-foreground">
              共 {total} 条记录
            </span>
          </div>
          <Table>
            <TableHeader>
              <TableRow className="hover:bg-transparent">
                <TableHead className="text-xs">时间</TableHead>
                <TableHead className="text-xs">模型</TableHead>
                <TableHead className="text-xs">服务商</TableHead>
                <TableHead className="text-center text-xs">状态</TableHead>
                <TableHead className="text-right text-xs">延迟</TableHead>
                <TableHead className="text-right text-xs">Token</TableHead>
              </TableRow>
            </TableHeader>
            <TableBody>
              {filteredLogs.map(log => (
                <TableRow
                  key={log.id}
                  className="cursor-pointer transition-colors"
                  onClick={() => setSelectedLog(log)}
                >
                  <TableCell className="text-xs text-muted-foreground">
                    <div className="flex items-center gap-1.5">
                      <Clock className="size-3 text-muted-foreground/60" />
                      {new Date(log.createdAt).toLocaleString()}
                    </div>
                  </TableCell>
                  <TableCell>
                    <span className="text-xs font-medium">{log.modelName}</span>
                  </TableCell>
                  <TableCell className="text-xs text-muted-foreground">
                    {log.providerName || '-'}
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
                  <TableCell className="text-right text-xs tabular-nums">
                    <span className={log.totalLatencyMs > 5000 ? 'text-amber-600 dark:text-amber-400' : 'text-muted-foreground'}>
                      {formatLatency(log.totalLatencyMs)}
                    </span>
                  </TableCell>
                  <TableCell className="text-right text-xs tabular-nums text-muted-foreground">
                    {(log.inputTokens + log.outputTokens).toLocaleString()}
                  </TableCell>
                </TableRow>
              ))}
              {filteredLogs.length === 0 && (
                <TableRow>
                  <TableCell colSpan={6} className="py-12 text-center">
                    <div className="flex flex-col items-center gap-2">
                      <FileText className="size-8 text-muted-foreground/40" />
                      <p className="text-sm text-muted-foreground">暂无日志记录</p>
                    </div>
                  </TableCell>
                </TableRow>
              )}
            </TableBody>
          </Table>
        </BentoCard>

        {/* ─── Pagination ──────────────────────────────────────────────── */}
        {total > pageSize && (
          <div className="mt-4 flex items-center justify-between">
            <p className="text-xs text-muted-foreground">
              共 {total} 条记录，第 {page}/{totalPages} 页
            </p>
            <div className="flex items-center gap-1.5">
              <Button
                variant="outline"
                size="sm"
                onClick={() => setPage(p => Math.max(1, p - 1))}
                disabled={page <= 1}
              >
                <ChevronLeft className="size-3.5" />
                上一页
              </Button>
              <div className="flex items-center gap-1 px-2">
                {Array.from({ length: Math.min(5, totalPages) }, (_, i) => {
                  let pageNum: number
                  if (totalPages <= 5) {
                    pageNum = i + 1
                  } else if (page <= 3) {
                    pageNum = i + 1
                  } else if (page >= totalPages - 2) {
                    pageNum = totalPages - 4 + i
                  } else {
                    pageNum = page - 2 + i
                  }
                  return (
                    <button
                      key={pageNum}
                      onClick={() => setPage(pageNum)}
                      className={`flex h-8 w-8 items-center justify-center rounded-md text-xs font-medium transition-all ${
                        page === pageNum
                          ? 'bg-primary text-primary-foreground'
                          : 'text-muted-foreground hover:bg-muted hover:text-foreground'
                      }`}
                    >
                      {pageNum}
                    </button>
                  )
                })}
              </div>
              <Button
                variant="outline"
                size="sm"
                onClick={() => setPage(p => Math.min(totalPages, p + 1))}
                disabled={page >= totalPages}
              >
                下一页
                <ChevronRight className="size-3.5" />
              </Button>
            </div>
          </div>
        )}
      </div>

      {/* ─── Detail Modal ───────────────────────────────────────────────── */}
      <Dialog open={!!selectedLog} onOpenChange={() => setSelectedLog(null)}>
        <DialogContent className="max-w-2xl">
          <DialogHeader>
            <DialogTitle className="flex items-center gap-2">
              <FileText className="size-5 text-muted-foreground" />
              请求详情
            </DialogTitle>
          </DialogHeader>
          {selectedLog && (
            <div className="space-y-5">
              {/* Status & Timing */}
              <div className="flex items-center gap-3">
                <Badge
                  variant={selectedLog.success ? 'default' : 'destructive'}
                  className={`text-xs ${
                    selectedLog.success
                      ? 'border-emerald-200 bg-emerald-50 text-emerald-700 dark:border-emerald-800 dark:bg-emerald-950/30 dark:text-emerald-400'
                      : ''
                  }`}
                >
                  {selectedLog.success ? '成功' : '失败'}
                </Badge>
                <span className="text-sm text-muted-foreground">
                  {new Date(selectedLog.createdAt).toLocaleString()}
                </span>
              </div>

              {/* Request Info */}
              <div className="rounded-xl border border-border/60 bg-muted/30 p-4">
                <p className="mb-3 text-xs font-semibold uppercase tracking-wider text-muted-foreground">
                  请求信息
                </p>
                <div className="grid grid-cols-2 gap-4">
                  <div>
                    <p className="text-xs text-muted-foreground">请求ID</p>
                    <p className="mt-0.5 font-mono text-sm">{selectedLog.requestId}</p>
                  </div>
                  <div>
                    <p className="text-xs text-muted-foreground">路由</p>
                    <p className="mt-0.5 text-sm">{selectedLog.route || '-'}</p>
                  </div>
                  <div>
                    <p className="text-xs text-muted-foreground">模型</p>
                    <p className="mt-0.5 text-sm font-medium">{selectedLog.modelName}</p>
                  </div>
                  <div>
                    <p className="text-xs text-muted-foreground">服务商</p>
                    <p className="mt-0.5 text-sm">{selectedLog.providerName || '-'}</p>
                  </div>
                  <div>
                    <p className="text-xs text-muted-foreground">接口类型</p>
                    <p className="mt-0.5 text-sm">{selectedLog.interfaceType || '-'}</p>
                  </div>
                  <div>
                    <p className="text-xs text-muted-foreground">流式</p>
                    <p className="mt-0.5 text-sm">{selectedLog.stream ? '是' : '否'}</p>
                  </div>
                  <div>
                    <p className="text-xs text-muted-foreground">API Key</p>
                    <p className="mt-0.5 text-sm">{selectedLog.apiKeyName || '-'}</p>
                  </div>
                  <div>
                    <p className="text-xs text-muted-foreground">客户端IP</p>
                    <p className="mt-0.5 font-mono text-sm">{selectedLog.clientIp || '-'}</p>
                  </div>
                </div>
              </div>

              {/* Performance */}
              <div className="rounded-xl border border-border/60 bg-muted/30 p-4">
                <p className="mb-3 text-xs font-semibold uppercase tracking-wider text-muted-foreground">
                  性能指标
                </p>
                <div className="grid grid-cols-2 gap-4 sm:grid-cols-4">
                  <div>
                    <p className="text-xs text-muted-foreground">总延迟</p>
                    <p className={`mt-0.5 text-sm font-semibold tabular-nums ${
                      selectedLog.totalLatencyMs > 5000 ? 'text-amber-600 dark:text-amber-400' : ''
                    }`}>
                      {formatLatency(selectedLog.totalLatencyMs)}
                    </p>
                  </div>
                  <div>
                    <p className="text-xs text-muted-foreground">首Token延迟</p>
                    <p className="mt-0.5 text-sm font-semibold tabular-nums">
                      {selectedLog.firstTokenLatencyMs ? formatLatency(selectedLog.firstTokenLatencyMs) : '-'}
                    </p>
                  </div>
                  <div>
                    <p className="text-xs text-muted-foreground">输入 Token</p>
                    <p className="mt-0.5 text-sm font-semibold tabular-nums">
                      {selectedLog.inputTokens.toLocaleString()}
                    </p>
                  </div>
                  <div>
                    <p className="text-xs text-muted-foreground">输出 Token</p>
                    <p className="mt-0.5 text-sm font-semibold tabular-nums">
                      {selectedLog.outputTokens.toLocaleString()}
                    </p>
                  </div>
                </div>
              </div>

              {/* Error Message */}
              {selectedLog.errorMessage && (
                <div className="rounded-xl border border-red-200 bg-red-50 p-4 dark:border-red-800 dark:bg-red-950/20">
                  <div className="mb-2 flex items-center gap-2">
                    <AlertCircle className="size-4 text-red-600 dark:text-red-400" />
                    <p className="text-xs font-semibold uppercase tracking-wider text-red-600 dark:text-red-400">
                      错误信息
                    </p>
                  </div>
                  <pre className="max-h-40 overflow-auto rounded-lg bg-red-100/50 p-3 text-xs text-red-700 dark:bg-red-900/20 dark:text-red-300">
                    {selectedLog.errorMessage}
                  </pre>
                </div>
              )}
            </div>
          )}
        </DialogContent>
      </Dialog>
    </div>
  )
}
