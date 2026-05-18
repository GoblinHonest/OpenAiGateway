import { getAuthHeaders } from './auth'

const API_BASE = '/admin/v1'

export interface BindingInfo {
  id: string
  provider_id: string
  provider_name: string
  upstream_model_name: string
  weight: number
  priority: number
  enabled: boolean
}

export interface Model {
  id: string
  name: string
  display_name: string
  description: string
  model_type: string
  context_window: number
  max_output_tokens: number
  input_price_per_1k: number
  output_price_per_1k: number
  enabled: boolean
  created_at: string
  updated_at: string
  bindings?: BindingInfo[]
}

export interface ModelProviderBinding {
  id: string
  model_id: string
  provider_id: string
  provider_name: string
  upstream_model_name: string
  weight: number
  priority: number
  enabled: boolean
}

export async function fetchModels(): Promise<{ items: Model[]; total: number }> {
  const res = await fetch(`${API_BASE}/models`, {
    headers: getAuthHeaders(),
  })
  if (!res.ok) throw new Error('Failed to fetch models')
  return res.json()
}

export async function fetchModelBindings(modelId: string): Promise<{ items: ModelProviderBinding[] }> {
  const res = await fetch(`${API_BASE}/models/${modelId}/bindings`, {
    headers: getAuthHeaders(),
  })
  if (!res.ok) throw new Error('Failed to fetch bindings')
  return res.json()
}

export async function createModel(data: Partial<Model>): Promise<Model> {
  const res = await fetch(`${API_BASE}/models`, {
    method: 'POST',
    headers: {
      'Content-Type': 'application/json',
      ...getAuthHeaders(),
    },
    body: JSON.stringify(data),
  })
  if (!res.ok) throw new Error('Failed to create model')
  return res.json()
}

export async function createModelWithBindings(
  model: Partial<Model>,
  bindings: { provider_id: string; upstream_model_name: string; weight: number; priority: number }[]
): Promise<Model> {
  const res = await fetch(`${API_BASE}/models/with-bindings`, {
    method: 'POST',
    headers: {
      'Content-Type': 'application/json',
      ...getAuthHeaders(),
    },
    body: JSON.stringify({ model, bindings }),
  })
  if (!res.ok) throw new Error('Failed to create model with bindings')
  return res.json()
}

export async function updateModel(id: string, data: Partial<Model>): Promise<Model> {
  const res = await fetch(`${API_BASE}/models/${id}`, {
    method: 'PUT',
    headers: {
      'Content-Type': 'application/json',
      ...getAuthHeaders(),
    },
    body: JSON.stringify(data),
  })
  if (!res.ok) throw new Error('Failed to update model')
  return res.json()
}

export async function deleteModel(id: string): Promise<void> {
  const res = await fetch(`${API_BASE}/models/${id}`, {
    method: 'DELETE',
    headers: getAuthHeaders(),
  })
  if (!res.ok) throw new Error('Failed to delete model')
}

export async function bindProvider(modelId: string, providerId: string, upstreamModelName: string = '', weight: number = 1, priority: number = 0): Promise<ModelProviderBinding> {
  const res = await fetch(`${API_BASE}/model-provider-bindings`, {
    method: 'POST',
    headers: {
      'Content-Type': 'application/json',
      ...getAuthHeaders(),
    },
    body: JSON.stringify({ model_id: modelId, provider_id: providerId, upstream_model_name: upstreamModelName, weight, priority }),
  })
  if (!res.ok) throw new Error('Failed to bind provider')
  return res.json()
}

export async function removeBinding(bindingId: string): Promise<void> {
  const res = await fetch(`${API_BASE}/model-provider-bindings/${bindingId}`, {
    method: 'DELETE',
    headers: getAuthHeaders(),
  })
  if (!res.ok) throw new Error('Failed to remove binding')
}
