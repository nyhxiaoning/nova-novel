import { jsonHeaders, requestJSON } from '@/lib/api-client'
import type { LayeredSettings, Settings } from './types'

export async function fetchSettings(): Promise<LayeredSettings> {
  return requestJSON('/api/settings')
}

export async function updateUserSettings(s: Settings): Promise<LayeredSettings> {
  return requestJSON('/api/settings/user', {
    method: 'PUT',
    headers: jsonHeaders,
    body: JSON.stringify(s),
  })
}

export async function updateWorkspaceSettings(s: Settings): Promise<LayeredSettings> {
  return requestJSON('/api/settings/workspace', {
    method: 'PUT',
    headers: jsonHeaders,
    body: JSON.stringify(s),
  })
}

export interface RemoteModel {
  id: string
  object: string
  owned_by: string
}

export async function fetchModelList(base_url: string, api_key: string): Promise<RemoteModel[]> {
  const data = await requestJSON<{ models: RemoteModel[] }>('/api/models/list', {
    method: 'POST',
    headers: jsonHeaders,
    body: JSON.stringify({ base_url, api_key }),
  })
  return data.models ?? []
}
