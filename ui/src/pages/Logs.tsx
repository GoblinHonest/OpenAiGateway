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
import { fetchLogs, fetchLogDetail, LogEntry } from '../api/log'
import { Button } from '@/components/ui/button'
import { BentoCard } from '@/components/ui/bento-card'
import { PageHeader } from '@/components/ui/page-header'
import { Input } from '@/components/ui/input'
import { Badge } from '@/components/ui/badge'
import { Dialog, DialogContent, DialogHeader, DialogTitle } from '@/components/ui/dialog'
import { Tabs, TabsContent, TabsList, TabsTrigger } from '@/components/ui/tabs'
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

// ─── Parsed Request Body ────────────────────────────────────────────────────

function ParsedRequestBody({ body }: { body: string }) {
  try {
    const parsed = JSON.parse(body)
    if (!parsed.messages || !Array.isArray(parsed.messages)) {
      return <pre className="max-h-96 overflow-auto rounded-lg bg-muted/50 p-3 text-[11px] font-mono leading-relaxed whitespace-pre-wrap">{JSON.stringify(parsed, null, 2)}</pre>
    }

    const roleColors: Record<string, { bg: string; border: string; label: string }> = {
      system: { bg: 'bg-slate-50 dark:bg-slate-900/30', border: 'border-slate-200 dark:border-slate-700', label: 'System' },
      user: { bg: 'bg-blue-50 dark:bg-blue-950/30', border: 'border-blue-200 dark:border-blue-800', label: 'User' },
      assistant: { bg: 'bg-emerald-50 dark:bg-emerald-950/30', border: 'border-emerald-200 dark:border-emerald-800', label: 'Assistant' },
      tool: { bg: 'bg-amber-50 dark:bg-amber-950/30', border: 'border-amber-200 dark:border-amber-800', label: 'Tool' },
    }

    const roleCounters: Record<string, number> = {}

    return (
      <div className="space-y-2">
        {/* Other fields */}
        {Object.entries(parsed).filter(([k]) => k !== 'messages').map(([k, v]) => (
          <div key={k} className="flex gap-2 text-[11px]">
            <span className="font-semibold text-muted-foreground shrink-0">{k}:</span>
            <span className="font-mono break-all">{typeof v === 'string' ? v : JSON.stringify(v)}</span>
          </div>
        ))}
        {/* Messages */}
        <div className="space-y-2 mt-2">
          {parsed.messages.map((msg: any, i: number) => {
            const role = msg.role || 'unknown'
            roleCounters[role] = (roleCounters[role] || 0) + 1
            const cfg = roleColors[role] || { bg: 'bg-muted/30', border: 'border-border/60', label: role }
            const content = typeof msg.content === 'string'
              ? msg.content
              : Array.isArray(msg.content)
                ? msg.content.map((c: any) => c.text || c.type || JSON.stringify(c)).join('\n')
                : JSON.stringify(msg.content, null, 2)

            return (
              <div key={i} className={`rounded-lg border ${cfg.border} ${cfg.bg} p-3`}>
                <div className="flex items-center gap-2 mb-1.5">
                  <span className="text-[10px] font-bold uppercase tracking-wider">{cfg.label} {roleCounters[role]}</span>
                  {msg.name && <span className="text-[10px] text-muted-foreground">({msg.name})</span>}
                </div>
                <pre className="text-[11px] font-mono leading-relaxed whitespace-pre-wrap break-all max-h-48 overflow-auto">{content}</pre>
              </div>
            )
          })}
        </div>
      </div>
    )
  } catch {
    return <pre className="max-h-96 overflow-auto rounded-lg bg-muted/50 p-3 text-[11px] font-mono leading-relaxed whitespace-pre-wrap">{body}</pre>
  }
}

// ─── Parse stream chunks from body ──────────────────────────────────────────

function parseStreamChunks(body: string): { chunks: any[]; content: string; usage: any } {
  const chunks: any[] = []
  const contentParts: string[] = []
  let usage: any = null

  // Split by }{ to handle concatenated JSON objects (no newlines)
  const jsonStrings = body.replace(/}\s*\{/g, '}\n{').split('\n').filter(s => s.trim())

  for (const s of jsonStrings) {
    try {
      const obj = JSON.parse(s.trim())
      chunks.push(obj)
      if (obj.choices?.[0]?.delta?.content) {
        contentParts.push(obj.choices[0].delta.content)
      }
      if (obj.usage && (obj.usage.prompt_tokens > 0 || obj.usage.completion_tokens > 0 || obj.usage.input_tokens > 0)) {
        usage = obj.usage
      }
      if (obj.type === 'content_block_delta' && obj.delta?.text) {
        contentParts.push(obj.delta.text)
      }
      if (obj.type === 'message_delta' && obj.usage) {
        usage = obj.usage
      }
    } catch { /* skip */ }
  }

  return { chunks, content: contentParts.join(''), usage }
}

function StreamChunksView({ chunks, raw }: { chunks: any[]; raw: string }) {
  const contentParts: string[] = []
  let usage: any = null
  for (const obj of chunks) {
    if (obj.choices?.[0]?.delta?.content) contentParts.push(obj.choices[0].delta.content)
    if (obj.usage && (obj.usage.prompt_tokens > 0 || obj.usage.completion_tokens > 0)) usage = obj.usage
    if (obj.type === 'content_block_delta' && obj.delta?.text) contentParts.push(obj.delta.text)
    if (obj.type === 'message_delta' && obj.usage) usage = obj.usage
  }
  const fullContent = contentParts.join('')
  return (
    <div className="space-y-3">
      {fullContent && (
        <div className="rounded-lg border border-emerald-200 bg-emerald-50 dark:border-emerald-800 dark:bg-emerald-950/30 p-3">
          <p className="text-[10px] font-bold uppercase tracking-wider text-emerald-700 dark:text-emerald-400 mb-1.5">Assistant 回复</p>
          <pre className="text-[11px] font-mono leading-relaxed whitespace-pre-wrap">{fullContent}</pre>
        </div>
      )}
      {usage && (
        <div className="flex gap-3 text-[11px]">
          <span className="text-muted-foreground">input: <strong>{usage.prompt_tokens || usage.input_tokens || 0}</strong></span>
          <span className="text-muted-foreground">output: <strong>{usage.completion_tokens || usage.output_tokens || 0}</strong></span>
        </div>
      )}
      <pre className="max-h-48 overflow-auto rounded-lg bg-muted/50 p-3 text-[11px] font-mono leading-relaxed whitespace-pre-wrap">{raw}</pre>
    </div>
  )
}

// ─── Parsed Response Body ───────────────────────────────────────────────────

function ParsedResponseBody({ body, stream }: { body: string; stream?: boolean }) {
  try {
    const parsed = JSON.parse(body)
    if (Array.isArray(parsed)) {
      return <StreamChunksView chunks={parsed} raw={body} />
    }
    if (parsed.choices) {
      const content = parsed.choices?.[0]?.message?.content || parsed.choices?.[0]?.text || ''
      const usage = parsed.usage
      return (
        <div className="space-y-3">
          {content && (
            <div className="rounded-lg border border-emerald-200 bg-emerald-50 dark:border-emerald-800 dark:bg-emerald-950/30 p-3">
              <p className="text-[10px] font-bold uppercase tracking-wider text-emerald-700 dark:text-emerald-400 mb-1.5">Assistant</p>
              <pre className="text-[11px] font-mono leading-relaxed whitespace-pre-wrap">{content}</pre>
            </div>
          )}
          {usage && (
            <div className="flex gap-3 text-[11px]">
              <span className="text-muted-foreground">input: <strong>{usage.prompt_tokens || usage.input_tokens || 0}</strong></span>
              <span className="text-muted-foreground">output: <strong>{usage.completion_tokens || usage.output_tokens || 0}</strong></span>
            </div>
          )}
          <pre className="max-h-48 overflow-auto rounded-lg bg-muted/50 p-3 text-[11px] font-mono leading-relaxed whitespace-pre-wrap">{JSON.stringify(parsed, null, 2)}</pre>
        </div>
      )
    }
    if (parsed.content || parsed.type === 'message') {
      const text = parsed.content?.map?.((c: any) => c.text || '').join('') || ''
      return (
        <div className="space-y-3">
          {text && (
            <div className="rounded-lg border border-emerald-200 bg-emerald-50 dark:border-emerald-800 dark:bg-emerald-950/30 p-3">
              <p className="text-[10px] font-bold uppercase tracking-wider text-emerald-700 dark:text-emerald-400 mb-1.5">Assistant</p>
              <pre className="text-[11px] font-mono leading-relaxed whitespace-pre-wrap">{text}</pre>
            </div>
          )}
          <pre className="max-h-48 overflow-auto rounded-lg bg-muted/50 p-3 text-[11px] font-mono leading-relaxed whitespace-pre-wrap">{JSON.stringify(parsed, null, 2)}</pre>
        </div>
      )
    }
    return <pre className="max-h-96 overflow-auto rounded-lg bg-muted/50 p-3 text-[11px] font-mono leading-relaxed whitespace-pre-wrap">{JSON.stringify(parsed, null, 2)}</pre>
  } catch {
    // Streaming: concatenated JSON objects
    const { chunks, content, usage } = parseStreamChunks(body)
    return (
      <div className="space-y-3">
        {content && (
          <div className="rounded-lg border border-emerald-200 bg-emerald-50 dark:border-emerald-800 dark:bg-emerald-950/30 p-3">
            <p className="text-[10px] font-bold uppercase tracking-wider text-emerald-700 dark:text-emerald-400 mb-1.5">Assistant 回复</p>
            <pre className="text-[11px] font-mono leading-relaxed whitespace-pre-wrap">{content}</pre>
          </div>
        )}
        {usage && (
          <div className="flex gap-3 text-[11px]">
            <span className="text-muted-foreground">input: <strong>{usage.prompt_tokens || usage.input_tokens || 0}</strong></span>
            <span className="text-muted-foreground">output: <strong>{usage.completion_tokens || usage.output_tokens || 0}</strong></span>
          </div>
        )}
        <div>
          <p className="text-[10px] font-semibold uppercase tracking-wider text-muted-foreground mb-1.5">原始流数据 ({chunks.length} chunks)</p>
          <pre className="max-h-48 overflow-auto rounded-lg bg-muted/50 p-3 text-[11px] font-mono leading-relaxed whitespace-pre-wrap">{body}</pre>
        </div>
      </div>
    )
  }
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
                  onClick={() => {
                    setSelectedLog(log)
                    fetchLogDetail(log.requestId).then(detail => setSelectedLog(detail)).catch(() => {})
                  }}
                >
                  <TableCell className="text-xs text-muted-foreground">
                    <div className="flex items-center gap-1.5">
                      <Clock className="size-3 text-muted-foreground/60" />
                      {new Date(log.createdAt || log.timestamp).toLocaleString()}
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

      {/* ─── Detail Dialog ──────────────────────────────────────────────────── */}
      <Dialog open={!!selectedLog} onOpenChange={(open) => { if (!open) setSelectedLog(null) }}>
        <DialogContent className="max-w-3xl max-h-[85vh] flex flex-col">
          <DialogHeader>
            <DialogTitle className="flex items-center gap-2">
              <FileText className="size-5 text-muted-foreground" />
              请求详情
              {selectedLog && (
                <Badge
                  variant={selectedLog.success ? 'default' : 'destructive'}
                  className={`ml-auto text-[10px] ${
                    selectedLog.success
                      ? 'border-emerald-200 bg-emerald-50 text-emerald-700 dark:border-emerald-800 dark:bg-emerald-950/30 dark:text-emerald-400'
                      : ''
                  }`}
                >
                  {selectedLog.success ? '成功' : '失败'}
                </Badge>
              )}
            </DialogTitle>
          </DialogHeader>
          {selectedLog && (
            <Tabs defaultValue="request" className="flex flex-col min-h-0 flex-1">
              <TabsList className="shrink-0">
                <TabsTrigger value="request">请求</TabsTrigger>
                <TabsTrigger value="response">响应</TabsTrigger>
              </TabsList>

              {/* ─── Request Tab ────────────────────────────────────────────── */}
              <TabsContent value="request" className="flex-1 overflow-y-auto space-y-4 mt-3">
                {/* Meta info */}
                <div className="grid grid-cols-2 sm:grid-cols-4 gap-3">
                  {[
                    { label: '模型', value: selectedLog.modelName },
                    { label: '服务商', value: selectedLog.providerName || '-' },
                    { label: '路由', value: selectedLog.route || '-' },
                    { label: '接口', value: selectedLog.interfaceType || '-' },
                    { label: '流式', value: selectedLog.stream ? '是' : '否' },
                    { label: 'API Key', value: selectedLog.apiKeyName || '-' },
                    { label: '客户端IP', value: selectedLog.clientIp || '-' },
                    { label: '时间', value: new Date(selectedLog.createdAt || selectedLog.timestamp).toLocaleString() },
                  ].map(item => (
                    <div key={item.label} className="rounded-lg border border-border/60 bg-muted/30 px-3 py-2">
                      <p className="text-[10px] text-muted-foreground uppercase tracking-wider">{item.label}</p>
                      <p className="mt-0.5 text-xs font-medium truncate">{item.value}</p>
                    </div>
                  ))}
                </div>

                {/* Performance */}
                <div className="grid grid-cols-4 gap-3">
                  {[
                    { label: '总延迟', value: formatLatency(selectedLog.totalLatencyMs), warn: selectedLog.totalLatencyMs > 5000 },
                    { label: '首Token', value: selectedLog.firstTokenLatencyMs ? formatLatency(selectedLog.firstTokenLatencyMs) : '-' },
                    { label: '输入Token', value: selectedLog.inputTokens.toLocaleString() },
                    { label: '输出Token', value: selectedLog.outputTokens.toLocaleString() },
                  ].map(item => (
                    <div key={item.label} className="rounded-lg border border-border/60 bg-muted/30 px-3 py-2 text-center">
                      <p className="text-[10px] text-muted-foreground uppercase tracking-wider">{item.label}</p>
                      <p className={`mt-0.5 text-sm font-bold tabular-nums ${item.warn ? 'text-amber-600 dark:text-amber-400' : ''}`}>{item.value}</p>
                    </div>
                  ))}
                </div>

                {/* Request Headers */}
                {selectedLog.requestHeaders && Object.keys(selectedLog.requestHeaders).length > 0 && (
                  <div>
                    <p className="text-[10px] font-semibold uppercase tracking-wider text-muted-foreground mb-2">请求头</p>
                    <pre className="max-h-32 overflow-auto rounded-lg bg-muted/50 p-3 text-[11px] font-mono leading-relaxed">
                      {Object.entries(selectedLog.requestHeaders).map(([k, v]) => `${k}: ${v}`).join('\n')}
                    </pre>
                  </div>
                )}

                {/* Request Body - parsed as conversation */}
                {selectedLog.requestBody && (
                  <div>
                    <p className="text-[10px] font-semibold uppercase tracking-wider text-muted-foreground mb-2">请求体</p>
                    <ParsedRequestBody body={selectedLog.requestBody} />
                  </div>
                )}
              </TabsContent>

              {/* ─── Response Tab ───────────────────────────────────────────── */}
              <TabsContent value="response" className="flex-1 overflow-y-auto space-y-4 mt-3">
                {/* Response Headers */}
                {selectedLog.responseHeaders && Object.keys(selectedLog.responseHeaders).length > 0 && (
                  <div>
                    <p className="text-[10px] font-semibold uppercase tracking-wider text-muted-foreground mb-2">响应头</p>
                    <pre className="max-h-32 overflow-auto rounded-lg bg-muted/50 p-3 text-[11px] font-mono leading-relaxed">
                      {Object.entries(selectedLog.responseHeaders).map(([k, v]) => `${k}: ${v}`).join('\n')}
                    </pre>
                  </div>
                )}

                {/* Response Body */}
                {selectedLog.responseBody && (
                  <div>
                    <p className="text-[10px] font-semibold uppercase tracking-wider text-muted-foreground mb-2">响应体</p>
                    <ParsedResponseBody body={selectedLog.responseBody} stream={selectedLog.stream} />
                  </div>
                )}

                {/* Error */}
                {selectedLog.errorMessage && (
                  <div className="rounded-xl border border-red-200 bg-red-50 p-4 dark:border-red-800 dark:bg-red-950/20">
                    <div className="flex items-center gap-2 mb-2">
                      <AlertCircle className="size-4 text-red-600 dark:text-red-400" />
                      <p className="text-xs font-semibold text-red-600 dark:text-red-400">错误信息</p>
                    </div>
                    <pre className="max-h-40 overflow-auto rounded-lg bg-red-100/50 p-3 text-xs text-red-700 dark:bg-red-900/20 dark:text-red-300 whitespace-pre-wrap">
                      {selectedLog.errorMessage}
                    </pre>
                  </div>
                )}
              </TabsContent>
            </Tabs>
          )}
        </DialogContent>
      </Dialog>
    </div>
  )
}
