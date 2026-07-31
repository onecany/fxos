import { CryptoService } from '../crypto'
import { httpClient } from '../httpClient'

// API_BASE: when the frontend is served by Nginx (Docker), '/api' works because
// Nginx proxies /api/ to the backend. When accessed directly (no reverse proxy),
// set VITE_API_BASE_URL env var at build time, e.g. VITE_API_BASE_URL=http://IP:8080
const apiBase = import.meta.env.VITE_API_BASE_URL
  ? `${import.meta.env.VITE_API_BASE_URL}/api`
  : '/api'

export const API_BASE = apiBase

/** Build a full API URL from a path like "/login" or "/traders/:id" */
export function apiUrl(path: string): string {
  return `${API_BASE}${path}`
}

export { CryptoService, httpClient }

// Helper function to get auth headers
export function getAuthHeaders(): Record<string, string> {
  const token = localStorage.getItem('auth_token')
  const headers: Record<string, string> = {
    'Content-Type': 'application/json',
  }

  if (token) {
    headers['Authorization'] = `Bearer ${token}`
  }

  return headers
}

export async function handleJSONResponse<T>(res: Response): Promise<T> {
  const text = await res.text()
  if (!res.ok) {
    let message = text || res.statusText
    try {
      const data = text ? JSON.parse(text) : null
      if (data && typeof data === 'object') {
        message = data.error || data.message || message
      }
    } catch {
      /* ignore JSON parse errors */
    }
    throw new Error(message || 'Request failed')
  }
  if (!text) {
    return {} as T
  }
  return JSON.parse(text) as T
}
