import axios from 'axios'

const api = axios.create({
  baseURL: '/api',
  timeout: 30000,
  withCredentials: true,
})

// Add auth token to requests
api.interceptors.request.use((config) => {
  const token = localStorage.getItem('access_token')
  // Only add Bearer token for real API keys (sk-), not for OAuth session tokens
  if (token && token.startsWith('sk-')) {
    config.headers.Authorization = `Bearer ${token}`
  }
  return config
})

// Handle auth errors
api.interceptors.response.use(
  (response) => response,
  (error) => {
    if (error.response?.status === 401) {
      const token = localStorage.getItem('access_token')
      // Only clear and redirect for real API keys (sk-)
      // OAuth session users rely on session cookie, not Bearer token
      if (token && token.startsWith('sk-')) {
        localStorage.removeItem('access_token')
        window.location.href = '/login'
      }
    }
    return Promise.reject(error)
  }
)

export default api

// Types
export interface Model {
  id: string
  name: string
  provider: string
  model_type: string
  description: string
  context_len: number
  input_price: number
  output_price: number
  capabilities: string
  status: string
  icon_url: string
  group_id?: number
  is_trial?: boolean
  trial_quota?: number
  sla?: string
}

export interface ModelGroup {
  id: number
  name: string
  code: string
  description: string
  icon_url: string
  model_count: number
}

export interface ModelTrial {
  id: number
  user_id: number
  model_id: string
  quota_used: number
  status: string
  created_at: number
  expires_at: number
}

export interface MarketStats {
  total_models: number
  total_providers: number
  total_groups: number
  chat_models: number
  embedding_models: number
  image_models: number
  trial_models: number
  avg_input_price: number
  avg_output_price: number
}

export interface User {
  id: number
  username: string
  email: string
  quota: number
  display_name: string
  role: number
}

export interface TopupOrder {
  id: string
  amount: number
  quota: number
  status: string
  pay_method: string
  created_at: number
  paid_at?: number
  payment_url?: string
  expired_at?: number
}

export interface Invoice {
  id: string
  order_ids: string
  amount: number
  status: string
  title: string
  tax_no: string
  created_at: number
}

export interface UsageSummary {
  total_requests: number
  total_quota: number
  total_prompt_tokens: number
  total_completion_tokens: number
  total_tokens: number
}

export interface ModelUsage {
  model_name: string
  request_count: number
  quota: number
  prompt_tokens: number
  completion_tokens: number
}

// Ops types
export interface OpsStats {
  today_revenue: number
  today_usage_tokens: number
  active_users: number
  channel_health_rate: number
  total_users: number
  total_channels: number
  total_tokens: number
  total_quota: number
  revenue_by_day: Record<string, number>
  usage_by_model: Record<string, number>
  topup_by_day: Record<string, number>
}

export interface ChannelHealth {
  id: number
  name: string
  status: string
  type: number
  base_url: string
  balance: number
  success_rate: number
  avg_latency: number
  priority: number
  is_enabled: boolean
}

export interface SystemHealth {
  uptime: number
  queue: {
    size: number
    max_size: number
    utilization: number
  }
  channels: Record<string, any>
  circuit_breakers: Record<string, any>
  config: {
    max_concurrent: number
    health_interval: number
    cb_threshold: number
  }
}

export interface AlertConfig {
  channel_failure_threshold: number
  queue_utilization_alert: number
  error_rate_alert: number
  latency_threshold: number
  alert_email: string
  alert_webhook: string
  enabled: boolean
}

// Tenant types
export interface Tenant {
  id: number
  name: string
  code: string
  status: number
  owner_id: number
  quota_limit: number
  quota_used: number
  max_users: number
  max_channels: number
  settings: string
  created_at: string
  updated_at: string
}

export interface TenantUser {
  id: number
  username: string
  email: string
  display_name: string
  role: number
  quota_alloc: number
  used_quota: number
  created_at: number
}

export interface QuotaAllocation {
  id: number
  tenant_id: number
  user_id: number
  quota: number
  used_quota: number
}

export interface AuditLog {
  id: number
  tenant_id: number
  user_id: number
  action: string
  target: string
  target_id: number
  details: string
  ip: string
  user_agent: string
  created_at: number
}

// API functions
export const getMarketModels = (params?: { type?: string; q?: string; limit?: number; offset?: number }) =>
  api.get('/market/models', { params })

export const getMarketModel = (id: string) =>
  api.get(`/market/models/${id}`)

export const getMarketProviders = () =>
  api.get('/market/providers')

export const getMarketStats = () =>
  api.get('/market/stats')

export const calculatePrice = (params: { model_id: string; prompt_tokens: number; completion_tokens: number }) =>
  api.get('/market/calculate', { params })

export const getDashboard = () =>
  api.get('/dashboard')

export const getUsageSummary = (params?: { start?: string; end?: string }) =>
  api.get('/usage/summary', { params })

export const getUsageByModel = (params?: { start?: string; end?: string }) =>
  api.get('/usage/by-model', { params })

export const getUsageByDay = (params?: { start?: string; end?: string }) =>
  api.get('/usage/daily', { params })

export const getUsageDetail = (params?: { model?: string; start?: string; end?: string }) =>
  api.get('/usage/detail', { params })

export const getTopupOrders = (params?: { status?: string; limit?: number; offset?: number }) =>
  api.get('/topup', { params })

export const createTopupOrder = (data: { amount: number; pay_method: string }) =>
  api.post('/topup/create', data)

export const cancelTopupOrder = (id: string) =>
  api.post(`/topup/${id}/cancel`)

export const getInvoices = (params?: { status?: string; limit?: number; offset?: number }) =>
  api.get('/invoice', { params })

export const createInvoice = (data: {
  order_ids: string
  amount: number
  title: string
  tax_no?: string
  address?: string
  phone?: string
  bank?: string
  account?: string
  email?: string
  remark?: string
}) => api.post('/invoice', data)

export const getTokens = (params?: { limit?: number; offset?: number }) =>
  api.get('/token', { params })

export const createToken = (data: { name: string; models?: string }) =>
  api.post('/token', data)

export const deleteToken = (id: number) =>
  api.delete(`/token/${id}`)

export const getUserInfo = () =>
  api.get('/user/self')

export const login = (data: { username: string; password: string }) =>
  api.post('/user/login', data)

export const register = (data: {
  username: string
  email: string
  password: string
  email_code?: string
  invitation_code?: string
}) => api.post('/user/register', data)

export const logout = () =>
  api.get('/user/logout')

export const sendEmailCode = (email: string, type: 'register' | 'reset' = 'register') => {
  const endpoint = type === 'register' ? '/verification' : '/reset_password'
  return api.get(endpoint, { params: { email } })
}

export const resetPassword = (data: { email: string; code: string; password: string }) =>
  api.post('/user/reset', data)

// Ops API functions
export const getOpsStats = () =>
  api.get('/admin/ops/stats')

export const getOpsRevenue = (params?: { start?: string; end?: string }) =>
  api.get('/admin/ops/revenue', { params })

export const getOpsUsage = (params?: { start?: string; end?: string }) =>
  api.get('/admin/ops/usage', { params })

export const getChannelHealth = () =>
  api.get('/admin/channels/health')

export const getOpsUsers = (params?: { limit?: number; offset?: number }) =>
  api.get('/admin/ops/users', { params })

export const getAlertConfig = () =>
  api.get('/admin/alerts/config')

export const updateAlertConfig = (data: Partial<AlertConfig>) =>
  api.put('/admin/alerts/config', data)

export const getSystemHealth = () =>
  api.get('/admin/system/health')

export const exportReport = (params: { type: string; start?: string; end?: string }) =>
  api.get('/admin/reports/export', { params })

// Tenant API functions
export const createTenant = (data: { name: string; code: string }) =>
  api.post('/tenant', data)

export const getMyTenants = () =>
  api.get('/tenant')

export const getTenant = (id: number) =>
  api.get(`/tenant/${id}`)

export const updateTenant = (id: number, data: { name?: string; settings?: string; max_users?: number; max_channels?: number }) =>
  api.put(`/tenant/${id}`, data)

export const getTenantUsers = (id: number) =>
  api.get(`/tenant/${id}/users`)

export const inviteUser = (tenantId: number, data: { user_id: number; role: number; quota?: number }) =>
  api.post(`/tenant/${tenantId}/users`, data)

export const removeUser = (tenantId: number, userId: number) =>
  api.delete(`/tenant/${tenantId}/users/${userId}`)

export const updateUserRole = (tenantId: number, userId: number, data: { role: number }) =>
  api.put(`/tenant/${tenantId}/users/${userId}`, data)

export const allocateQuota = (tenantId: number, data: { target_user_id: number; quota: number }) =>
  api.post(`/tenant/${tenantId}/quota`, data)

export const getAuditLogs = (tenantId: number, params?: { limit?: number; offset?: number }) =>
  api.get(`/tenant/${tenantId}/audit`, { params })

export const leaveTenant = (tenantId: number) =>
  api.post(`/tenant/${tenantId}/leave`)

// Model market API functions
export const getMarketGroups = () =>
  api.get('/market/groups')

export const getMarketModelsByGroup = (groupId: number) =>
  api.get(`/market/groups/${groupId}/models`)

export const getModelPricing = (modelId: string) =>
  api.get(`/market/models/${modelId}/pricing`)

export const getModelTrial = (modelId: string) =>
  api.get(`/market/models/${modelId}/trial`)

export const startModelTrial = (modelId: string) =>
  api.post(`/market/models/${modelId}/trial`)

export const getUserTrials = () =>
  api.get('/market/trials')

// Model Management APIs (Admin)
export interface ModelItem {
  id: string
  name: string
  provider: string
  model_type: string
  description: string
  context_len: number
  input_price: number
  output_price: number
  capabilities: string
  status: string
  icon_url: string
  sort_order: number
  created_time: number
  updated_time: number
}

export const listModels = () =>
  api.get<{ success: boolean; data: ModelItem[] }>('/admin/models')

export const getModel = (id: string) =>
  api.get<{ success: boolean; data: ModelItem }>(`/admin/models/${id}`)

export const createModel = (data: Partial<ModelItem>) =>
  api.post<{ success: boolean; data: ModelItem }>('/admin/models', data)

export const updateModel = (id: string, data: Partial<ModelItem>) =>
  api.put<{ success: boolean; data: ModelItem }>(`/admin/models/${id}`, data)

export const deleteModel = (id: string) =>
  api.delete<{ success: boolean }>(`/admin/models/${id}`)

export const uploadModelLogo = (formData: FormData) =>
  api.post<{ success: boolean; data: { url: string } }>('/admin/models/upload-logo', formData, {
    headers: { 'Content-Type': 'multipart/form-data' },
  })

export const getModelTypes = () =>
  api.get<{ success: boolean; data: { value: string; label: string }[] }>('/admin/models/types')

export const getModelStatuses = () =>
  api.get<{ success: boolean; data: { value: string; label: string }[] }>('/admin/models/statuses')