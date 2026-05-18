import React, { useState } from 'react'
import { setAdminToken } from '../api/auth'
import { Button } from '@/components/ui/button'
import { Input } from '@/components/ui/input'
import { BentoCard } from '@/components/ui/bento-card'

interface LoginProps {
  onLogin: () => void
}

const Login: React.FC<LoginProps> = ({ onLogin }) => {
  const [token, setToken] = useState('')
  const [error, setError] = useState('')
  const [loading, setLoading] = useState(false)

  const handleSubmit = async (e: React.FormEvent) => {
    e.preventDefault()
    if (!token.trim()) {
      setError('请输入管理员 Token')
      return
    }

    setLoading(true)
    setError('')

    try {
      const res = await fetch('/admin/v1/dashboard/overview', {
        headers: { Authorization: `Bearer ${token}` },
      })

      if (res.ok) {
        setAdminToken(token)
        onLogin()
      } else {
        setError('Token 无效，请检查后重试')
      }
    } catch (err) {
      setError('连接失败，请检查服务是否运行')
    } finally {
      setLoading(false)
    }
  }

  return (
    <div className="flex min-h-screen items-center justify-center bg-background p-5">
      <BentoCard className="w-full max-w-md p-10">
        <div className="mb-8 flex flex-col items-center">
          <div className="mb-4 flex h-16 w-16 items-center justify-center rounded-2xl bg-primary">
            <span className="text-3xl font-bold text-white">G</span>
          </div>
          <h1 className="text-2xl font-bold">AI Gateway</h1>
          <p className="mt-1 text-sm text-muted-foreground">管理控制台</p>
        </div>

        <form onSubmit={handleSubmit} className="space-y-5">
          <div>
            <label className="mb-2 block text-sm font-medium">管理员 Token</label>
            <Input
              type="password"
              value={token}
              onChange={e => { setToken(e.target.value); setError('') }}
              placeholder="请输入管理员 Token"
              className={error ? 'border-destructive' : ''}
            />
            {error && (
              <p className="mt-2 text-sm text-destructive">{error}</p>
            )}
          </div>

          <Button type="submit" className="w-full" disabled={loading}>
            {loading ? '验证中...' : '登录'}
          </Button>
        </form>

        <div className="mt-6 rounded-xl bg-muted/50 p-4 text-sm text-muted-foreground">
          <p className="mb-2 font-medium">默认 Token</p>
          <code className="block rounded-lg border border-border bg-background p-2 font-mono text-xs">
            admin-xxxxxxxxxxxxxxxx
          </code>
          <p className="mt-2 text-xs">
            请在配置文件 <code>config/config.yaml</code> 中查看或修改
          </p>
        </div>
      </BentoCard>
    </div>
  )
}

export default Login
