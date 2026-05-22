import { getAuthHeaders } from './auth'

const API_BASE = '/admin/v1'

export interface LogEntry {
  id: number
  requestId: string
  timestamp: string
  createdAt: string
  modelName: string
  providerName: string
  success: boolean
  totalLatencyMs: number
  inputTokens: number
  outputTokens: number
  cacheReadInputTokens: number
  interfaceType: string
  apiKeyName: string
  clientIp: string
  stream: boolean
  errorMessage: string
  firstTokenLatencyMs: number
  route: string
  requestHeaders: Record<string, string>
  requestBody: string
  responseHeaders: Record<string, string>
  responseBody: string
}

export async function fetchLogs(params: Record<string, string>): Promise<{ items: LogEntry[]; total: number }> {
  const query = new URLSearchParams(params).toString()
  const res = await fetch(`${API_BASE}/logs/requests?${query}`, {
    headers: getAuthHeaders(),
  })
  if (!res.ok) throw new Error('Failed to fetch logs')
  return res.json()
}

export async function fetchLogDetail(requestId: string): Promise<LogEntry> {
  const res = await fetch(`${API_BASE}/logs/requests/${requestId}`, {
    headers: getAuthHeaders(),
  })
  if (!res.ok) throw new Error('Failed to fetch log detail')
  return res.json()
}
