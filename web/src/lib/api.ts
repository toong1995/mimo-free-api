const ENV_BASE_URL = import.meta.env.VITE_API_BASE_URL ?? ''

function getBaseUrl(): string {
  return localStorage.getItem('api_base_url') || ENV_BASE_URL
}

// 管理密钥：自用场景复用 API Key，存储在 localStorage，所有 admin 请求自动携带
export function getAdminKey(): string {
  return localStorage.getItem('admin_key') || ''
}

export function setAdminKey(key: string): void {
  if (key) {
    localStorage.setItem('admin_key', key)
  } else {
    localStorage.removeItem('admin_key')
  }
}

export async function apiFetch(path: string, init?: RequestInit): Promise<Response> {
  const headers = new Headers(init?.headers)
  const key = getAdminKey()
  if (key && !headers.has('Authorization')) {
    headers.set('Authorization', `Bearer ${key}`)
  }
  return fetch(`${getBaseUrl()}${path}`, { ...init, headers })
}
