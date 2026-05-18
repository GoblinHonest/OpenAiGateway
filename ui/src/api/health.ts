import { getAuthHeaders } from './auth'

const API_BASE = '/admin/v1'

export interface ProviderHealth {
  status: 'healthy' | 'degraded' | 'unhealthy'
  avg_latency_ms: number
  error_rate: number
  availability: number
  last_check_at: string
  consecutive_failures: number
  consecutive_passes: number
}

export interface ProviderHealthResponse {
  providers: ProviderHealth[]
}

export async function fetchProviderHealth(): Promise<ProviderHealthResponse> {
  const res = await fetch(`${API_BASE}/health/providers`, {
    headers: getAuthHeaders(),
  })
  if (!res.ok) throw new Error('Failed to fetch provider health')
  return res.json()
}
