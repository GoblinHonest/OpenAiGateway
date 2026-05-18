import { getAuthHeaders } from './auth'

const API_BASE = '/admin/v1'

export interface APIKey {
  id: string
  name: string
  key_prefix: string
  group_id: string
  group_name: string
  status: 'active' | 'revoked'
  quota_limit: number
  quota_used: number
  rate_limit: number
  expires_at: string | null
  created_at: string
  updated_at: string
}

export interface CreateAPIKeyRequest {
  name: string
  group_id?: string
  quota_limit?: number
  rate_limit?: number
  expires_at?: string
}

export interface CreateAPIKeyResponse {
  id: string
  name: string
  key: string
  key_prefix: string
}

export async function fetchAPIKeys(): Promise<{ items: APIKey[]; total: number }> {
  const res = await fetch(`${API_BASE}/api-keys`, {
    headers: getAuthHeaders(),
  })
  if (!res.ok) throw new Error('Failed to fetch API keys')
  return res.json()
}

export async function createAPIKey(data: CreateAPIKeyRequest): Promise<CreateAPIKeyResponse> {
  const res = await fetch(`${API_BASE}/api-keys`, {
    method: 'POST',
    headers: {
      'Content-Type': 'application/json',
      ...getAuthHeaders(),
    },
    body: JSON.stringify(data),
  })
  if (!res.ok) throw new Error('Failed to create API key')
  return res.json()
}

export async function revokeAPIKey(id: string): Promise<void> {
  const res = await fetch(`${API_BASE}/api-keys/${id}`, {
    method: 'DELETE',
    headers: getAuthHeaders(),
  })
  if (!res.ok) throw new Error('Failed to revoke API key')
}

export async function revealAPIKey(id: string): Promise<CreateAPIKeyResponse> {
  const res = await fetch(`${API_BASE}/api-keys/${id}/reveal`, {
    headers: getAuthHeaders(),
  })
  if (!res.ok) throw new Error('Failed to reveal API key')
  return res.json()
}

export interface UpdateAPIKeyRequest {
  name?: string
  group_id?: string
  rate_limit_config?: Record<string, unknown>
}

export async function updateAPIKey(id: string, data: UpdateAPIKeyRequest): Promise<APIKey> {
  const res = await fetch(`${API_BASE}/api-keys/${id}`, {
    method: 'PUT',
    headers: {
      'Content-Type': 'application/json',
      ...getAuthHeaders(),
    },
    body: JSON.stringify(data),
  })
  if (!res.ok) throw new Error('Failed to update API key')
  return res.json()
}
