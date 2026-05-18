import React, { useEffect, useMemo, useState } from 'react'
import { toast } from 'sonner'
import {
  Plus, Trash2, RefreshCw, Search, Users, X, Pencil, Check,
  Layers, GitBranch, Shuffle, Zap, Target, BarChart3, Settings2, CheckCircle2,
} from 'lucide-react'
import { fetchGroups, createGroup, updateGroup, deleteGroup, Group } from '../api/group'
import { fetchModels, Model } from '../api/model'
import { Button } from '@/components/ui/button'
import { BentoCard } from '@/components/ui/bento-card'
import { PageHeader } from '@/components/ui/page-header'
import { Input } from '@/components/ui/input'
import { Badge } from '@/components/ui/badge'
import { Dialog, DialogContent, DialogHeader, DialogTitle, DialogFooter } from '@/components/ui/dialog'
import { Skeleton } from '@/components/ui/skeleton'
import {
  Select,
  SelectContent,
  SelectItem,
  SelectTrigger,
  SelectValue,
} from '@/components/ui/select'

// ─── Constants ──────────────────────────────────────────────────────────────

const strategyConfig: Record<string, { label: string; icon: typeof Shuffle; color: string; bg: string }> = {
  round_robin: {
    label: '轮询',
    icon: Shuffle,
    color: 'text-blue-600 dark:text-blue-400',
    bg: 'bg-blue-500/10',
  },
  weighted: {
    label: '加权',
    icon: BarChart3,
    color: 'text-violet-600 dark:text-violet-400',
    bg: 'bg-violet-500/10',
  },
  least_connections: {
    label: '最少连接',
    icon: GitBranch,
    color: 'text-emerald-600 dark:text-emerald-400',
    bg: 'bg-emerald-500/10',
  },
  priority: {
    label: '优先级',
    icon: Target,
    color: 'text-amber-600 dark:text-amber-400',
    bg: 'bg-amber-500/10',
  },
  adaptive: {
    label: '自适应',
    icon: Zap,
    color: 'text-cyan-600 dark:text-cyan-400',
    bg: 'bg-cyan-500/10',
  },
}

const strategyBadgeStyles: Record<string, string> = {
  round_robin: 'border-blue-200 bg-blue-50 text-blue-700 dark:border-blue-800 dark:bg-blue-950/30 dark:text-blue-400',
  weighted: 'border-violet-200 bg-violet-50 text-violet-700 dark:border-violet-800 dark:bg-violet-950/30 dark:text-violet-400',
  least_connections: 'border-emerald-200 bg-emerald-50 text-emerald-700 dark:border-emerald-800 dark:bg-emerald-950/30 dark:text-emerald-400',
  priority: 'border-amber-200 bg-amber-50 text-amber-700 dark:border-amber-800 dark:bg-amber-950/30 dark:text-amber-400',
  adaptive: 'border-cyan-200 bg-cyan-50 text-cyan-700 dark:border-cyan-800 dark:bg-cyan-950/30 dark:text-cyan-400',
}

// ─── Loading State ──────────────────────────────────────────────────────────

function GroupsSkeleton() {
  return (
    <div className="space-y-6">
      <div className="flex items-center justify-between">
        <Skeleton className="h-8 w-32" />
        <Skeleton className="h-9 w-24" />
      </div>
      {/* Metrics bar skeleton */}
      <div className="grid grid-cols-2 gap-2.5 sm:grid-cols-4">
        {Array.from({ length: 4 }).map((_, i) => (
          <Skeleton key={i} className="h-[72px] rounded-xl" />
        ))}
      </div>
      {/* Search skeleton */}
      <Skeleton className="h-10 w-full rounded-xl" />
      {/* Cards skeleton */}
      <div className="grid grid-cols-1 gap-4 md:grid-cols-2 lg:grid-cols-3">
        {Array.from({ length: 6 }).map((_, i) => (
          <Skeleton key={i} className="h-[180px] rounded-2xl" />
        ))}
      </div>
    </div>
  )
}

// ─── Main Component ─────────────────────────────────────────────────────────

export default function Groups() {
  const [groups, setGroups] = useState<Group[]>([])
  const [models, setModels] = useState<Model[]>([])
  const [loading, setLoading] = useState(true)
  const [search, setSearch] = useState('')
  const [showModal, setShowModal] = useState(false)
  const [editingGroup, setEditingGroup] = useState<Group | null>(null)
  const [form, setForm] = useState({
    name: '',
    description: '',
    load_balance_strategy: 'round_robin',
  })
  const [selectedModelIds, setSelectedModelIds] = useState<string[]>([])

  const loadGroups = () => {
    setLoading(true)
    fetchGroups()
      .then(res => setGroups(res.items || []))
      .catch(err => {
        toast.error('加载分组失败', { description: err instanceof Error ? err.message : '未知错误' })
      })
      .finally(() => setLoading(false))
  }

  const loadModels = () => {
    fetchModels()
      .then(res => setModels(res.items || []))
      .catch(err => {
        toast.error('加载模型失败', { description: err instanceof Error ? err.message : '未知错误' })
      })
  }

  useEffect(() => { loadGroups(); loadModels() }, [])

  const openAdd = () => {
    setEditingGroup(null)
    setForm({ name: '', description: '', load_balance_strategy: 'round_robin' })
    setSelectedModelIds([])
    setShowModal(true)
  }

  const openEdit = (group: Group) => {
    setEditingGroup(group)
    setForm({
      name: group.name,
      description: group.description || '',
      load_balance_strategy: group.load_balance_strategy || 'round_robin',
    })
    setSelectedModelIds(group.model_ids || [])
    setShowModal(true)
  }

  const closeModal = () => {
    setShowModal(false)
    setEditingGroup(null)
    setSelectedModelIds([])
  }

  const handleSave = async () => {
    try {
      const data = { ...form, model_ids: selectedModelIds }
      if (editingGroup) {
        await updateGroup(editingGroup.id, data)
        toast.success('分组已更新')
      } else {
        await createGroup(data)
        toast.success('分组已创建')
      }
      closeModal()
      loadGroups()
    } catch (err) {
      toast.error('保存分组失败', { description: err instanceof Error ? err.message : '未知错误' })
    }
  }

  const handleDelete = async (id: string) => {
    if (!confirm('确定要删除此分组吗？')) return
    try {
      await deleteGroup(id)
      toast.success('分组已删除')
      loadGroups()
    } catch (err) {
      toast.error('删除分组失败', { description: err instanceof Error ? err.message : '未知错误' })
    }
  }

  const toggleModel = (modelId: string) => {
    setSelectedModelIds(prev =>
      prev.includes(modelId)
        ? prev.filter(id => id !== modelId)
        : [...prev, modelId]
    )
  }

  const getModelName = (modelId: string) => {
    return models.find(m => m.id === modelId)?.display_name || models.find(m => m.id === modelId)?.name || modelId
  }

  const filtered = groups.filter(g =>
    g.name.toLowerCase().includes(search.toLowerCase()) ||
    (g.description || '').toLowerCase().includes(search.toLowerCase())
  )

  // ─── Derived Metrics ──────────────────────────────────────────────────────

  const metrics = useMemo(() => {
    const totalModelsBound = groups.reduce((sum, g) => sum + (g.model_ids?.length || 0), 0)
    const strategiesUsed = new Set(groups.map(g => g.load_balance_strategy)).size
    const avgModelsPerGroup = groups.length > 0 ? (totalModelsBound / groups.length).toFixed(1) : '0'
    return { totalModelsBound, strategiesUsed, avgModelsPerGroup }
  }, [groups])

  // ─── Metrics Bar ──────────────────────────────────────────────────────────

  const metricsBar = [
    {
      label: '总分组',
      value: String(groups.length),
      icon: Users,
      bg: 'bg-blue-500/8',
      color: 'text-blue-600 dark:text-blue-400',
    },
    {
      label: '绑定模型',
      value: String(metrics.totalModelsBound),
      icon: Layers,
      bg: 'bg-emerald-500/8',
      color: 'text-emerald-600 dark:text-emerald-400',
    },
    {
      label: '策略类型',
      value: String(metrics.strategiesUsed),
      icon: Settings2,
      bg: 'bg-violet-500/8',
      color: 'text-violet-600 dark:text-violet-400',
    },
    {
      label: '平均模型',
      value: metrics.avgModelsPerGroup,
      icon: BarChart3,
      bg: 'bg-amber-500/8',
      color: 'text-amber-600 dark:text-amber-400',
    },
  ]

  if (loading) return <GroupsSkeleton />

  return (
    <div className="space-y-6">
      {/* ─── Header ─────────────────────────────────────────────────────── */}
      <div className="flex flex-col gap-4 sm:flex-row sm:items-center sm:justify-between">
        <div>
          <h1 className="text-2xl font-semibold tracking-tight">分组管理</h1>
          <p className="mt-1 text-sm text-muted-foreground">管理模型分组，控制 API Key 访问权限</p>
        </div>
        <div className="flex items-center gap-2">
          <Button variant="outline" size="sm" onClick={loadGroups}>
            <RefreshCw className="size-3.5" />
            刷新
          </Button>
          <Button size="sm" onClick={openAdd}>
            <Plus className="size-3.5" />
            添加分组
          </Button>
        </div>
      </div>

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

      {/* ─── Search Bar ─────────────────────────────────────────────────── */}
      <div className="relative">
        <Search className="absolute left-3.5 top-1/2 size-4 -translate-y-1/2 text-muted-foreground" />
        <Input
          placeholder="搜索分组名称或描述..."
          value={search}
          onChange={e => setSearch(e.target.value)}
          className="h-10 rounded-xl pl-10 border-border/60 bg-card focus-visible:ring-primary/20"
        />
        {search && (
          <button
            onClick={() => setSearch('')}
            className="absolute right-3 top-1/2 -translate-y-1/2 rounded-md p-0.5 text-muted-foreground hover:text-foreground transition-colors"
          >
            <X className="size-3.5" />
          </button>
        )}
      </div>

      {/* ─── Section Label ──────────────────────────────────────────────── */}
      <p className="text-xs font-semibold uppercase tracking-wider text-muted-foreground">
        分组列表 {search && <span className="text-muted-foreground/60">({filtered.length} 个结果)</span>}
      </p>

      {/* ─── Groups Grid ────────────────────────────────────────────────── */}
      {filtered.length > 0 ? (
        <div className="grid grid-cols-1 gap-4 md:grid-cols-2 lg:grid-cols-3">
          {filtered.map(group => {
            const strategy = strategyConfig[group.load_balance_strategy] || strategyConfig.round_robin
            const StrategyIcon = strategy.icon
            const modelCount = group.model_ids?.length || 0

            return (
              <BentoCard key={group.id} className="group/card relative flex flex-col transition-all hover:shadow-md">
                {/* Accent glow */}
                <div
                  className="pointer-events-none absolute -right-6 -top-6 h-20 w-20 rounded-full opacity-[0.06] blur-2xl"
                  style={{ background: strategy.color.includes('blue') ? '#3b82f6' : strategy.color.includes('violet') ? '#8b5cf6' : strategy.color.includes('emerald') ? '#10b981' : strategy.color.includes('amber') ? '#f59e0b' : '#06b6d4' }}
                />

                {/* Card Header */}
                <div className="relative z-10 flex items-start justify-between">
                  <div className="flex items-center gap-3 min-w-0 flex-1">
                    <div className={`flex h-11 w-11 items-center justify-center rounded-xl ${strategy.bg} shrink-0 transition-transform group-hover/card:scale-105`}>
                      <Users className={`size-5 ${strategy.color}`} />
                    </div>
                    <div className="min-w-0 flex-1">
                      <h3 className="text-sm font-semibold truncate text-foreground">{group.name}</h3>
                      <div className="flex items-center gap-1.5 mt-0.5">
                        <StrategyIcon className={`size-3 ${strategy.color}`} />
                        <span className={`text-[11px] font-medium ${strategy.color}`}>{strategy.label}</span>
                      </div>
                    </div>
                  </div>
                  <div className="flex items-center gap-0.5 shrink-0 opacity-0 group-hover/card:opacity-100 transition-opacity">
                    <Button variant="ghost" size="icon-sm" onClick={() => openEdit(group)} className="h-7 w-7">
                      <Pencil className="size-3.5" />
                    </Button>
                    <Button variant="ghost" size="icon-sm" onClick={() => handleDelete(group.id)} className="h-7 w-7 hover:text-destructive hover:bg-destructive/10">
                      <Trash2 className="size-3.5" />
                    </Button>
                  </div>
                </div>

                {/* Description */}
                {group.description && (
                  <p className="relative z-10 text-xs text-muted-foreground line-clamp-2 mt-2 leading-relaxed">
                    {group.description}
                  </p>
                )}

                {/* Models Section */}
                {modelCount > 0 && (
                  <div className="relative z-10 border-t border-border/60 pt-3 mt-auto">
                    <div className="flex items-center justify-between mb-2">
                      <p className="text-[11px] font-medium text-muted-foreground uppercase tracking-wider">绑定模型</p>
                      <Badge variant="secondary" className="text-[10px] font-bold tabular-nums">
                        {modelCount}
                      </Badge>
                    </div>
                    <div className="flex flex-wrap gap-1.5">
                      {group.model_ids!.slice(0, 3).map(modelId => (
                        <Badge key={modelId} variant="secondary" className="text-[10px] font-medium rounded-md">
                          {getModelName(modelId)}
                        </Badge>
                      ))}
                      {modelCount > 3 && (
                        <Badge variant="outline" className="text-[10px] font-medium rounded-md border-dashed">
                          +{modelCount - 3}
                        </Badge>
                      )}
                    </div>
                  </div>
                )}

                {/* Empty models hint */}
                {modelCount === 0 && (
                  <div className="relative z-10 border-t border-border/60 pt-3 mt-auto">
                    <p className="text-[11px] text-muted-foreground/60 italic">未绑定模型</p>
                  </div>
                )}
              </BentoCard>
            )
          })}
        </div>
      ) : (
        /* ─── Empty State ───────────────────────────────────────────────── */
        <BentoCard className="flex flex-col items-center justify-center py-16">
          <div className="flex h-16 w-16 items-center justify-center rounded-2xl bg-muted/50 mb-4">
            <Users className="size-8 text-muted-foreground/50" />
          </div>
          <p className="text-base font-medium text-foreground mb-1">
            {search ? '未找到匹配的分组' : '暂无分组'}
          </p>
          <p className="text-sm text-muted-foreground mb-4">
            {search ? '尝试使用不同的搜索词' : '创建第一个分组来开始管理模型访问权限'}
          </p>
          {!search && (
            <Button size="sm" onClick={openAdd}>
              <Plus className="size-3.5" />
              创建分组
            </Button>
          )}
        </BentoCard>
      )}

      {/* ─── Add/Edit Dialog ────────────────────────────────────────────── */}
      <Dialog open={showModal} onOpenChange={setShowModal}>
        <DialogContent
          className="max-w-2xl max-h-[85vh] overflow-y-auto"
          onPointerDownOutside={(e) => {
            const target = e.target as HTMLElement
            if (target.closest('[role="listbox"]') || target.closest('[role="option"]') || target.closest('[data-radix-select-viewport]')) {
              e.preventDefault()
            }
          }}
        >
          <DialogHeader>
            <DialogTitle className="text-lg">
              {editingGroup ? '编辑分组' : '添加分组'}
            </DialogTitle>
          </DialogHeader>
          <div className="space-y-5">
            {/* Name & Strategy */}
            <div className="grid grid-cols-2 gap-4">
              <div className="space-y-1.5">
                <label className="text-xs font-semibold uppercase tracking-wider text-muted-foreground">分组名称 *</label>
                <Input
                  placeholder="例如：production"
                  value={form.name}
                  onChange={e => setForm({...form, name: e.target.value})}
                  className="h-10 rounded-xl"
                />
              </div>
              <div className="space-y-1.5">
                <label className="text-xs font-semibold uppercase tracking-wider text-muted-foreground">负载均衡策略</label>
                <Select value={form.load_balance_strategy} onValueChange={v => setForm({...form, load_balance_strategy: v})}>
                  <SelectTrigger className="h-10 rounded-xl"><SelectValue /></SelectTrigger>
                  <SelectContent>
                    <SelectItem value="round_robin">
                      <div className="flex items-center gap-2">
                        <Shuffle className="size-3.5 text-blue-500" />
                        <span>轮询 (Round Robin)</span>
                      </div>
                    </SelectItem>
                    <SelectItem value="weighted">
                      <div className="flex items-center gap-2">
                        <BarChart3 className="size-3.5 text-violet-500" />
                        <span>加权 (Weighted)</span>
                      </div>
                    </SelectItem>
                    <SelectItem value="priority">
                      <div className="flex items-center gap-2">
                        <Target className="size-3.5 text-amber-500" />
                        <span>优先级 (Priority)</span>
                      </div>
                    </SelectItem>
                    <SelectItem value="least_connections">
                      <div className="flex items-center gap-2">
                        <GitBranch className="size-3.5 text-emerald-500" />
                        <span>最少连接</span>
                      </div>
                    </SelectItem>
                    <SelectItem value="adaptive">
                      <div className="flex items-center gap-2">
                        <Zap className="size-3.5 text-cyan-500" />
                        <span>自适应</span>
                      </div>
                    </SelectItem>
                  </SelectContent>
                </Select>
              </div>
            </div>

            {/* Description */}
            <div className="space-y-1.5">
              <label className="text-xs font-semibold uppercase tracking-wider text-muted-foreground">描述</label>
              <Input
                placeholder="分组描述（可选）"
                value={form.description}
                onChange={e => setForm({...form, description: e.target.value})}
                className="h-10 rounded-xl"
              />
            </div>

            {/* Models Section */}
            <div className="border-t border-border/60 pt-5">
              <div className="mb-3">
                <div className="flex items-center justify-between">
                  <label className="text-xs font-semibold uppercase tracking-wider text-muted-foreground">绑定模型</label>
                  {selectedModelIds.length > 0 && (
                    <Badge variant="secondary" className="text-[10px] font-bold">
                      已选 {selectedModelIds.length}
                    </Badge>
                  )}
                </div>
                <p className="text-xs text-muted-foreground mt-1">选择此分组可以使用的模型，API Key 将只能访问这些模型</p>
              </div>
              {models.length === 0 ? (
                <div className="flex items-center gap-3 p-4 rounded-xl border border-amber-200 bg-amber-50 dark:border-amber-800 dark:bg-amber-950/20">
                  <div className="flex h-9 w-9 shrink-0 items-center justify-center rounded-lg bg-amber-500/10">
                    <Layers className="size-4 text-amber-600 dark:text-amber-400" />
                  </div>
                  <p className="text-xs font-medium text-amber-700 dark:text-amber-300">
                    暂无可用模型，请先在模型管理中添加模型
                  </p>
                </div>
              ) : (
                <div className="grid grid-cols-1 gap-2 max-h-[300px] overflow-y-auto pr-1">
                  {models.map(model => {
                    const isSelected = selectedModelIds.includes(model.id)
                    return (
                      <div
                        key={model.id}
                        onClick={() => toggleModel(model.id)}
                        className={`flex items-center gap-3 p-3 rounded-xl border cursor-pointer transition-all ${
                          isSelected
                            ? 'bg-primary/5 border-primary/30 shadow-sm'
                            : 'bg-card border-border/60 hover:border-border hover:bg-muted/30'
                        }`}
                      >
                        <div className={`flex h-5 w-5 items-center justify-center rounded-md shrink-0 transition-all ${
                          isSelected
                            ? 'bg-primary text-primary-foreground shadow-sm'
                            : 'border border-muted-foreground/20'
                        }`}>
                          {isSelected && <Check className="size-3" />}
                        </div>
                        <div className="min-w-0 flex-1">
                          <p className="text-sm font-medium truncate">{model.display_name || model.name}</p>
                          <p className="text-[11px] text-muted-foreground truncate">{model.name}</p>
                        </div>
                        <Badge
                          variant="secondary"
                          className="text-[10px] capitalize shrink-0 font-medium rounded-md"
                        >
                          {model.model_type}
                        </Badge>
                      </div>
                    )
                  })}
                </div>
              )}
            </div>
          </div>
          <DialogFooter className="mt-4 pt-4 border-t border-border/60">
            <Button variant="outline" onClick={closeModal} className="rounded-xl">取消</Button>
            <Button onClick={handleSave} disabled={!form.name} className="rounded-xl">
              {editingGroup ? '保存更改' : '创建分组'}
            </Button>
          </DialogFooter>
        </DialogContent>
      </Dialog>
    </div>
  )
}
