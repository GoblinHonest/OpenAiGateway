import { getAuthHeaders } from './auth'

const API_BASE = '/admin/v1'

export interface Group {
  id: string
  name: string
  description: string
  load_balance_strategy: string
  rate_limit_config: Record<string, any>
  quota_config: Record<string, any>
  enabled: boolean
  version: number
  metadata: Record<string, any>
  created_at: string
  updated_at: string
  model_ids?: string[]
}

export interface GroupModel {
  id: string
  group_id: string
  model_id: string
  enabled: boolean
  created_at: string
  updated_at: string
}

export async function fetchGroups(): Promise<{ items: Group[]; total: number }> {
  const res = await fetch(`${API_BASE}/groups`, {
    headers: getAuthHeaders(),
  })
  if (!res.ok) throw new Error('Failed to fetch groups')
  return res.json()
}

export async function fetchGroup(id: string): Promise<Group> {
  const res = await fetch(`${API_BASE}/groups/${id}`, {
    headers: getAuthHeaders(),
  })
  if (!res.ok) throw new Error('Failed to fetch group')
  return res.json()
}

export async function createGroup(data: Partial<Group>): Promise<Group> {
  const res = await fetch(`${API_BASE}/groups`, {
    method: 'POST',
    headers: {
      'Content-Type': 'application/json',
      ...getAuthHeaders(),
    },
    body: JSON.stringify(data),
  })
  if (!res.ok) throw new Error('Failed to create group')
  return res.json()
}

export async function updateGroup(id: string, data: Partial<Group>): Promise<Group> {
  const res = await fetch(`${API_BASE}/groups/${id}`, {
    method: 'PUT',
    headers: {
      'Content-Type': 'application/json',
      ...getAuthHeaders(),
    },
    body: JSON.stringify(data),
  })
  if (!res.ok) throw new Error('Failed to update group')
  return res.json()
}

export async function deleteGroup(id: string): Promise<void> {
  const res = await fetch(`${API_BASE}/groups/${id}`, {
    method: 'DELETE',
    headers: getAuthHeaders(),
  })
  if (!res.ok) throw new Error('Failed to delete group')
}

export async function fetchGroupModels(groupId: string): Promise<{ items: GroupModel[] }> {
  const res = await fetch(`${API_BASE}/groups/${groupId}/models`, {
    headers: getAuthHeaders(),
  })
  if (!res.ok) throw new Error('Failed to fetch group models')
  return res.json()
}

export async function setGroupModels(groupId: string, modelIds: string[]): Promise<void> {
  const res = await fetch(`${API_BASE}/groups/${groupId}/models`, {
    method: 'PUT',
    headers: {
      'Content-Type': 'application/json',
      ...getAuthHeaders(),
    },
    body: JSON.stringify({ model_ids: modelIds }),
  })
  if (!res.ok) throw new Error('Failed to set group models')
}
