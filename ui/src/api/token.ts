import { getAuthHeaders } from './auth'

const API_BASE = '/admin/v1'

export interface Token {
  id: string
  provider_id: string
  name: string
  status: string
  quota_total: number
  quota_used: number
  quota_remaining: number
  success_rate: number
  last_used_at: string
  created_at: string
}

export async function fetchTokens(providerId?: string): Promise<{ items: Token[]; total: number }> {
  const url = providerId ? `${API_BASE}/tokens?provider_id=${providerId}` : `${API_BASE}/tokens`
  const res = await fetch(url, {
    headers: getAuthHeaders(),
  })
  if (!res.ok) throw new Error('Failed to fetch tokens')
  return res.json()
}

export async function createToken(data: { provider_id: string; name: string; token_value: string; quota_total?: number }): Promise<Token> {
  const res = await fetch(`${API_BASE}/tokens`, {
    method: 'POST',
    headers: {
      'Content-Type': 'application/json',
      ...getAuthHeaders(),
    },
    body: JSON.stringify(data),
  })
  if (!res.ok) throw new Error('Failed to create token')
  return res.json()
}

export async function deleteToken(id: string): Promise<void> {
  const res = await fetch(`${API_BASE}/tokens/${id}`, {
    method: 'DELETE',
    headers: getAuthHeaders(),
  })
  if (!res.ok) throw new Error('Failed to delete token')
}

export async function updateToken(id: string, data: Partial<Token>): Promise<Token> {
  const res = await fetch(`${API_BASE}/tokens/${id}`, {
    method: 'PUT',
    headers: {
      'Content-Type': 'application/json',
      ...getAuthHeaders(),
    },
    body: JSON.stringify(data),
  })
  if (!res.ok) throw new Error('Failed to update token')
  return res.json()
}

export async function revealToken(id: string): Promise<{ token_value: string }> {
  const res = await fetch(`${API_BASE}/tokens/${id}/reveal`, {
    headers: getAuthHeaders(),
  })
  if (!res.ok) throw new Error('Failed to reveal token')
  return res.json()
}
