import { getAuthHeaders } from './auth'

const API_BASE = '/admin/v1'

export interface DashboardOverview {
  overview: {
    todayRequests: number
    sevenDayRequests: number
    thirtyDayRequests: number
    todayInputTokens: number
    todayOutputTokens: number
    todayFailedRequests: number
    failureRate: number
  }
  trend: Array<{ time: string; requests: number; tokens: number }>
  modelDistribution: Array<{ name: string; tokens: number }>
  providerDistribution: Array<{ name: string; tokens: number }>
  recentLogs?: Array<{
    id: number
    requestId: string
    timestamp: string
    modelName: string
    providerName: string
    success: boolean
    totalLatencyMs: number
    cacheReadInputTokens: number
  }>
}

export async function fetchDashboard(): Promise<DashboardOverview> {
  const res = await fetch(`${API_BASE}/dashboard/overview`, {
    headers: getAuthHeaders(),
  })
  if (!res.ok) throw new Error('Failed to fetch dashboard')
  return res.json()
}
