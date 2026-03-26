import axios from 'axios'

const api = axios.create({
  baseURL: '/api',
  timeout: 30000,
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
      localStorage.removeItem('access_token')
      window.location.href = '/login'
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
}

export interface User {
  id: number
  username: string
  email: string
  quota: number
  display_name: string
}

export interface TopupOrder {
  id: string
  amount: number
  quota: number
  status: string
  pay_method: string
  created_at: number
  paid_at?: number
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
  api.get('/user/token', { params })

export const createToken = (data: { name: string; models?: string }) =>
  api.post('/user/token', data)

export const deleteToken = (id: number) =>
  api.delete(`/user/token/${id}`)

export const getUserInfo = () =>
  api.get('/user/self')

export const login = (data: { username: string; password: string }) =>
  api.post('/user/login', data)

export const logout = () =>
  api.get('/user/logout')