import { getAuthHeaders } from './auth'

const API_BASE = '/admin/v1'

export interface FormatEndpoint {
  format: string
  url: string
  path: string
}

export interface Provider {
  id: string
  name: string
  base_url: string
  status: string
  supported_formats: string[]
  format_endpoints: FormatEndpoint[]
  models: string[]
  custom_headers: Record<string, string>
  created_at: string
}

export async function fetchProviders(): Promise<{ items: Provider[]; total: number }> {
  const res = await fetch(`${API_BASE}/providers`, {
    headers: getAuthHeaders(),
  })
  if (!res.ok) throw new Error('Failed to fetch providers')
  return res.json()
}

export async function createProvider(data: Partial<Provider>): Promise<Provider> {
  const res = await fetch(`${API_BASE}/providers`, {
    method: 'POST',
    headers: {
      'Content-Type': 'application/json',
      ...getAuthHeaders(),
    },
    body: JSON.stringify(data),
  })
  if (!res.ok) throw new Error('Failed to create provider')
  return res.json()
}

export async function updateProvider(id: string, data: Partial<Provider>): Promise<Provider> {
  const res = await fetch(`${API_BASE}/providers/${id}`, {
    method: 'PUT',
    headers: {
      'Content-Type': 'application/json',
      ...getAuthHeaders(),
    },
    body: JSON.stringify(data),
  })
  if (!res.ok) throw new Error('Failed to update provider')
  return res.json()
}

export async function fetchProviderModels(id: string): Promise<string[]> {
  const res = await fetch(`${API_BASE}/providers/${id}/models`, {
    headers: getAuthHeaders(),
  })
  if (!res.ok) return []
  const data = await res.json()
  return data.models || []
}

export async function deleteProvider(id: string): Promise<void> {
  const res = await fetch(`${API_BASE}/providers/${id}`, {
    method: 'DELETE',
    headers: getAuthHeaders(),
  })
  if (!res.ok) throw new Error('Failed to delete provider')
}
