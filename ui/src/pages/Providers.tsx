import React, { useEffect, useState } from 'react'
import { toast } from 'sonner'
import {
  Plus,
  Trash2,
  RefreshCw,
  Search,
  Pencil,
  Key,
  Building2,
  CheckCircle2,
  PauseCircle,
  Wrench,
  Server,
  Shield,
  Globe,
  Settings2,
  Eye,
  EyeOff,
  Copy,
  Check,
} from 'lucide-react'
import { fetchProviders, createProvider, updateProvider, deleteProvider, Provider, FormatEndpoint } from '../api/provider'
import { fetchTokens, createToken, deleteToken, revealToken, Token } from '../api/token'
import { Button } from '@/components/ui/button'
import { BentoCard } from '@/components/ui/bento-card'
import { PageHeader } from '@/components/ui/page-header'
import { Input } from '@/components/ui/input'
import { Badge } from '@/components/ui/badge'
import { Dialog, DialogContent, DialogHeader, DialogTitle, DialogFooter } from '@/components/ui/dialog'
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
import { Skeleton } from '@/components/ui/skeleton'
import { Select, SelectContent, SelectItem, SelectTrigger, SelectValue } from '@/components/ui/select'
import { Tabs, TabsContent, TabsList, TabsTrigger } from '@/components/ui/tabs'
import {
  Table,
  TableBody,
  TableCell,
  TableHead,
  TableHeader,
  TableRow,
} from '@/components/ui/table'

// ─── Constants ──────────────────────────────────────────────────────────────

const FORMAT_OPTIONS = [
  { value: 'openai', label: 'OpenAI' },
  { value: 'anthropic', label: 'Anthropic' },
  { value: 'gemini', label: 'Gemini' },
]

const FORMAT_COLORS: Record<string, { bg: string; text: string; border: string }> = {
  openai: {
    bg: 'bg-emerald-50 dark:bg-emerald-950/30',
    text: 'text-emerald-700 dark:text-emerald-400',
    border: 'border-emerald-200 dark:border-emerald-800',
  },
  anthropic: {
    bg: 'bg-amber-50 dark:bg-amber-950/30',
    text: 'text-amber-700 dark:text-amber-400',
    border: 'border-amber-200 dark:border-amber-800',
  },
  gemini: {
    bg: 'bg-blue-50 dark:bg-blue-950/30',
    text: 'text-blue-700 dark:text-blue-400',
    border: 'border-blue-200 dark:border-blue-800',
  },
}

const PROVIDER_ICON_COLORS = [
  { bg: 'bg-violet-500/8', color: 'text-violet-600 dark:text-violet-400' },
  { bg: 'bg-blue-500/8', color: 'text-blue-600 dark:text-blue-400' },
  { bg: 'bg-emerald-500/8', color: 'text-emerald-600 dark:text-emerald-400' },
  { bg: 'bg-amber-500/8', color: 'text-amber-600 dark:text-amber-400' },
  { bg: 'bg-cyan-500/8', color: 'text-cyan-600 dark:text-cyan-400' },
  { bg: 'bg-rose-500/8', color: 'text-rose-600 dark:text-rose-400' },
  { bg: 'bg-indigo-500/8', color: 'text-indigo-600 dark:text-indigo-400' },
  { bg: 'bg-orange-500/8', color: 'text-orange-600 dark:text-orange-400' },
]

// ─── Types ──────────────────────────────────────────────────────────────────

interface ProviderFormData {
  id?: string
  name: string
  base_url: string
  format_endpoints: FormatEndpoint[]
}

const emptyForm: ProviderFormData = {
  name: '',
  base_url: '',
  format_endpoints: [{ format: 'openai', url: '', path: '' }],
}

// ─── Helpers ────────────────────────────────────────────────────────────────

function formatQuota(n: number): string {
  if (n >= 1000000) return (n / 1000000).toFixed(1) + 'M'
  if (n >= 1000) return (n / 1000).toFixed(0) + 'K'
  return n.toString()
}

function getStatusConfig(status: string) {
  switch (status) {
    case 'active':
      return {
        label: '活跃',
        badgeClass: 'border-emerald-200 bg-emerald-50 text-emerald-700 dark:border-emerald-800 dark:bg-emerald-950/30 dark:text-emerald-400',
        dotClass: 'bg-emerald-500',
      }
    case 'inactive':
      return {
        label: '停用',
        badgeClass: 'border-slate-200 bg-slate-50 text-slate-600 dark:border-slate-700 dark:bg-slate-800/50 dark:text-slate-400',
        dotClass: 'bg-slate-400',
      }
    case 'maintenance':
      return {
        label: '维护中',
        badgeClass: 'border-amber-200 bg-amber-50 text-amber-700 dark:border-amber-800 dark:bg-amber-950/30 dark:text-amber-400',
        dotClass: 'bg-amber-500',
      }
    default:
      return {
        label: status,
        badgeClass: 'border-slate-200 bg-slate-50 text-slate-600 dark:border-slate-700 dark:bg-slate-800/50 dark:text-slate-400',
        dotClass: 'bg-slate-400',
      }
  }
}

function getFormatConfig(format: string) {
  return FORMAT_COLORS[format] || {
    bg: 'bg-slate-50 dark:bg-slate-800/50',
    text: 'text-slate-700 dark:text-slate-400',
    border: 'border-slate-200 dark:border-slate-700',
  }
}

// ─── Loading State ──────────────────────────────────────────────────────────

function ProvidersSkeleton() {
  return (
    <div className="space-y-6">
      <div className="flex items-center justify-between">
        <Skeleton className="h-8 w-32" />
        <Skeleton className="h-9 w-24" />
      </div>
      <div className="grid grid-cols-2 gap-2.5 sm:grid-cols-4">
        {Array.from({ length: 4 }).map((_, i) => (
          <Skeleton key={i} className="h-[60px] rounded-xl" />
        ))}
      </div>
      <Skeleton className="h-10 w-full rounded-lg" />
      <Skeleton className="h-[400px] rounded-2xl" />
    </div>
  )
}

// ─── Main Component ─────────────────────────────────────────────────────────

export default function Providers() {
  const [providers, setProviders] = useState<Provider[]>([])
  const [loading, setLoading] = useState(true)
  const [search, setSearch] = useState('')
  const [showModal, setShowModal] = useState(false)
  const [editingProvider, setEditingProvider] = useState<Provider | null>(null)
  const [form, setForm] = useState<ProviderFormData>(emptyForm)

  // 删除确认
  const [deleteTarget, setDeleteTarget] = useState<Provider | null>(null)
  const [deleteTokenTarget, setDeleteTokenTarget] = useState<Token | null>(null)

  // 详情弹窗
  const [detailProvider, setDetailProvider] = useState<Provider | null>(null)
  const [showDetailModal, setShowDetailModal] = useState(false)
  const [tokens, setTokens] = useState<Token[]>([])
  const [tokensLoading, setTokensLoading] = useState(false)
  const [showTokenModal, setShowTokenModal] = useState(false)
  const [tokenForm, setTokenForm] = useState({ provider_id: '', name: '', token_value: '', quota_total: 1000000 })

  // Token reveal
  const [revealTokenValue, setRevealTokenValue] = useState<string | null>(null)
  const [revealTokenLoading, setRevealTokenLoading] = useState(false)
  const [copiedToken, setCopiedToken] = useState(false)

  // ─── Data Loading ───────────────────────────────────────────────────────

  const loadProviders = () => {
    setLoading(true)
    fetchProviders()
      .then(res => setProviders(res.items))
      .catch(err => {
        toast.error('加载服务商失败', { description: err instanceof Error ? err.message : '未知错误' })
      })
      .finally(() => setLoading(false))
  }

  useEffect(() => { loadProviders() }, [])

  const loadTokens = async (providerId: string) => {
    setTokensLoading(true)
    try {
      const res = await fetchTokens(providerId)
      setTokens(res.items)
    } catch (err) {
      toast.error('加载 Token 失败', { description: err instanceof Error ? err.message : '未知错误' })
    } finally {
      setTokensLoading(false)
    }
  }

  // ─── Modal Handlers ────────────────────────────────────────────────────

  const openDetail = (p: Provider) => {
    setDetailProvider(p)
    setShowDetailModal(true)
    loadTokens(p.id)
  }

  const openAddModal = () => {
    setEditingProvider(null)
    setForm(emptyForm)
    setShowModal(true)
  }

  const openEditModal = (p: Provider) => {
    setEditingProvider(p)
    setForm({
      id: p.id,
      name: p.name,
      base_url: p.base_url,
      format_endpoints: p.format_endpoints?.length
        ? p.format_endpoints
        : (p.supported_formats || ['openai']).map(f => ({ format: f, url: '', path: '' })),
    })
    setShowModal(true)
  }

  const handleSave = async () => {
    try {
      const data = {
        ...form,
        supported_formats: form.format_endpoints.map(ep => ep.format),
      }
      if (editingProvider) {
        await updateProvider(editingProvider.id, data)
        toast.success('服务商已更新')
      } else {
        await createProvider(data)
        toast.success('服务商已创建')
      }
      setShowModal(false)
      setForm(emptyForm)
      setEditingProvider(null)
      loadProviders()
    } catch (err) {
      toast.error('保存服务商失败', { description: err instanceof Error ? err.message : '未知错误' })
    }
  }

  const handleDelete = async () => {
    if (!deleteTarget) return
    try {
      await deleteProvider(deleteTarget.id)
      toast.success('服务商已删除')
      setDeleteTarget(null)
      loadProviders()
    } catch (err) {
      toast.error('删除服务商失败', { description: err instanceof Error ? err.message : '未知错误' })
    }
  }

  // ─── Token Handlers ────────────────────────────────────────────────────

  const openAddToken = () => {
    if (!detailProvider) return
    setTokenForm({ provider_id: detailProvider.id, name: '', token_value: '', quota_total: 1000000 })
    setShowTokenModal(true)
  }

  const handleAddToken = async () => {
    try {
      await createToken(tokenForm)
      toast.success('Token 已创建')
      setShowTokenModal(false)
      loadTokens(tokenForm.provider_id)
    } catch (err) {
      toast.error('创建 Token 失败', { description: err instanceof Error ? err.message : '未知错误' })
    }
  }

  const handleDeleteToken = async () => {
    if (!deleteTokenTarget) return
    try {
      await deleteToken(deleteTokenTarget.id)
      toast.success('Token 已删除')
      setDeleteTokenTarget(null)
      if (detailProvider) loadTokens(detailProvider.id)
    } catch (err) {
      toast.error('删除 Token 失败', { description: err instanceof Error ? err.message : '未知错误' })
    }
  }

  const handleRevealToken = async (tokenId: string) => {
    setRevealTokenLoading(true)
    setRevealTokenValue(null)
    try {
      const result = await revealToken(tokenId)
      setRevealTokenValue(result.token_value)
    } catch (err) {
      toast.error('查看 Token 失败', { description: err instanceof Error ? err.message : '未知错误' })
    } finally {
      setRevealTokenLoading(false)
    }
  }

  const copyTokenToClipboard = (text: string) => {
    navigator.clipboard.writeText(text)
    setCopiedToken(true)
    setTimeout(() => setCopiedToken(false), 2000)
  }

  // ─── Format Endpoint Handlers ──────────────────────────────────────────

  const addFormatEndpoint = () => {
    setForm({
      ...form,
      format_endpoints: [...form.format_endpoints, { format: 'openai', url: '', path: '' }],
    })
  }

  const removeFormatEndpoint = (index: number) => {
    setForm({
      ...form,
      format_endpoints: form.format_endpoints.filter((_, i) => i !== index),
    })
  }

  const updateFormatEndpoint = (index: number, field: keyof FormatEndpoint, value: string) => {
    const newEndpoints = [...form.format_endpoints]
    newEndpoints[index] = { ...newEndpoints[index], [field]: value }
    setForm({ ...form, format_endpoints: newEndpoints })
  }

  // ─── Derived Data ──────────────────────────────────────────────────────

  const filteredProviders = providers.filter(p =>
    p.name.toLowerCase().includes(search.toLowerCase())
  )

  const activeCount = providers.filter(p => p.status === 'active').length
  const inactiveCount = providers.filter(p => p.status === 'inactive').length
  const maintenanceCount = providers.filter(p => p.status === 'maintenance').length

  // ─── Metrics Bar Config ────────────────────────────────────────────────

  const metricsBar = [
    {
      label: '服务商总数',
      value: String(providers.length),
      icon: Building2,
      bg: 'bg-violet-500/8',
      color: 'text-violet-600 dark:text-violet-400',
    },
    {
      label: '活跃',
      value: String(activeCount),
      icon: CheckCircle2,
      bg: 'bg-emerald-500/8',
      color: 'text-emerald-600 dark:text-emerald-400',
    },
    {
      label: '停用',
      value: String(inactiveCount),
      icon: PauseCircle,
      bg: 'bg-slate-500/8',
      color: 'text-slate-600 dark:text-slate-400',
    },
    {
      label: '维护中',
      value: String(maintenanceCount),
      icon: Wrench,
      bg: 'bg-amber-500/8',
      color: 'text-amber-600 dark:text-amber-400',
    },
  ]

  // ─── Loading ───────────────────────────────────────────────────────────

  if (loading) return <ProvidersSkeleton />

  // ─── Render ────────────────────────────────────────────────────────────

  return (
    <div className="space-y-6">
      {/* ─── Header ─────────────────────────────────────────────────────── */}
      <PageHeader title="服务商管理" description="管理 AI 服务提供商配置和 API 密钥">
        <Button variant="outline" size="sm" onClick={loadProviders}>
          <RefreshCw className="size-3.5" />
          刷新
        </Button>
        <Button onClick={openAddModal}>
          <Plus className="size-4" />
          添加服务商
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

      {/* ─── Search ─────────────────────────────────────────────────────── */}
      <div className="relative">
        <Search className="absolute left-3 top-1/2 size-4 -translate-y-1/2 text-muted-foreground" />
        <Input
          placeholder="搜索服务商..."
          value={search}
          onChange={e => setSearch(e.target.value)}
          className="pl-9"
        />
      </div>

      {/* ─── Providers Table ────────────────────────────────────────────── */}
      <div>
        <p className="mb-3 text-xs font-semibold uppercase tracking-wider text-muted-foreground">
          服务商列表
        </p>
        <BentoCard compact>
          <div className="overflow-x-auto">
            <Table>
              <TableHeader>
                <TableRow className="hover:bg-transparent">
                  <TableHead>服务商</TableHead>
                  <TableHead>Base URL</TableHead>
                  <TableHead>支持格式</TableHead>
                  <TableHead className="text-center">状态</TableHead>
                  <TableHead className="text-right">操作</TableHead>
                </TableRow>
              </TableHeader>
              <TableBody>
                {filteredProviders.map((p, idx) => {
                  const statusCfg = getStatusConfig(p.status)
                  const iconColor = PROVIDER_ICON_COLORS[idx % PROVIDER_ICON_COLORS.length]
                  return (
                    <TableRow key={p.id} className="group">
                      <TableCell>
                        <div className="flex items-center gap-3">
                          <div className={`flex h-9 w-9 shrink-0 items-center justify-center rounded-lg ${iconColor.bg}`}>
                            <Building2 className={`size-4 ${iconColor.color}`} />
                          </div>
                          <div>
                            <button
                              onClick={() => openDetail(p)}
                              className="text-sm font-semibold text-foreground hover:text-primary hover:underline cursor-pointer transition-colors"
                            >
                              {p.name}
                            </button>
                            <p className="text-[11px] text-muted-foreground">
                              {p.format_endpoints?.length || p.supported_formats?.length || 0} 个端点
                            </p>
                          </div>
                        </div>
                      </TableCell>
                      <TableCell>
                        <span className="font-mono text-xs text-muted-foreground">{p.base_url}</span>
                      </TableCell>
                      <TableCell>
                        <div className="flex flex-wrap gap-1.5">
                          {(p.format_endpoints?.length
                            ? p.format_endpoints.map(ep => ep.format)
                            : p.supported_formats || []
                          ).map(f => {
                            const fmtCfg = getFormatConfig(f)
                            return (
                              <span
                                key={f}
                                className={`inline-flex items-center rounded-md border px-2 py-0.5 text-[10px] font-semibold ${fmtCfg.bg} ${fmtCfg.text} ${fmtCfg.border}`}
                              >
                                {f}
                              </span>
                            )
                          })}
                        </div>
                      </TableCell>
                      <TableCell className="text-center">
                        <span className={`inline-flex items-center gap-1.5 rounded-full border px-2.5 py-1 text-[10px] font-semibold ${statusCfg.badgeClass}`}>
                          <span className={`h-1.5 w-1.5 rounded-full ${statusCfg.dotClass}`} />
                          {statusCfg.label}
                        </span>
                      </TableCell>
                      <TableCell className="text-right">
                        <div className="flex items-center justify-end gap-0.5 opacity-60 transition-opacity group-hover:opacity-100">
                          <Button
                            variant="ghost"
                            size="icon-sm"
                            onClick={() => openDetail(p)}
                            title="详情"
                            className="text-muted-foreground hover:text-foreground"
                          >
                            <Key className="size-4" />
                          </Button>
                          <Button
                            variant="ghost"
                            size="icon-sm"
                            onClick={() => openEditModal(p)}
                            title="编辑"
                            className="text-muted-foreground hover:text-foreground"
                          >
                            <Pencil className="size-4" />
                          </Button>
                          <Button
                            variant="ghost"
                            size="icon-sm"
                            onClick={() => setDeleteTarget(p)}
                            title="删除"
                            className="text-muted-foreground hover:text-destructive"
                          >
                            <Trash2 className="size-4" />
                          </Button>
                        </div>
                      </TableCell>
                    </TableRow>
                  )
                })}
                {filteredProviders.length === 0 && (
                  <TableRow>
                    <TableCell colSpan={5} className="py-16">
                      <div className="flex flex-col items-center justify-center text-muted-foreground">
                        <div className="flex h-14 w-14 items-center justify-center rounded-2xl bg-muted/50 mb-3">
                          <Server className="size-6" />
                        </div>
                        <p className="text-sm font-medium">暂无服务商</p>
                        <p className="mt-1 text-xs">点击上方"添加服务商"按钮开始配置</p>
                      </div>
                    </TableCell>
                  </TableRow>
                )}
              </TableBody>
            </Table>
          </div>
        </BentoCard>
      </div>

      {/* ─── Detail Modal ───────────────────────────────────────────────── */}
      <Dialog open={showDetailModal} onOpenChange={setShowDetailModal}>
        <DialogContent className="max-w-2xl">
          <DialogHeader>
            <DialogTitle className="flex items-center gap-3">
              {detailProvider && (
                <>
                  <div className="flex h-10 w-10 items-center justify-center rounded-xl bg-violet-500/8">
                    <Building2 className="size-5 text-violet-600 dark:text-violet-400" />
                  </div>
                  <div>
                    <span className="text-lg">{detailProvider.name}</span>
                    <p className="text-xs font-normal text-muted-foreground">
                      {detailProvider.base_url}
                    </p>
                  </div>
                </>
              )}
            </DialogTitle>
          </DialogHeader>
          <Tabs defaultValue="info">
            <TabsList>
              <TabsTrigger value="info">
                <Settings2 className="mr-1.5 size-3.5" />
                基本信息
              </TabsTrigger>
              <TabsTrigger value="tokens">
                <Key className="mr-1.5 size-3.5" />
                Token管理
              </TabsTrigger>
            </TabsList>
            <TabsContent value="info" className="space-y-4 pt-4">
              {detailProvider && (
                <>
                  <div className="grid grid-cols-2 gap-4">
                    <div className="rounded-lg border border-border/60 bg-muted/30 p-3">
                      <p className="text-[10px] font-medium uppercase tracking-wider text-muted-foreground">名称</p>
                      <p className="mt-1 text-sm font-semibold">{detailProvider.name}</p>
                    </div>
                    <div className="rounded-lg border border-border/60 bg-muted/30 p-3">
                      <p className="text-[10px] font-medium uppercase tracking-wider text-muted-foreground">状态</p>
                      <div className="mt-1">
                        {(() => {
                          const cfg = getStatusConfig(detailProvider.status)
                          return (
                            <span className={`inline-flex items-center gap-1.5 rounded-full border px-2.5 py-1 text-[10px] font-semibold ${cfg.badgeClass}`}>
                              <span className={`h-1.5 w-1.5 rounded-full ${cfg.dotClass}`} />
                              {cfg.label}
                            </span>
                          )
                        })()}
                      </div>
                    </div>
                  </div>
                  <div className="rounded-lg border border-border/60 bg-muted/30 p-3">
                    <p className="text-[10px] font-medium uppercase tracking-wider text-muted-foreground">Base URL</p>
                    <p className="mt-1 text-sm font-mono">{detailProvider.base_url}</p>
                  </div>
                  <div className="rounded-lg border border-border/60 bg-muted/30 p-3">
                    <p className="text-[10px] font-medium uppercase tracking-wider text-muted-foreground">格式端点</p>
                    <div className="mt-2 space-y-2">
                      {detailProvider.format_endpoints?.map((ep, i) => {
                        const fmtCfg = getFormatConfig(ep.format)
                        return (
                          <div key={i} className="flex items-center gap-2 text-sm">
                            <span className={`inline-flex items-center rounded-md border px-2 py-0.5 text-[10px] font-semibold ${fmtCfg.bg} ${fmtCfg.text} ${fmtCfg.border}`}>
                              {ep.format}
                            </span>
                            <span className="font-mono text-xs text-muted-foreground">{ep.url || detailProvider.base_url}</span>
                          </div>
                        )
                      })}
                    </div>
                  </div>
                </>
              )}
            </TabsContent>
            <TabsContent value="tokens" className="pt-4">
              <div className="flex items-center justify-between mb-4">
                <div className="flex items-center gap-2">
                  <Shield className="size-4 text-muted-foreground" />
                  <h4 className="text-sm font-semibold">Token 列表</h4>
                </div>
                <Button size="sm" onClick={openAddToken}>
                  <Plus className="size-3.5" />
                  添加 Token
                </Button>
              </div>
              {tokensLoading ? (
                <Skeleton className="h-32 rounded-xl" />
              ) : tokens.length > 0 ? (
                <BentoCard compact>
                  <Table>
                    <TableHeader>
                      <TableRow className="hover:bg-transparent">
                        <TableHead>名称</TableHead>
                        <TableHead className="text-center">状态</TableHead>
                        <TableHead className="text-right">配额</TableHead>
                        <TableHead className="text-right">已用</TableHead>
                        <TableHead className="text-right">操作</TableHead>
                      </TableRow>
                    </TableHeader>
                    <TableBody>
                      {tokens.map(t => (
                        <TableRow key={t.id}>
                          <TableCell className="font-medium text-sm">{t.name || t.id}</TableCell>
                          <TableCell className="text-center">
                            {(() => {
                              const cfg = getStatusConfig(t.status)
                              return (
                                <span className={`inline-flex items-center gap-1.5 rounded-full border px-2 py-0.5 text-[10px] font-semibold ${cfg.badgeClass}`}>
                                  <span className={`h-1.5 w-1.5 rounded-full ${cfg.dotClass}`} />
                                  {cfg.label}
                                </span>
                              )
                            })()}
                          </TableCell>
                          <TableCell className="text-right text-sm tabular-nums text-muted-foreground">
                            {t.quota_total ? formatQuota(t.quota_total) : '-'}
                          </TableCell>
                          <TableCell className="text-right text-sm tabular-nums text-muted-foreground">
                            {t.quota_used ? formatQuota(t.quota_used) : '0'}
                          </TableCell>
                          <TableCell className="text-right">
                            <div className="flex items-center justify-end gap-0.5">
                              <Button
                                variant="ghost"
                                size="icon-sm"
                                onClick={() => handleRevealToken(t.id)}
                                title="查看 Token"
                                className="text-muted-foreground hover:text-foreground"
                              >
                                <Eye className="size-3.5" />
                              </Button>
                              <Button
                                variant="ghost"
                                size="icon-sm"
                                onClick={() => setDeleteTokenTarget(t)}
                                className="text-muted-foreground hover:text-destructive"
                              >
                                <Trash2 className="size-3.5" />
                              </Button>
                            </div>
                          </TableCell>
                        </TableRow>
                      ))}
                    </TableBody>
                  </Table>
                </BentoCard>
              ) : (
                <div className="flex flex-col items-center justify-center py-12 text-muted-foreground">
                  <div className="flex h-14 w-14 items-center justify-center rounded-2xl bg-muted/50 mb-3">
                    <Key className="size-6" />
                  </div>
                  <p className="text-sm font-medium">暂无 Token</p>
                  <p className="mt-1 text-xs">点击上方按钮添加 Token</p>
                </div>
              )}
            </TabsContent>
          </Tabs>
        </DialogContent>
      </Dialog>

      {/* ─── Add/Edit Provider Modal ────────────────────────────────────── */}
      <Dialog open={showModal} onOpenChange={setShowModal}>
        <DialogContent className="max-w-lg">
          <DialogHeader>
            <DialogTitle className="flex items-center gap-2">
              {editingProvider ? (
                <>
                  <Pencil className="size-4 text-muted-foreground" />
                  编辑服务商
                </>
              ) : (
                <>
                  <Plus className="size-4 text-muted-foreground" />
                  添加服务商
                </>
              )}
            </DialogTitle>
          </DialogHeader>
          <div className="space-y-4">
            <div>
              <label className="mb-1.5 block text-[11px] font-medium uppercase tracking-wider text-muted-foreground">名称</label>
              <Input
                placeholder="例如：MiMo"
                value={form.name}
                onChange={e => setForm({ ...form, name: e.target.value })}
              />
            </div>
            <div>
              <label className="mb-1.5 block text-[11px] font-medium uppercase tracking-wider text-muted-foreground">默认 Base URL</label>
              <Input
                placeholder="https://api.mimo.com"
                value={form.base_url}
                onChange={e => setForm({ ...form, base_url: e.target.value })}
              />
              <p className="mt-1.5 text-xs text-muted-foreground">默认URL，未配置独立端点时使用</p>
            </div>
            <div>
              <div className="mb-2 flex items-center justify-between">
                <label className="text-[11px] font-medium uppercase tracking-wider text-muted-foreground">格式端点</label>
                <Button variant="ghost" size="xs" onClick={addFormatEndpoint} className="text-xs text-primary">
                  <Plus className="mr-1 size-3" />
                  添加
                </Button>
              </div>
              <p className="mb-3 text-xs text-muted-foreground">为每个格式配置独立的URL，Token可共享</p>
              <div className="space-y-2">
                {form.format_endpoints.map((ep, index) => {
                  const fmtCfg = getFormatConfig(ep.format)
                  return (
                    <div key={index} className="flex gap-2 rounded-lg border border-border/60 bg-muted/30 p-3">
                      <Select
                        value={ep.format}
                        onValueChange={value => updateFormatEndpoint(index, 'format', value)}
                      >
                        <SelectTrigger className="w-[130px]">
                          <SelectValue />
                        </SelectTrigger>
                        <SelectContent>
                          {FORMAT_OPTIONS.map(opt => (
                            <SelectItem key={opt.value} value={opt.value}>{opt.label}</SelectItem>
                          ))}
                        </SelectContent>
                      </Select>
                      <Input
                        placeholder="URL"
                        value={ep.url}
                        onChange={e => updateFormatEndpoint(index, 'url', e.target.value)}
                        className="flex-1"
                      />
                      <Input
                        placeholder="路径"
                        value={ep.path}
                        onChange={e => updateFormatEndpoint(index, 'path', e.target.value)}
                        className="w-32"
                      />
                      {form.format_endpoints.length > 1 && (
                        <Button
                          variant="ghost"
                          size="icon-sm"
                          onClick={() => removeFormatEndpoint(index)}
                          className="text-muted-foreground hover:text-destructive"
                        >
                          <Trash2 className="size-4" />
                        </Button>
                      )}
                    </div>
                  )
                })}
              </div>
            </div>
          </div>
          <DialogFooter>
            <Button variant="outline" onClick={() => setShowModal(false)}>取消</Button>
            <Button onClick={handleSave} disabled={!form.name || !form.base_url}>
              {editingProvider ? '保存' : '添加'}
            </Button>
          </DialogFooter>
        </DialogContent>
      </Dialog>

      {/* ─── Add Token Modal ────────────────────────────────────────────── */}
      <Dialog open={showTokenModal} onOpenChange={setShowTokenModal}>
        <DialogContent>
          <DialogHeader>
            <DialogTitle className="flex items-center gap-2">
              <Key className="size-4 text-muted-foreground" />
              添加 Token
            </DialogTitle>
          </DialogHeader>
          <div className="space-y-4">
            <div>
              <label className="mb-1.5 block text-[11px] font-medium uppercase tracking-wider text-muted-foreground">名称</label>
              <Input
                placeholder="Token名称"
                value={tokenForm.name}
                onChange={e => setTokenForm({ ...tokenForm, name: e.target.value })}
              />
            </div>
            <div>
              <label className="mb-1.5 block text-[11px] font-medium uppercase tracking-wider text-muted-foreground">Token值</label>
              <Input
                placeholder="sk-xxxx"
                type="password"
                value={tokenForm.token_value}
                onChange={e => setTokenForm({ ...tokenForm, token_value: e.target.value })}
              />
              <p className="mt-1.5 text-xs text-muted-foreground">Token将明文存储，请确保数据库安全</p>
            </div>
            <div>
              <label className="mb-1.5 block text-[11px] font-medium uppercase tracking-wider text-muted-foreground">配额限制</label>
              <Input
                type="number"
                placeholder="1000000"
                value={tokenForm.quota_total}
                onChange={e => setTokenForm({ ...tokenForm, quota_total: Number(e.target.value) })}
              />
              <p className="mt-1.5 text-xs text-muted-foreground">Token数量限制，留空表示无限制</p>
            </div>
          </div>
          <DialogFooter>
            <Button variant="outline" onClick={() => setShowTokenModal(false)}>取消</Button>
            <Button onClick={handleAddToken} disabled={!tokenForm.token_value}>添加</Button>
          </DialogFooter>
        </DialogContent>
      </Dialog>

      {/* ─── Delete Provider AlertDialog ────────────────────────────────── */}
      <AlertDialog open={!!deleteTarget} onOpenChange={(open) => { if (!open) setDeleteTarget(null) }}>
        <AlertDialogContent>
          <AlertDialogHeader>
            <AlertDialogTitle>确定要删除此服务商吗？</AlertDialogTitle>
            <AlertDialogDescription>
              删除服务商 "{deleteTarget?.name}" 后，该服务商下的所有Token也将被删除。此操作不可撤销。
            </AlertDialogDescription>
          </AlertDialogHeader>
          <AlertDialogFooter>
            <AlertDialogCancel onClick={() => setDeleteTarget(null)}>取消</AlertDialogCancel>
            <Button variant="destructive" onClick={handleDelete}>
              删除
            </Button>
          </AlertDialogFooter>
        </AlertDialogContent>
      </AlertDialog>

      {/* ─── Delete Token AlertDialog ───────────────────────────────────── */}
      <AlertDialog open={!!deleteTokenTarget} onOpenChange={(open) => { if (!open) setDeleteTokenTarget(null) }}>
        <AlertDialogContent>
          <AlertDialogHeader>
            <AlertDialogTitle>确定要删除此Token吗？</AlertDialogTitle>
            <AlertDialogDescription>
              删除Token "{deleteTokenTarget?.name || deleteTokenTarget?.id}" 后将无法恢复。
            </AlertDialogDescription>
          </AlertDialogHeader>
          <AlertDialogFooter>
            <AlertDialogCancel onClick={() => setDeleteTokenTarget(null)}>取消</AlertDialogCancel>
            <Button variant="destructive" onClick={handleDeleteToken}>
              删除
            </Button>
          </AlertDialogFooter>
        </AlertDialogContent>
      </AlertDialog>

      {/* ─── Token Reveal Dialog ───────────────────────────────────────── */}
      <Dialog open={revealTokenValue !== null || revealTokenLoading} onOpenChange={(open) => { if (!open) { setRevealTokenValue(null); setRevealTokenLoading(false) } }}>
        <DialogContent className="sm:max-w-md">
          <DialogHeader>
            <DialogTitle className="flex items-center gap-2">
              <div className="flex h-8 w-8 items-center justify-center rounded-lg bg-violet-500/8">
                <Key className="size-4 text-violet-600 dark:text-violet-400" />
              </div>
              Token 值
            </DialogTitle>
          </DialogHeader>
          {revealTokenLoading ? (
            <Skeleton className="h-16 rounded-xl" />
          ) : revealTokenValue ? (
            <div className="space-y-4">
              <div className="flex items-start gap-2.5 rounded-lg border border-amber-200 bg-amber-50 p-3 dark:border-amber-800 dark:bg-amber-950/30">
                <Shield className="mt-0.5 size-4 shrink-0 text-amber-600 dark:text-amber-400" />
                <p className="text-xs text-amber-700 dark:text-amber-300">
                  请勿在不安全的环境中暴露 Token 值。关闭后需重新查看。
                </p>
              </div>
              <div className="flex items-center gap-2 rounded-lg border border-border/60 bg-muted/50 p-3">
                <code className="flex-1 break-all font-mono text-sm">
                  {revealTokenValue}
                </code>
                <Button
                  variant="ghost"
                  size="icon-sm"
                  onClick={() => copyTokenToClipboard(revealTokenValue)}
                  title="复制"
                >
                  {copiedToken
                    ? <Check className="size-4 text-emerald-600 dark:text-emerald-400" />
                    : <Copy className="size-4" />
                  }
                </Button>
              </div>
            </div>
          ) : null}
        </DialogContent>
      </Dialog>
    </div>
  )
}
