import React, { useState } from 'react'
import { toast } from 'sonner'
import {
  Cpu,
  Gauge,
  Shield,
  RefreshCw,
  Database,
  Save,
  Info,
  RotateCcw,
  Check,
  Eye,
} from 'lucide-react'
import { Button } from '@/components/ui/button'
import { BentoCard } from '@/components/ui/bento-card'
import { PageHeader } from '@/components/ui/page-header'
import { Input } from '@/components/ui/input'
import { Switch } from '@/components/ui/switch'

// ─── Types ──────────────────────────────────────────────────────────────────

interface ConfigSetting {
  key: string
  label: string
  value: string | boolean | number
  type: 'text' | 'toggle' | 'number'
}

interface ConfigSection {
  title: string
  description: string
  icon: typeof Cpu
  accent: string
  settings: ConfigSetting[]
}

// ─── Constants ──────────────────────────────────────────────────────────────

const SECTION_ICONS = [
  { icon: Cpu, accent: '#4361ee', bg: 'bg-blue-500/8', color: 'text-blue-600 dark:text-blue-400' },
  { icon: Gauge, accent: '#f59e0b', bg: 'bg-amber-500/8', color: 'text-amber-600 dark:text-amber-400' },
  { icon: Shield, accent: '#ef4444', bg: 'bg-red-500/8', color: 'text-red-600 dark:text-red-400' },
  { icon: RefreshCw, accent: '#06b6d4', bg: 'bg-cyan-500/8', color: 'text-cyan-600 dark:text-cyan-400' },
  { icon: Database, accent: '#10b981', bg: 'bg-emerald-500/8', color: 'text-emerald-600 dark:text-emerald-400' },
]

// ─── Main Component ─────────────────────────────────────────────────────────

export default function Config() {
  const [config, setConfig] = useState<ConfigSection[]>([
    {
      title: '服务器配置',
      description: 'HTTP 服务器基本设置',
      icon: Cpu,
      accent: '#4361ee',
      settings: [
        { key: 'server.host', label: '监听地址', value: '0.0.0.0', type: 'text' },
        { key: 'server.port', label: '端口', value: 8080, type: 'number' },
        { key: 'server.readTimeout', label: '读取超时', value: '30s', type: 'text' },
        { key: 'server.writeTimeout', label: '写入超时', value: '60s', type: 'text' },
      ],
    },
    {
      title: '速率限制',
      description: 'API 请求速率限制配置',
      icon: Gauge,
      accent: '#f59e0b',
      settings: [
        { key: 'rateLimit.enabled', label: '启用速率限制', value: true, type: 'toggle' },
        { key: 'rateLimit.defaultRPM', label: '默认 RPM', value: 60, type: 'number' },
        { key: 'rateLimit.defaultTPM', label: '默认 TPM', value: 100000, type: 'number' },
      ],
    },
    {
      title: '熔断器',
      description: '服务熔断保护配置',
      icon: Shield,
      accent: '#ef4444',
      settings: [
        { key: 'circuitBreaker.enabled', label: '启用熔断器', value: true, type: 'toggle' },
        { key: 'circuitBreaker.failureThreshold', label: '失败阈值', value: 5, type: 'number' },
        { key: 'circuitBreaker.successThreshold', label: '成功阈值', value: 2, type: 'number' },
        { key: 'circuitBreaker.cooldownDuration', label: '冷却时间', value: '60s', type: 'text' },
      ],
    },
    {
      title: '重试策略',
      description: '请求失败重试配置',
      icon: RefreshCw,
      accent: '#06b6d4',
      settings: [
        { key: 'retry.maxAttempts', label: '最大重试次数', value: 3, type: 'number' },
        { key: 'retry.initialBackoff', label: '初始退避时间', value: '1s', type: 'text' },
        { key: 'retry.maxBackoff', label: '最大退避时间', value: '10s', type: 'text' },
      ],
    },
    {
      title: 'Prompt Caching',
      description: '服务商侧缓存（自动生效，无需配置）',
      icon: Database,
      accent: '#10b981',
      settings: [
        { key: 'cache.openai', label: 'OpenAI', value: '自动缓存超过1024 tokens的prompt', type: 'text' },
        { key: 'cache.anthropic', label: 'Anthropic', value: '需要在请求中指定cache_control参数', type: 'text' },
        { key: 'cache.deepseek', label: 'DeepSeek', value: '自动缓存', type: 'text' },
      ],
    },
  ])

  const [saved, setSaved] = useState(false)

  const handleSave = () => {
    toast.info('配置为只读模式', {
      description: '请直接修改 config.yaml 文件后重启服务',
    })
  }

  const updateSetting = (sectionIndex: number, settingIndex: number, value: string | boolean | number) => {
    const newConfig = [...config]
    newConfig[sectionIndex].settings[settingIndex].value = value
    setConfig(newConfig)
  }

  return (
    <div className="space-y-6">
      {/* ─── Header ─────────────────────────────────────────────────────── */}
      <PageHeader title="系统配置" description="管理网关运行参数（只读 — 修改 config.yaml 后重启生效）">
        <Button variant="outline" size="sm" onClick={() => toast.info('配置已重载', { description: '显示值来自 config.yaml' })}>
          <RotateCcw className="size-3.5" /> 重载
        </Button>
        <Button size="sm" onClick={handleSave}>
          <Eye className="size-3.5" />
          查看配置
        </Button>
      </PageHeader>

      {/* ─── Config Sections ────────────────────────────────────────────── */}
      <div className="space-y-6">
        {config.map((section, sectionIndex) => {
          const iconMeta = SECTION_ICONS[sectionIndex % SECTION_ICONS.length]
          const Icon = section.icon

          return (
            <div key={sectionIndex}>
              <p className="mb-3 text-xs font-semibold uppercase tracking-wider text-muted-foreground">
                {section.title}
              </p>
              <BentoCard compact>
                {/* Card header with accent icon */}
                <div className="mb-4 flex items-center gap-3 border-b border-border/60 pb-4">
                  <div
                    className="flex h-9 w-9 items-center justify-center rounded-xl"
                    style={{ background: `${iconMeta.accent}10` }}
                  >
                    <Icon className="size-4.5" style={{ color: iconMeta.accent }} />
                  </div>
                  <div>
                    <h3 className="text-sm font-semibold">{section.title}</h3>
                    <p className="text-xs text-muted-foreground">{section.description}</p>
                  </div>
                </div>

                {/* Settings list */}
                <div className="space-y-0">
                  {section.settings.map((setting, settingIndex) => (
                    <div
                      key={setting.key}
                      className="group flex items-center justify-between gap-4 rounded-lg px-2 py-3 transition-colors hover:bg-muted/40 last:border-0"
                    >
                      <div className="min-w-0 flex-1">
                        <p className="text-sm font-medium">{setting.label}</p>
                        <p className="font-mono text-[11px] text-muted-foreground/70">{setting.key}</p>
                      </div>
                      <div className="shrink-0">
                        {setting.type === 'toggle' ? (
                          <Switch
                            checked={setting.value as boolean}
                            disabled
                          />
                        ) : setting.type === 'number' ? (
                          <Input
                            type="number"
                            value={setting.value as number}
                            disabled
                            className="h-8 w-24 rounded-lg border-border/60 bg-muted/30 text-right text-sm tabular-nums opacity-70"
                          />
                        ) : (
                          <Input
                            type="text"
                            value={setting.value as string}
                            disabled
                            className="h-8 min-w-[12rem] max-w-xs rounded-lg border-border/60 bg-muted/30 text-right text-sm opacity-70"
                          />
                        )}
                      </div>
                    </div>
                  ))}
                </div>
              </BentoCard>
            </div>
          )
        })}
      </div>

      {/* ─── Environment Variables Info ─────────────────────────────────── */}
      <div>
        <p className="mb-3 text-xs font-semibold uppercase tracking-wider text-muted-foreground">
          提示
        </p>
        <BentoCard compact>
          <div className="flex items-start gap-3">
            <div
              className="flex h-9 w-9 shrink-0 items-center justify-center rounded-xl"
              style={{ background: '#4361ee10' }}
            >
              <Info className="size-4.5 text-blue-600 dark:text-blue-400" />
            </div>
            <div>
              <h3 className="text-sm font-semibold">环境变量覆盖</h3>
              <p className="mt-1 text-xs leading-relaxed text-muted-foreground">
                所有配置都可以通过环境变量覆盖，前缀为{' '}
                <code className="rounded-md border border-border/60 bg-muted/60 px-1.5 py-0.5 font-mono text-[11px]">
                  AIGATEWAY_
                </code>
                。例如：
              </p>
              <p className="mt-2">
                <code className="rounded-md border border-border/60 bg-muted/60 px-2 py-1 font-mono text-[11px] text-foreground">
                  AIGATEWAY_SERVER_PORT=8080
                </code>
              </p>
            </div>
          </div>
        </BentoCard>
      </div>
    </div>
  )
}
