import { getAuthHeaders } from './auth'

const API_BASE = '/admin/v1'

export interface CacheConfig {
  enabled: boolean
  type: string
  description: string
}

export interface CacheStats {
  enabled: boolean
  total_entries: number
}

export interface CacheEntries {
  entries: Record<string, unknown>
  total: number
  message: string
}

export async function fetchCacheConfig(): Promise<CacheConfig> {
  const res = await fetch(`${API_BASE}/cache/config`, {
    headers: getAuthHeaders(),
  })
  if (!res.ok) throw new Error('Failed to fetch cache config')
  return res.json()
}

export async function fetchCacheStats(): Promise<CacheStats> {
  const res = await fetch(`${API_BASE}/cache/stats`, {
    headers: getAuthHeaders(),
  })
  if (!res.ok) throw new Error('Failed to fetch cache stats')
  return res.json()
}

export async function fetchCacheEntries(): Promise<CacheEntries> {
  const res = await fetch(`${API_BASE}/cache/entries`, {
    headers: getAuthHeaders(),
  })
  if (!res.ok) throw new Error('Failed to fetch cache entries')
  return res.json()
}

export async function clearCache(): Promise<{ message: string }> {
  const res = await fetch(`${API_BASE}/cache`, {
    method: 'DELETE',
    headers: getAuthHeaders(),
  })
  if (!res.ok) throw new Error('Failed to clear cache')
  return res.json()
}
