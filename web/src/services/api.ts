import axios from 'axios'

const api = axios.create({
  baseURL: '/api',
  timeout: 30000,
  withCredentials: true,
})

// Add auth token to requests
api.interceptors.request.use((config) => {
  const token = localStorage.getItem('access_token')
  if (token) {
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
  logo_url: string
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
  // System config
  max_concurrent_requests: number
  request_queue_timeout: number
  health_check_interval: number
  health_check_fail_threshold: number
  circuit_breaker_threshold: number
  circuit_breaker_timeout: number
  relay_timeout: number
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
  rate_limit_rpm: number
  rate_limit_tpm: number
  rate_limit_concurrent: number
  settings: string
  company_id: number
  department_id: number
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
  api.get('/user/market/models', { params })

export const getMarketModel = (id: string) =>
  api.get(`/market/models/${id}`)

export const getMarketProviders = () =>
  api.get('/user/market/providers')

export const getMarketStats = () =>
  api.get('/user/market/stats')

export const calculatePrice = (params: { model_id: string; prompt_tokens: number; completion_tokens: number }) =>
  api.get('/user/market/calculate', { params })

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

export const updateToken = (id: number, data: { name?: string; models?: string; status?: number; remain_quota?: number; unlimited_quota?: boolean; rate_limit_rpm?: number; rate_limit_tpm?: number; rate_limit_concurrent?: number }) =>
  api.put(`/token/${id}`, data)

export const getUserInfo = () =>
  api.get('/user/self')

export const updateUserInfo = (data: { username?: string; password?: string; display_name?: string; email?: string }) =>
  api.put('/user/self', data)

export const getNotifications = (limit?: number, offset?: number) =>
  api.get('/user/notifications', { params: { limit, offset } })

export const getUnreadNotificationCount = () =>
  api.get('/user/notifications/unread-count')

export const markNotificationAsRead = (id: number) =>
  api.put(`/user/notifications/${id}/read`)

export const markAllNotificationsAsRead = () =>
  api.put('/user/notifications/read-all')

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

// Admin usage stats
export const getAdminUsageSummary = (params?: { start?: string; end?: string; model?: string; channel?: number }) =>
  api.get('/admin/usage/summary', { params })

export const getAdminUsageByUsers = (params?: { start?: string; end?: string }) =>
  api.get('/admin/usage/by-users', { params })

export const getAdminUsageByModels = (params?: { start?: string; end?: string }) =>
  api.get('/admin/usage/by-models', { params })

export const getChannelHealth = () =>
  api.get('/admin/channels/health')

// Channel types
export interface Channel {
  id: number
  type: number
  type_name: string
  key: string
  status: number
  name: string
  weight: number
  created_time: number
  test_time: number
  response_time: number
  base_url: string
  other: string
  balance: number
  balance_updated_time: number
  models: string
  group: string
  used_quota: number
  model_mapping: string
  priority: number
  config: string
  system_prompt: string
}

export const getChannels = (params?: { limit?: number; offset?: number }) =>
  api.get<{ success: boolean; data: Channel[]; message?: string; total?: number }>('/channel', { params })

export const getChannelGroups = () =>
  api.get<{ success: boolean; data: string[] }>('/channel/groups')

export const getChannel = (id: number) =>
  api.get<{ success: boolean; data: Channel; message?: string }>(`/channel/${id}`)

export const createChannel = (data: Partial<Channel>) =>
  api.post<{ success: boolean; data?: Channel; message?: string }>('/channel', data)

export const updateChannel = (_id: number, data: Partial<Channel>) =>
  api.put<{ success: boolean; data?: Channel; message?: string }>('/channel', { ...data, id: _id })

export const deleteChannel = (id: number) =>
  api.delete<{ success: boolean; message?: string }>(`/channel/${id}`)

export const testChannel = (id: number) =>
  api.get<{ success: boolean; message?: string }>(`/channel/test/${id}`)

export const getOpsUsers = (params?: { limit?: number; offset?: number }) =>
  api.get('/admin/ops/users', { params })

export const updateUser = (id: number, data: { group?: string; quota?: number; role?: number; status?: number }) =>
  api.put(`/admin/${id}`, data)

export const getAlertConfig = () =>
  api.get('/admin/alerts/config')

export const updateAlertConfig = (data: Partial<AlertConfig>) =>
  api.put('/admin/alerts/config', data)

export const getAllNotifications = (limit?: number, offset?: number) =>
  api.get('/admin/notifications', { params: { limit, offset } })

export const createNotification = (data: { title: string; content: string; type?: string }) =>
  api.post('/admin/notifications', data)

export const deleteNotification = (id: number) =>
  api.delete(`/admin/notifications/${id}`)

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

// Sign-in API functions
export interface SignInRecord {
  date: string
  status: string
  quota: number
}

export const getSignInRecords = () =>
  api.get('/user/signin/records')

export const signIn = () =>
  api.post('/user/signin')

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

// Company API functions (Platform/Company/Department/Team hierarchy)
export interface Company {
  id: number
  name: string
  code: string
  logo_url: string
  description: string
  quota_limit: number
  quota_used: number
  status: number
  department_count?: number
  team_count?: number
}

export interface Department {
  id: number
  company_id: number
  name: string
  code: string
  description: string
  quota_limit: number
  quota_used: number
  status: number
  team_count?: number
}

export const createCompany = (data: { name: string; code: string; logo_url?: string; description?: string; quota_limit?: number }) =>
  api.post('/company', data)

export const getAllCompanies = () =>
  api.get<{ success: boolean; data: Company[] }>('/company')

export const getCompany = (id: number) =>
  api.get<{ success: boolean; data: Company & { departments: Department[] } }>(`/company/${id}`)

export const updateCompany = (id: number, data: Partial<Company>) =>
  api.put(`/company/${id}`, data)

export const deleteCompany = (id: number) =>
  api.delete<{ success: boolean; message?: string }>(`/company/${id}`)

export const createDepartment = (companyId: number, data: { name: string; code?: string; description?: string; quota_limit?: number }) =>
  api.post(`/company/${companyId}/departments`, data)

export const getDepartments = (companyId: number) =>
  api.get<{ success: boolean; data: Department[] }>(`/company/${companyId}/departments`)

export const getDepartment = (id: number) =>
  api.get<{ success: boolean; data: Department & { teams: any[] } }>(`/department/${id}`)

export const updateDepartment = (id: number, data: Partial<Department>) =>
  api.put(`/department/${id}`, data)

export const deleteDepartment = (id: number) =>
  api.delete<{ success: boolean; message?: string }>(`/department/${id}`)

// Model market API functions
export const getMarketGroups = () =>
  api.get('/user/market/groups')

export const getMarketModelsByGroup = (groupId: number) =>
  api.get(`/user/market/groups/${groupId}/models`)

export const getModelPricing = (modelId: string) =>
  api.get(`/user/market/models/${modelId}/pricing`)

export const getModelTrial = (modelId: string) =>
  api.get(`/user/market/models/${modelId}/trial`)

export const startModelTrial = (modelId: string) =>
  api.post(`/user/market/models/${modelId}/trial`)

export const getUserTrials = () =>
  api.get('/user/market/trials')

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
  group_id: number
  tags: string
  is_trial: boolean
  trial_quota: number
  sla: string
  rate_limit_rpm: number
  rate_limit_tpm: number
  created_at: number
  updated_at: number
}

export const listModels = () =>
  api.get<{ success: boolean; data: ModelItem[]; message?: string }>('/admin/models')

export const getModel = (id: string) =>
  api.get<{ success: boolean; data: ModelItem; message?: string }>(`/admin/models/model?id=${encodeURIComponent(id)}`)

export const createModel = (data: Partial<ModelItem>) =>
  api.post<{ success: boolean; data?: ModelItem; message?: string }>('/admin/models', data)

export const updateModel = (id: string, data: Partial<ModelItem>) =>
  api.put<{ success: boolean; data?: ModelItem; message?: string }>(`/admin/models/model?id=${encodeURIComponent(id)}`, data)

export const deleteModel = (id: string) =>
  api.delete<{ success: boolean; message?: string }>(`/admin/models/model?id=${encodeURIComponent(id)}`)

export const batchDeleteModels = (ids: string[]) =>
  api.post<{ success: boolean; message?: string }>('/admin/models/batch-delete', { ids })

export const uploadModelLogo = (formData: FormData) =>
  api.post<{ success: boolean; data: { url: string } }>('/admin/models/upload-logo', formData, {
    headers: { 'Content-Type': 'multipart/form-data' },
  })

export const uploadProviderLogo = (formData: FormData) =>
  api.post<{ success: boolean; data: { url: string } }>('/admin/providers/upload-logo', formData, {
    headers: { 'Content-Type': 'multipart/form-data' },
  })

export const listLogos = () =>
  api.get<{ success: boolean; data: { name: string; url: string }[] }>('/admin/logos')

export const deleteLogo = (filename: string) =>
  api.delete<{ success: boolean; message?: string }>('/admin/logos', { params: { filename } })

export const getModelTypes = () =>
  api.get<{ success: boolean; data: { value: string; label: string }[] }>('/admin/models/types')

export const getModelStatuses = () =>
  api.get<{ success: boolean; data: { value: string; label: string }[] }>('/admin/models/statuses')

// Provider Management APIs
export interface Provider {
  id: number
  code: string
  name: string
  logo_url: string
  description: string
  website: string
  status: string
  sort_order: number
  created_at: number
  updated_at: number
}

export const getProviders = () =>
  api.get<{ success: boolean; data: Provider[]; message?: string }>('/admin/providers')

export const getProvider = (id: number) =>
  api.get<{ success: boolean; data: Provider; message?: string }>(`/admin/providers/${id}`)

export const createProvider = (data: Partial<Provider>) =>
  api.post<{ success: boolean; data?: Provider; message?: string }>('/admin/providers', data)

export const updateProvider = (id: number, data: Partial<Provider>) =>
  api.put<{ success: boolean; data?: Provider; message?: string }>(`/admin/providers/${id}`, data)

export const deleteProvider = (id: number) =>
  api.delete<{ success: boolean; message?: string }>(`/admin/providers/${id}`)

export const getProviderStatuses = () =>
  api.get<{ success: boolean; data: { value: string; label: string }[] }>('/admin/providers/statuses')

// Option APIs for system settings
export const getOptions = () =>
  api.get<{ success: boolean; data: { key: string; value: string }[] }>('/option/')

export const updateOption = (key: string, value: string) =>
  api.put<{ success: boolean; message?: string }>('/option/', { key, value })