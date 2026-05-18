import React, { useEffect, useMemo, useState } from 'react'
import { toast } from 'sonner'
import {
  Plus,
  Trash2,
  RefreshCw,
  Search,
  Key,
  Copy,
  Eye,
  EyeOff,
  Check,
  Shield,
  ShieldOff,
  Layers,
  AlertTriangle,
  Pencil,
} from 'lucide-react'
import { fetchAPIKeys, createAPIKey, revokeAPIKey, revealAPIKey, updateAPIKey, APIKey, CreateAPIKeyResponse } from '../api/apikey'
import { fetchGroups, Group } from '../api/group'
import { Button } from '@/components/ui/button'
import { BentoCard } from '@/components/ui/bento-card'
import { PageHeader } from '@/components/ui/page-header'
import { Input } from '@/components/ui/input'
import { Badge } from '@/components/ui/badge'
import { Skeleton } from '@/components/ui/skeleton'
import {
  Table,
  TableBody,
  TableCell,
  TableHead,
  TableHeader,
  TableRow,
} from '@/components/ui/table'
import { Dialog, DialogContent, DialogHeader, DialogTitle, DialogFooter } from '@/components/ui/dialog'
import {
  Select,
  SelectContent,
  SelectItem,
  SelectTrigger,
  SelectValue,
} from '@/components/ui/select'

// ─── Loading State ──────────────────────────────────────────────────────────

function ApiKeysSkeleton() {
  return (
    <div className="space-y-6">
      <div className="flex items-center justify-between">
        <Skeleton className="h-8 w-40" />
        <div className="flex gap-2">
          <Skeleton className="h-9 w-20" />
          <Skeleton className="h-9 w-28" />
        </div>
      </div>
      {/* Metrics bar skeleton */}
      <div className="grid grid-cols-2 gap-2.5 sm:grid-cols-4">
        {Array.from({ length: 4 }).map((_, i) => (
          <Skeleton key={i} className="h-[68px] rounded-xl" />
        ))}
      </div>
      {/* Search skeleton */}
      <Skeleton className="h-10 w-full rounded-lg" />
      {/* Table skeleton */}
      <Skeleton className="h-64 rounded-2xl" />
    </div>
  )
}

// ─── Main Component ─────────────────────────────────────────────────────────

export default function ApiKeys() {
  const [apiKeys, setApiKeys] = useState<APIKey[]>([])
  const [groups, setGroups] = useState<Group[]>([])
  const [loading, setLoading] = useState(true)
  const [search, setSearch] = useState('')
  const [showAddModal, setShowAddModal] = useState(false)
  const [newKeyName, setNewKeyName] = useState('')
  const [selectedGroupId, setSelectedGroupId] = useState<string>('')
  const [generatedKey, setGeneratedKey] = useState<CreateAPIKeyResponse | null>(null)
  const [showKey, setShowKey] = useState(false)
  const [revealedKeys, setRevealedKeys] = useState<Map<string, string>>(new Map())
  const [copiedId, setCopiedId] = useState<string | null>(null)

  // Edit modal
  const [showEditModal, setShowEditModal] = useState(false)
  const [editingKey, setEditingKey] = useState<APIKey | null>(null)
  const [editName, setEditName] = useState('')
  const [editGroupId, setEditGroupId] = useState('')
  const [editRateLimit, setEditRateLimit] = useState('')

  const loadKeys = () => {
    setLoading(true)
    fetchAPIKeys()
      .then(res => setApiKeys(res.items || []))
      .catch(err => {
        toast.error('加载 API Key 失败', { description: err instanceof Error ? err.message : '未知错误' })
      })
      .finally(() => setLoading(false))
  }

  const loadGroups = () => {
    fetchGroups()
      .then(res => setGroups(res.items || []))
      .catch(err => {
        toast.error('加载分组失败', { description: err instanceof Error ? err.message : '未知错误' })
      })
  }

  useEffect(() => { loadKeys(); loadGroups() }, [])

  // ─── Derived data ────────────────────────────────────────────────────────

  const activeCount = useMemo(() => apiKeys.filter(k => k.status === 'active').length, [apiKeys])
  const revokedCount = useMemo(() => apiKeys.filter(k => k.status !== 'active').length, [apiKeys])

  const filteredKeys = useMemo(() => apiKeys.filter(k =>
    k.name?.toLowerCase().includes(search.toLowerCase()) ||
    k.key_prefix.toLowerCase().includes(search.toLowerCase())
  ), [apiKeys, search])

  // ─── Metrics bar items ──────────────────────────────────────────────────

  const metricsBar = [
    {
      label: '总计',
      value: String(apiKeys.length),
      icon: Key,
      bg: 'bg-blue-500/8',
      color: 'text-blue-600 dark:text-blue-400',
    },
    {
      label: '活跃',
      value: String(activeCount),
      icon: Shield,
      bg: 'bg-emerald-500/8',
      color: 'text-emerald-600 dark:text-emerald-400',
    },
    {
      label: '已撤销',
      value: String(revokedCount),
      icon: ShieldOff,
      bg: revokedCount > 0 ? 'bg-red-500/8' : 'bg-muted',
      color: revokedCount > 0 ? 'text-red-600 dark:text-red-400' : 'text-muted-foreground',
    },
    {
      label: '分组',
      value: String(new Set(apiKeys.filter(k => k.group_id).map(k => k.group_id)).size),
      icon: Layers,
      bg: 'bg-violet-500/8',
      color: 'text-violet-600 dark:text-violet-400',
    },
  ]

  // ─── Handlers ──────────────────────────────────────────────────────────

  const handleCreate = async () => {
    try {
      const result = await createAPIKey({
        name: newKeyName,
        group_id: selectedGroupId && selectedGroupId !== '__none__' ? selectedGroupId : undefined,
      })
      setGeneratedKey(result)
      toast.success('API Key 已生成')
      loadKeys()
    } catch (err) {
      toast.error('创建 API Key 失败', { description: err instanceof Error ? err.message : '未知错误' })
    }
  }

  const handleRevoke = async (id: string) => {
    if (!confirm('确定要撤销此 API Key 吗？')) return
    try {
      await revokeAPIKey(id)
      toast.success('API Key 已撤销')
      loadKeys()
    } catch (err) {
      toast.error('撤销 API Key 失败', { description: err instanceof Error ? err.message : '未知错误' })
    }
  }

  const copyToClipboard = (text: string, id: string) => {
    navigator.clipboard.writeText(text)
    setCopiedId(id)
    setTimeout(() => setCopiedId(null), 2000)
  }

  const handleReveal = async (id: string) => {
    if (revealedKeys.has(id)) {
      const newMap = new Map(revealedKeys)
      newMap.delete(id)
      setRevealedKeys(newMap)
    } else {
      try {
        const result = await revealAPIKey(id)
        const newMap = new Map(revealedKeys)
        newMap.set(id, result.key)
        setRevealedKeys(newMap)
      } catch (err) {
        toast.error('查看 API Key 失败', { description: err instanceof Error ? err.message : '未知错误' })
      }
    }
  }

  const openEditModal = (key: APIKey) => {
    setEditingKey(key)
    setEditName(key.name || '')
    setEditGroupId(key.group_id || '')
    setEditRateLimit(key.rate_limit ? String(key.rate_limit) : '')
    setShowEditModal(true)
  }

  const handleUpdate = async () => {
    if (!editingKey) return
    try {
      const data: Record<string, unknown> = {}
      if (editName) data.name = editName
      if (editGroupId && editGroupId !== '__none__') data.group_id = editGroupId
      else if (editGroupId === '__none__') data.group_id = ''
      if (editRateLimit) data.rate_limit_config = { rate_limit: Number(editRateLimit) }

      await updateAPIKey(editingKey.id, data)
      toast.success('API Key 已更新')
      setShowEditModal(false)
      setEditingKey(null)
      loadKeys()
    } catch (err) {
      toast.error('更新 API Key 失败', { description: err instanceof Error ? err.message : '未知错误' })
    }
  }

  const getGroupName = (groupId: string) => {
    return groups.find(g => g.id === groupId)?.name || groupId
  }

  // ─── Render ────────────────────────────────────────────────────────────

  if (loading) return <ApiKeysSkeleton />

  return (
    <div className="space-y-6">
      {/* ─── Header ─────────────────────────────────────────────────────── */}
      <PageHeader title="API Key 管理" description="管理访问 API 的密钥">
        <Button variant="outline" size="sm" onClick={loadKeys}>
          <RefreshCw className="size-3.5" />
          刷新
        </Button>
        <Button size="sm" onClick={() => { setShowAddModal(true); setGeneratedKey(null); setNewKeyName(''); setSelectedGroupId(''); setShowKey(false) }}>
          <Plus className="size-3.5" />
          生成 API Key
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
          placeholder="搜索 API Key 名称或前缀..."
          value={search}
          onChange={e => setSearch(e.target.value)}
          className="h-10 pl-9 rounded-lg border-border/60 bg-card"
        />
      </div>

      {/* ─── API Keys Table ─────────────────────────────────────────────── */}
      <div>
        <p className="mb-3 text-xs font-semibold uppercase tracking-wider text-muted-foreground">
          API Keys 列表
        </p>
        <BentoCard compact>
          <Table>
            <TableHeader>
              <TableRow className="hover:bg-transparent">
                <TableHead>名称</TableHead>
                <TableHead>API Key</TableHead>
                <TableHead>分组</TableHead>
                <TableHead className="text-center">状态</TableHead>
                <TableHead>创建时间</TableHead>
                <TableHead className="text-right">操作</TableHead>
              </TableRow>
            </TableHeader>
            <TableBody>
              {filteredKeys.length > 0 ? (
                filteredKeys.map(k => (
                  <TableRow key={k.id}>
                    <TableCell>
                      <div className="flex items-center gap-2">
                        <div className="flex h-7 w-7 shrink-0 items-center justify-center rounded-lg bg-blue-500/8">
                          <Key className="size-3.5 text-blue-600 dark:text-blue-400" />
                        </div>
                        <span className="text-sm font-medium">{k.name || '-'}</span>
                      </div>
                    </TableCell>
                    <TableCell>
                      <div className="flex items-center gap-1.5">
                        <code className="rounded-md border border-border/60 bg-muted/50 px-2 py-1 text-xs font-mono">
                          {revealedKeys.has(k.id) ? revealedKeys.get(k.id) : `${k.key_prefix}...`}
                        </code>
                        <Button
                          variant="ghost"
                          size="icon-sm"
                          className="shrink-0"
                          onClick={() => handleReveal(k.id)}
                          title={revealedKeys.has(k.id) ? '隐藏' : '显示'}
                        >
                          {revealedKeys.has(k.id) ? <EyeOff className="size-3.5" /> : <Eye className="size-3.5" />}
                        </Button>
                        {revealedKeys.has(k.id) && (
                          <Button
                            variant="ghost"
                            size="icon-sm"
                            className="shrink-0"
                            onClick={() => copyToClipboard(revealedKeys.get(k.id)!, k.id)}
                            title="复制"
                          >
                            {copiedId === k.id
                              ? <Check className="size-3.5 text-emerald-600 dark:text-emerald-400" />
                              : <Copy className="size-3.5" />
                            }
                          </Button>
                        )}
                      </div>
                    </TableCell>
                    <TableCell>
                      {k.group_id ? (
                        <Badge
                          variant="outline"
                          className="border-violet-200 bg-violet-50 text-violet-700 dark:border-violet-800 dark:bg-violet-950/30 dark:text-violet-400"
                        >
                          {getGroupName(k.group_id)}
                        </Badge>
                      ) : (
                        <span className="text-xs text-muted-foreground">-</span>
                      )}
                    </TableCell>
                    <TableCell className="text-center">
                      <Badge
                        variant="outline"
                        className={
                          k.status === 'active'
                            ? 'border-emerald-200 bg-emerald-50 text-emerald-700 dark:border-emerald-800 dark:bg-emerald-950/30 dark:text-emerald-400'
                            : 'border-red-200 bg-red-50 text-red-700 dark:border-red-800 dark:bg-red-950/30 dark:text-red-400'
                        }
                      >
                        {k.status === 'active' ? '活跃' : '已撤销'}
                      </Badge>
                    </TableCell>
                    <TableCell className="text-xs text-muted-foreground tabular-nums">
                      {new Date(k.created_at).toLocaleDateString('zh-CN', {
                        year: 'numeric',
                        month: '2-digit',
                        day: '2-digit',
                      })}
                    </TableCell>
                    <TableCell className="text-right">
                      <div className="flex items-center justify-end gap-0.5">
                        <Button
                          variant="ghost"
                          size="icon-sm"
                          onClick={() => openEditModal(k)}
                          title="编辑"
                          className="text-muted-foreground hover:text-foreground"
                        >
                          <Pencil className="size-4" />
                        </Button>
                        <Button
                          variant="ghost"
                          size="icon-sm"
                          onClick={() => handleRevoke(k.id)}
                          disabled={k.status !== 'active'}
                          title="撤销"
                        >
                          <Trash2 className={`size-4 ${k.status === 'active' ? 'text-red-500 hover:text-red-600' : 'text-muted-foreground/40'}`} />
                        </Button>
                      </div>
                    </TableCell>
                  </TableRow>
                ))
              ) : (
                <TableRow>
                  <TableCell colSpan={6} className="py-16">
                    <div className="flex flex-col items-center gap-3">
                      <div className="flex h-12 w-12 items-center justify-center rounded-2xl bg-muted">
                        <Key className="size-5 text-muted-foreground" />
                      </div>
                      <div className="text-center">
                        <p className="text-sm font-medium text-foreground">暂无 API Key</p>
                        <p className="mt-1 text-xs text-muted-foreground">
                          点击「生成 API Key」创建您的第一个密钥
                        </p>
                      </div>
                    </div>
                  </TableCell>
                </TableRow>
              )}
            </TableBody>
          </Table>
        </BentoCard>
      </div>

      {/* ─── Create / Reveal Dialog ─────────────────────────────────────── */}
      <Dialog open={showAddModal} onOpenChange={setShowAddModal}>
        <DialogContent
          className="sm:max-w-md"
          onPointerDownOutside={(e) => {
            const target = e.target as HTMLElement
            if (target.closest('[role="listbox"]') || target.closest('[role="option"]') || target.closest('[data-radix-select-viewport]')) {
              e.preventDefault()
            }
          }}
        >
          <DialogHeader>
            <DialogTitle className="flex items-center gap-2">
              {generatedKey ? (
                <>
                  <div className="flex h-8 w-8 items-center justify-center rounded-lg bg-emerald-500/8">
                    <Check className="size-4 text-emerald-600 dark:text-emerald-400" />
                  </div>
                  API Key 已生成
                </>
              ) : (
                <>
                  <div className="flex h-8 w-8 items-center justify-center rounded-lg bg-blue-500/8">
                    <Key className="size-4 text-blue-600 dark:text-blue-400" />
                  </div>
                  生成 API Key
                </>
              )}
            </DialogTitle>
          </DialogHeader>

          {generatedKey ? (
            <div className="space-y-4">
              <div className="flex items-start gap-2.5 rounded-lg border border-amber-200 bg-amber-50 p-3 dark:border-amber-800 dark:bg-amber-950/30">
                <AlertTriangle className="mt-0.5 size-4 shrink-0 text-amber-600 dark:text-amber-400" />
                <p className="text-xs text-amber-700 dark:text-amber-300">
                  请妥善保存此 Key，关闭后将无法再次查看
                </p>
              </div>
              <div className="flex items-center gap-2 rounded-lg border border-border/60 bg-muted/50 p-3">
                <code className="flex-1 break-all font-mono text-sm">
                  {showKey ? generatedKey.key : generatedKey.key.replace(/.(?=.{4})/g, '•')}
                </code>
                <Button variant="ghost" size="icon-sm" onClick={() => setShowKey(!showKey)} title={showKey ? '隐藏' : '显示'}>
                  {showKey ? <EyeOff className="size-4" /> : <Eye className="size-4" />}
                </Button>
                <Button variant="ghost" size="icon-sm" onClick={() => copyToClipboard(generatedKey.key, 'dialog')} title="复制">
                  {copiedId === 'dialog'
                    ? <Check className="size-4 text-emerald-600 dark:text-emerald-400" />
                    : <Copy className="size-4" />
                  }
                </Button>
              </div>
            </div>
          ) : (
            <div className="space-y-4">
              <div>
                <label className="mb-1.5 block text-sm font-medium">名称</label>
                <Input placeholder="例如：生产环境、测试密钥" value={newKeyName} onChange={e => setNewKeyName(e.target.value)} />
              </div>
              <div>
                <label className="mb-1.5 block text-sm font-medium">绑定分组</label>
                <Select value={selectedGroupId} onValueChange={setSelectedGroupId}>
                  <SelectTrigger>
                    <SelectValue placeholder="选择分组（可选）" />
                  </SelectTrigger>
                  <SelectContent>
                    <SelectItem value="__none__">不限制（可访问所有模型）</SelectItem>
                    {groups.map(g => (
                      <SelectItem key={g.id} value={g.id}>{g.name}</SelectItem>
                    ))}
                  </SelectContent>
                </Select>
                <p className="text-xs text-muted-foreground mt-1.5">
                  绑定分组后，此 API Key 只能访问分组内绑定的模型
                </p>
              </div>
            </div>
          )}

          <DialogFooter>
            {generatedKey ? (
              <Button onClick={() => setShowAddModal(false)} className="w-full sm:w-auto">
                完成
              </Button>
            ) : (
              <>
                <Button variant="outline" onClick={() => setShowAddModal(false)}>取消</Button>
                <Button onClick={handleCreate} disabled={!newKeyName}>
                  <Key className="size-3.5" />
                  生成
                </Button>
              </>
            )}
          </DialogFooter>
        </DialogContent>
      </Dialog>

      {/* ─── Edit API Key Dialog ────────────────────────────────────────── */}
      <Dialog open={showEditModal} onOpenChange={setShowEditModal}>
        <DialogContent
          className="sm:max-w-md"
          onPointerDownOutside={(e) => {
            const target = e.target as HTMLElement
            if (target.closest('[role="listbox"]') || target.closest('[role="option"]') || target.closest('[data-radix-select-viewport]')) {
              e.preventDefault()
            }
          }}
        >
          <DialogHeader>
            <DialogTitle className="flex items-center gap-2">
              <div className="flex h-8 w-8 items-center justify-center rounded-lg bg-violet-500/8">
                <Pencil className="size-4 text-violet-600 dark:text-violet-400" />
              </div>
              编辑 API Key
            </DialogTitle>
          </DialogHeader>
          <div className="space-y-4">
            <div>
              <label className="mb-1.5 block text-[11px] font-medium uppercase tracking-wider text-muted-foreground">名称</label>
              <Input
                placeholder="API Key 名称"
                value={editName}
                onChange={e => setEditName(e.target.value)}
              />
            </div>
            <div>
              <label className="mb-1.5 block text-[11px] font-medium uppercase tracking-wider text-muted-foreground">绑定分组</label>
              <Select value={editGroupId} onValueChange={setEditGroupId}>
                <SelectTrigger>
                  <SelectValue placeholder="选择分组（可选）" />
                </SelectTrigger>
                <SelectContent>
                  <SelectItem value="__none__">不限制（可访问所有模型）</SelectItem>
                  {groups.map(g => (
                    <SelectItem key={g.id} value={g.id}>{g.name}</SelectItem>
                  ))}
                </SelectContent>
              </Select>
            </div>
            <div>
              <label className="mb-1.5 block text-[11px] font-medium uppercase tracking-wider text-muted-foreground">速率限制（次/分钟）</label>
              <Input
                type="number"
                placeholder="留空表示不限制"
                value={editRateLimit}
                onChange={e => setEditRateLimit(e.target.value)}
              />
              <p className="mt-1.5 text-xs text-muted-foreground">限制此 Key 每分钟的最大请求数</p>
            </div>
          </div>
          <DialogFooter>
            <Button variant="outline" onClick={() => setShowEditModal(false)}>取消</Button>
            <Button onClick={handleUpdate} disabled={!editName}>
              保存
            </Button>
          </DialogFooter>
        </DialogContent>
      </Dialog>
    </div>
  )
}
