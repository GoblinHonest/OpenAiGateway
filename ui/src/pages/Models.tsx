import React, { useEffect, useState } from 'react'
import { toast } from 'sonner'
import { Plus, Trash2, RefreshCw, Search, Cpu, Link2, X, GripVertical, Pencil, ChevronUp, ChevronDown, Brain, Mic, ImageIcon, MessageSquare, Boxes, Settings2 } from 'lucide-react'
import { fetchModels, createModelWithBindings, deleteModel, removeBinding, updateModel, bindProvider, Model } from '../api/model'
import { fetchProviders, fetchProviderModels, Provider } from '../api/provider'
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

interface ProviderBinding {
  id?: string                 // 已有绑定的 ID（编辑模式）
  provider_id: string
  upstream_model_name: string
  weight: number
  priority: number
  _removed?: boolean          // 编辑模式标记删除
}

// ─── Model type config ─────────────────────────────────────────────────────

const MODEL_TYPE_CONFIG: Record<string, { icon: React.ElementType; bg: string; color: string; label: string }> = {
  chat: { icon: MessageSquare, bg: 'bg-blue-500/8', color: 'text-blue-600 dark:text-blue-400', label: 'Chat' },
  embeddings: { icon: Brain, bg: 'bg-violet-500/8', color: 'text-violet-600 dark:text-violet-400', label: 'Embeddings' },
  image: { icon: ImageIcon, bg: 'bg-pink-500/8', color: 'text-pink-600 dark:text-pink-400', label: 'Image' },
  audio: { icon: Mic, bg: 'bg-amber-500/8', color: 'text-amber-600 dark:text-amber-400', label: 'Audio' },
}

function getModelTypeConfig(type?: string) {
  return MODEL_TYPE_CONFIG[type || 'chat'] || MODEL_TYPE_CONFIG.chat
}

// ─── Helpers ────────────────────────────────────────────────────────────────

function formatContextWindow(ctx: number): string {
  if (ctx >= 1_000_000) return (ctx / 1_000_000).toFixed(1) + 'M'
  if (ctx >= 1_000) return (ctx / 1_000).toFixed(0) + 'K'
  return String(ctx)
}

// ─── Main Component ─────────────────────────────────────────────────────────

export default function Models() {
  const [models, setModels] = useState<Model[]>([])
  const [providers, setProviders] = useState<Provider[]>([])
  const [loading, setLoading] = useState(true)
  const [search, setSearch] = useState('')
  const [showModal, setShowModal] = useState(false)
  const [editingModel, setEditingModel] = useState<Model | null>(null)
  const [form, setForm] = useState({ name: '', display_name: '', description: '', model_type: 'chat' })
  const [bindings, setBindings] = useState<ProviderBinding[]>([])
  const [providerModels, setProviderModels] = useState<Record<string, string[]>>({})
  const [loadingModels, setLoadingModels] = useState<Record<string, boolean>>({})

  const loadModels = () => {
    setLoading(true)
    fetchModels()
      .then(res => setModels(res.items || []))
      .catch(err => {
        toast.error('加载模型失败', { description: err instanceof Error ? err.message : '未知错误' })
      })
      .finally(() => setLoading(false))
  }

  const loadProviders = () => {
    fetchProviders()
      .then(res => setProviders(res.items || []))
      .catch(err => {
        toast.error('加载服务商失败', { description: err instanceof Error ? err.message : '未知错误' })
      })
  }

  useEffect(() => { loadModels(); loadProviders() }, [])

  const openAdd = () => {
    setEditingModel(null)
    setForm({ name: '', display_name: '', description: '', model_type: 'chat' })
    setBindings([])
    setShowModal(true)
  }

  const fetchModelsForProvider = (pid: string) => {
    if (!pid || providerModels[pid] || loadingModels[pid]) return
    setLoadingModels(prev => ({ ...prev, [pid]: true }))
    fetchProviderModels(pid)
      .then(m => setProviderModels(prev => ({ ...prev, [pid]: m })))
      .catch(() => {})
      .finally(() => setLoadingModels(prev => ({ ...prev, [pid]: false })))
  }

  const openEdit = (model: Model) => {
    setEditingModel(model)
    setForm({
      name: model.name,
      display_name: model.display_name || '',
      description: model.description || '',
      model_type: model.model_type || 'chat',
    })
    const mappedBindings = (model.bindings || []).map(b => ({
      id: b.id,
      provider_id: b.provider_id,
      upstream_model_name: b.upstream_model_name || '',
      weight: b.weight,
      priority: b.priority,
    }))
    setBindings(mappedBindings)
    // 预加载已绑定服务商的模型列表
    mappedBindings.forEach(b => fetchModelsForProvider(b.provider_id))
    setShowModal(true)
  }

  const closeModal = () => {
    setShowModal(false)
    setBindings([])
    setEditingModel(null)
  }

  const handleSave = async () => {
    try {
      if (editingModel) {
        // 更新模型基本信息
        await updateModel(editingModel.id, form)
        // 处理绑定：删除标记的，添加新的
        for (const b of bindings) {
          if (b._removed && b.id) {
            try { await removeBinding(b.id!) } catch { /* ignore */ }
          } else if (!b.id && b.provider_id) {
            try { await bindProvider(editingModel.id!, b.provider_id, b.upstream_model_name, b.weight, b.priority) } catch { /* ignore */ }
          }
        }
        toast.success('模型已更新')
      } else {
        await createModelWithBindings(form, bindings)
        toast.success('模型已创建')
      }
      closeModal()
      loadModels()
    } catch (err) {
      toast.error('保存模型失败', { description: err instanceof Error ? err.message : '未知错误' })
    }
  }

  const handleDelete = async (id: string) => {
    if (!confirm('确定要删除此模型吗？')) return
    try {
      await deleteModel(id)
      toast.success('模型已删除')
      loadModels()
    } catch (err) {
      toast.error('删除模型失败', { description: err instanceof Error ? err.message : '未知错误' })
    }
  }

  const handleRemoveBinding = async (bindingId: string) => {
    if (!confirm('确定要移除此绑定吗？')) return
    try {
      await removeBinding(bindingId)
      toast.success('绑定已移除')
      loadModels()
    } catch (err) {
      toast.error('移除绑定失败', { description: err instanceof Error ? err.message : '未知错误' })
    }
  }

  const addBinding = () => {
    const first = providers.find(p => p.status === 'active')
    if (!first) return
    setBindings([...bindings, { provider_id: first.id, upstream_model_name: '', weight: 1, priority: bindings.length }])
  }

  const markBindingRemoved = (i: number) => {
    const b = bindings[i]
    if (b.id) {
      // 编辑模式下标记删除
      const updated = [...bindings]
      updated[i] = { ...b, _removed: true }
      setBindings(updated)
    } else {
      setBindings(bindings.filter((_, j) => j !== i))
    }
  }

  const updateBinding = (i: number, field: keyof ProviderBinding, value: string | number) => {
    setBindings(prev => {
      const updated = [...prev]
      updated[i] = { ...updated[i], [field]: value }
      return updated
    })
    if (field === 'provider_id' && typeof value === 'string' && value) {
      fetchModelsForProvider(value)
    }
  }

  const getProviderModels = (pid: string): string[] => providerModels[pid] || providers.find(p => p.id === pid)?.models || []

  const filtered = models.filter(m => m.name.toLowerCase().includes(search.toLowerCase()) || (m.display_name || '').toLowerCase().includes(search.toLowerCase()))
  const activeProviders = providers.filter(p => p.status === 'active' && p.id)
  const visibleBindings = bindings.filter(b => !b._removed)

  // ─── Derived stats ──────────────────────────────────────────────────────
  const totalModels = models.length
  const chatModels = models.filter(m => m.model_type === 'chat').length
  const totalBindings = models.reduce((sum, m) => sum + (m.bindings?.length || 0), 0)
  const typesActive = new Set(models.map(m => m.model_type || 'chat')).size

  // ─── Loading skeleton ───────────────────────────────────────────────────

  if (loading) {
    return (
      <div className="space-y-6">
        <div className="flex items-center justify-between">
          <div>
            <Skeleton className="h-8 w-32" />
            <Skeleton className="mt-2 h-4 w-48" />
          </div>
          <div className="flex gap-2">
            <Skeleton className="h-9 w-20" />
            <Skeleton className="h-9 w-28" />
          </div>
        </div>
        <Skeleton className="h-10 w-full rounded-xl" />
        {/* Summary bar skeleton */}
        <div className="grid grid-cols-2 gap-2.5 sm:grid-cols-4">
          {Array.from({ length: 4 }).map((_, i) => (
            <Skeleton key={i} className="h-[60px] rounded-xl" />
          ))}
        </div>
        {/* Cards skeleton */}
        <p className="mb-3 text-xs font-semibold uppercase tracking-wider text-muted-foreground">
          模型列表
        </p>
        <div className="grid grid-cols-1 gap-4 md:grid-cols-2 lg:grid-cols-3">
          {Array.from({ length: 6 }).map((_, i) => (
            <Skeleton key={i} className="h-[180px] rounded-2xl" />
          ))}
        </div>
      </div>
    )
  }

  // ─── Metrics bar items ──────────────────────────────────────────────────

  const metricsBar = [
    {
      label: '总模型数',
      value: String(totalModels),
      icon: Boxes,
      bg: 'bg-blue-500/8',
      color: 'text-blue-600 dark:text-blue-400',
    },
    {
      label: 'Chat 模型',
      value: String(chatModels),
      icon: MessageSquare,
      bg: 'bg-emerald-500/8',
      color: 'text-emerald-600 dark:text-emerald-400',
    },
    {
      label: '绑定数',
      value: String(totalBindings),
      icon: Link2,
      bg: 'bg-violet-500/8',
      color: 'text-violet-600 dark:text-violet-400',
    },
    {
      label: '类型数',
      value: String(typesActive),
      icon: Settings2,
      bg: 'bg-amber-500/8',
      color: 'text-amber-600 dark:text-amber-400',
    },
  ]

  return (
    <div className="space-y-6">
      {/* ─── Header ─────────────────────────────────────────────────────── */}
      <PageHeader title="模型管理" description="管理可用的 AI 模型，配置多服务商负载均衡">
        <Button variant="outline" size="sm" onClick={loadModels}>
          <RefreshCw className="size-3.5" />
          刷新
        </Button>
        <Button size="sm" onClick={openAdd}>
          <Plus className="size-3.5" />
          添加模型
        </Button>
      </PageHeader>

      {/* ─── Search Bar ─────────────────────────────────────────────────── */}
      <div className="relative">
        <Search className="absolute left-3 top-1/2 size-4 -translate-y-1/2 text-muted-foreground" />
        <Input
          placeholder="搜索模型名称或显示名称..."
          value={search}
          onChange={e => setSearch(e.target.value)}
          className="h-10 rounded-xl pl-9"
        />
      </div>

      {/* ─── Summary Metrics Bar ────────────────────────────────────────── */}
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

      {/* ─── Models Grid ────────────────────────────────────────────────── */}
      <div>
        <p className="mb-3 text-xs font-semibold uppercase tracking-wider text-muted-foreground">
          模型列表
          {search && <span className="ml-2 normal-case tracking-normal text-muted-foreground/60">（{filtered.length} 个结果）</span>}
        </p>

        {filtered.length > 0 ? (
          <div className="grid grid-cols-1 gap-4 md:grid-cols-2 lg:grid-cols-3">
            {filtered.map(model => {
              const typeConfig = getModelTypeConfig(model.model_type)
              const TypeIcon = typeConfig.icon
              return (
                <BentoCard key={model.id} compact className="group transition-all hover:border-border hover:shadow-md">
                  {/* Card Header */}
                  <div className="flex items-start justify-between gap-2">
                    <div className="flex items-center gap-3 min-w-0 flex-1">
                      <div className={`flex h-10 w-10 shrink-0 items-center justify-center rounded-xl ${typeConfig.bg}`}>
                        <TypeIcon className={`size-5 ${typeConfig.color}`} />
                      </div>
                      <div className="min-w-0 flex-1">
                        <div className="flex items-center gap-2">
                          <h3 className="text-sm font-semibold truncate">{model.display_name || model.name}</h3>
                        </div>
                        <p className="text-xs text-muted-foreground truncate font-mono">{model.name}</p>
                      </div>
                    </div>
                    <div className="flex items-center gap-0.5 shrink-0 opacity-0 transition-opacity group-hover:opacity-100">
                      <Button variant="ghost" size="icon-sm" onClick={() => openEdit(model)}>
                        <Pencil className="size-3.5" />
                      </Button>
                      <Button variant="ghost" size="icon-sm" onClick={() => handleDelete(model.id)}>
                        <Trash2 className="size-3.5 text-destructive" />
                      </Button>
                    </div>
                  </div>

                  {/* Meta row */}
                  <div className="flex flex-wrap items-center gap-1.5 mt-1">
                    <Badge
                      variant="secondary"
                      className={`text-[10px] capitalize ${typeConfig.bg} ${typeConfig.color} border-0`}
                    >
                      {model.model_type}
                    </Badge>
                    {model.context_window > 0 && (
                      <Badge variant="outline" className="text-[10px] px-1.5 py-0">
                        {formatContextWindow(model.context_window)} ctx
                      </Badge>
                    )}
                    {model.bindings && model.bindings.length > 0 && (
                      <Badge variant="outline" className="text-[10px] px-1.5 py-0">
                        {model.bindings.length} 绑定
                      </Badge>
                    )}
                  </div>

                  {/* Description */}
                  {model.description && (
                    <p className="text-xs text-muted-foreground line-clamp-2 leading-relaxed">{model.description}</p>
                  )}

                  {/* Provider bindings */}
                  {model.bindings && model.bindings.length > 0 && (
                    <div className="border-t border-border/60 pt-2.5 mt-1">
                      <p className="text-[10px] font-semibold uppercase tracking-wider text-muted-foreground mb-2">
                        负载均衡
                      </p>
                      <div className="space-y-1.5">
                        {model.bindings.map(b => (
                          <div
                            key={b.id}
                            className="flex items-center justify-between gap-1.5 text-xs rounded-lg bg-muted/40 px-2.5 py-1.5 transition-colors hover:bg-muted/60"
                          >
                            <div className="flex items-center gap-1.5 min-w-0 flex-1">
                              <Link2 className="size-3 text-muted-foreground shrink-0" />
                              <span className="truncate font-medium">{b.provider_name || b.provider_id}</span>
                              {b.upstream_model_name && (
                                <span className="text-[10px] text-muted-foreground truncate">→ {b.upstream_model_name}</span>
                              )}
                            </div>
                            <Badge variant="outline" className="text-[10px] px-1.5 py-0 shrink-0 tabular-nums">
                              w{b.weight}
                            </Badge>
                          </div>
                        ))}
                      </div>
                    </div>
                  )}
                </BentoCard>
              )
            })}
          </div>
        ) : (
          /* ─── Empty State ─────────────────────────────────────────────── */
          <BentoCard className="flex flex-col items-center justify-center py-16">
            <div className="flex h-16 w-16 items-center justify-center rounded-2xl bg-muted/60 mb-4">
              <Cpu className="size-8 text-muted-foreground/60" />
            </div>
            <p className="text-sm font-medium text-foreground mb-1">
              {search ? '未找到匹配的模型' : '暂无模型'}
            </p>
            <p className="text-xs text-muted-foreground mb-4">
              {search ? '尝试其他搜索关键词' : '点击上方按钮添加第一个 AI 模型'}
            </p>
            {!search && (
              <Button size="sm" variant="outline" onClick={openAdd}>
                <Plus className="size-3.5 mr-1" />
                添加模型
              </Button>
            )}
          </BentoCard>
        )}
      </div>

      {/* ─── Add / Edit Dialog ──────────────────────────────────────────── */}
      <Dialog open={showModal} onOpenChange={setShowModal}>
        <DialogContent
          className="max-w-2xl max-h-[85vh] overflow-y-auto"
          onPointerDownOutside={(e) => {
            // 防止点击 Select 下拉框时关闭弹窗
            const target = e.target as HTMLElement
            if (target.closest('[role="listbox"]') || target.closest('[role="option"]') || target.closest('[data-radix-select-viewport]')) {
              e.preventDefault()
            }
          }}
        >
          <DialogHeader>
            <DialogTitle className="text-lg">
              {editingModel ? '编辑模型' : '添加模型'}
            </DialogTitle>
          </DialogHeader>
          <div className="space-y-5">
            {/* Basic info */}
            <div>
              <p className="text-[10px] font-semibold uppercase tracking-wider text-muted-foreground mb-3">
                基本信息
              </p>
              <div className="grid grid-cols-2 gap-3">
                <div className="space-y-1.5">
                  <label className="text-sm font-medium">模型名称 *</label>
                  <Input placeholder="gpt-4" value={form.name} onChange={e => setForm({...form, name: e.target.value})} />
                </div>
                <div className="space-y-1.5">
                  <label className="text-sm font-medium">显示名称</label>
                  <Input placeholder="GPT-4" value={form.display_name} onChange={e => setForm({...form, display_name: e.target.value})} />
                </div>
              </div>
              <div className="grid grid-cols-2 gap-3 mt-3">
                <div className="space-y-1.5">
                  <label className="text-sm font-medium">描述</label>
                  <Input placeholder="模型描述" value={form.description} onChange={e => setForm({...form, description: e.target.value})} />
                </div>
                <div className="space-y-1.5">
                  <label className="text-sm font-medium">模型类型</label>
                  <Select value={form.model_type} onValueChange={v => setForm({...form, model_type: v})}>
                    <SelectTrigger><SelectValue /></SelectTrigger>
                    <SelectContent>
                      <SelectItem value="chat">Chat</SelectItem>
                      <SelectItem value="embeddings">Embeddings</SelectItem>
                      <SelectItem value="image">Image</SelectItem>
                      <SelectItem value="audio">Audio</SelectItem>
                    </SelectContent>
                  </Select>
                </div>
              </div>
            </div>

            {/* Provider bindings */}
            <div className="border-t border-border/60 pt-4">
              <div className="flex items-center justify-between mb-3">
                <div>
                  <p className="text-[10px] font-semibold uppercase tracking-wider text-muted-foreground mb-0.5">
                    服务商绑定
                  </p>
                  <p className="text-xs text-muted-foreground">绑定多个服务商实现负载均衡</p>
                </div>
                <Button variant="outline" size="sm" onClick={addBinding} disabled={activeProviders.length === 0}>
                  <Plus className="size-3 mr-1" />
                  添加服务商
                </Button>
              </div>
              {activeProviders.length === 0 && (
                <p className="text-xs text-amber-600 dark:text-amber-400 text-center py-3.5 rounded-xl border border-amber-200 bg-amber-50 dark:border-amber-800 dark:bg-amber-950/30">
                  暂无可用的服务商
                </p>
              )}
              {activeProviders.length > 0 && visibleBindings.length === 0 && (
                <p className="text-xs text-muted-foreground text-center py-6 rounded-xl border border-dashed">
                  尚未绑定服务商，点击上方按钮添加
                </p>
              )}
              <div className="space-y-2">
                {/* Table header */}
                {visibleBindings.length > 0 && (
                  <div className="flex items-center gap-1.5 px-2 text-[10px] font-semibold uppercase tracking-wider text-muted-foreground">
                    <div className="w-6 shrink-0"></div>
                    <div className="flex-1 flex items-center gap-2">
                      <div className="w-[140px]">服务商</div>
                      <div className="flex-1">上游模型</div>
                    </div>
                    <div className="w-20 shrink-0 text-center">权重</div>
                    <div className="w-20 shrink-0 text-center">优先级</div>
                    <div className="w-8 shrink-0"></div>
                  </div>
                )}
                {bindings.map((b, i) => {
                  if (b._removed) return null
                  const pmodels = getProviderModels(b.provider_id)
                  return (
                  <div key={i} className={`flex items-center gap-1.5 p-2.5 rounded-xl border transition-colors ${b._removed ? 'opacity-40 line-through' : 'bg-muted/20 hover:bg-muted/30'}`}>
                    <div className="flex flex-col shrink-0 gap-0.5">
                      <button
                        onClick={() => { if (i > 0) { const u = [...bindings]; [u[i-1], u[i]] = [u[i], u[i-1]]; setBindings(u) } }}
                        className="text-muted-foreground hover:text-foreground rounded transition-colors hover:bg-muted/50"
                      >
                        <ChevronUp className="size-3" />
                      </button>
                      <button
                        onClick={() => { if (i < bindings.length - 1) { const u = [...bindings]; [u[i], u[i+1]] = [u[i+1], u[i]]; setBindings(u) } }}
                        className="text-muted-foreground hover:text-foreground rounded transition-colors hover:bg-muted/50"
                      >
                        <ChevronDown className="size-3" />
                      </button>
                    </div>
                    <div className="flex-1 min-w-0 flex items-center gap-2">
                      <Select value={b.provider_id} onValueChange={v => updateBinding(i, 'provider_id', v)}>
                        <SelectTrigger className="w-[140px]"><SelectValue placeholder="服务商..." /></SelectTrigger>
                        <SelectContent>{activeProviders.map(p => <SelectItem key={p.id} value={p.id}>{p.name}</SelectItem>)}</SelectContent>
                      </Select>
                      {pmodels.length > 0 ? (
                        <Select value={b.upstream_model_name} onValueChange={v => updateBinding(i, 'upstream_model_name', v)}>
                          <SelectTrigger className="flex-1"><SelectValue placeholder="选择模型" /></SelectTrigger>
                          <SelectContent>
                            {pmodels.map(m => <SelectItem key={m} value={m}>{m}</SelectItem>)}
                          </SelectContent>
                        </Select>
                      ) : (
                        <div className="relative flex-1">
                          <Input
                            placeholder={loadingModels[b.provider_id] ? '加载中...' : '输入模型名称'}
                            value={b.upstream_model_name}
                            onChange={e => updateBinding(i, 'upstream_model_name', e.target.value)}
                            className="h-9 text-sm"
                            disabled={loadingModels[b.provider_id]}
                          />
                        </div>
                      )}
                    </div>
                    <div className="w-20 shrink-0">
                      <Select value={String(b.weight)} onValueChange={v => updateBinding(i, 'weight', parseInt(v))}>
                        <SelectTrigger className="h-9 text-sm"><SelectValue /></SelectTrigger>
                        <SelectContent>
                          <SelectItem value="1">1 (低)</SelectItem>
                          <SelectItem value="5">5 (中)</SelectItem>
                          <SelectItem value="10">10 (高)</SelectItem>
                          <SelectItem value="20">20 (最高)</SelectItem>
                        </SelectContent>
                      </Select>
                    </div>
                    <div className="w-20 shrink-0">
                      <Select value={String(b.priority)} onValueChange={v => updateBinding(i, 'priority', parseInt(v))}>
                        <SelectTrigger className="h-9 text-sm"><SelectValue /></SelectTrigger>
                        <SelectContent>
                          <SelectItem value="0">0 (默认)</SelectItem>
                          <SelectItem value="1">1 (备选)</SelectItem>
                          <SelectItem value="2">2 (备用)</SelectItem>
                          <SelectItem value="3">3 (最低)</SelectItem>
                        </SelectContent>
                      </Select>
                    </div>
                    <button
                      onClick={() => markBindingRemoved(i)}
                      className="shrink-0 p-1.5 rounded-lg text-muted-foreground hover:text-destructive hover:bg-destructive/10 transition-colors"
                    >
                      <X className="size-4" />
                    </button>
                  </div>
                )})}
                {visibleBindings.length > 0 && (
                  <p className="text-[10px] text-muted-foreground px-2 mt-1 leading-relaxed">
                    权重：数值越大，分配的请求越多。优先级：数值越小，优先使用（0 最优先）。
                  </p>
                )}
              </div>
            </div>
          </div>
          <DialogFooter className="mt-4 pt-4 border-t border-border/60">
            <Button variant="outline" onClick={closeModal}>取消</Button>
            <Button onClick={handleSave} disabled={!form.name}>
              {editingModel ? '保存更改' : '创建模型'}
            </Button>
          </DialogFooter>
        </DialogContent>
      </Dialog>
    </div>
  )
}
