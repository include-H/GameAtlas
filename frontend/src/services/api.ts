import axios, { type AxiosInstance, type AxiosRequestConfig, type AxiosError } from 'axios'

// API configuration
const API_BASE_URL = import.meta.env.VITE_API_BASE_URL || '/api'
const mockMode = import.meta.env.VITE_MOCK_MODE === 'true' || false

// Create axios instance
const apiClient: AxiosInstance = axios.create({
  baseURL: API_BASE_URL,
  timeout: mockMode ? 1000 : 30000,  // Faster timeout in mock mode
  headers: {
    'Content-Type': 'application/json',
  },
})

// Response interceptor
apiClient.interceptors.response.use(
  (response) => response,
  (error: AxiosError) => {
    if (error.response) {
      // Handle specific error codes
      switch (error.response.status) {
        case 404:
          console.error('Not Found: The requested resource was not found')
          break
        case 500:
          console.error('Server Error: An unexpected error occurred')
          break
      }
    }
    return Promise.reject(error)
  }
)

// Generic request wrapper
export const request = async <T = any>(config: AxiosRequestConfig): Promise<T> => {
  const response = await apiClient.request<T>(config)
  return response.data
}

// HTTP method helpers
export const get = <T = any>(url: string, config?: AxiosRequestConfig): Promise<T> => {
  return request<T>({ ...config, method: 'GET', url })
}

export const post = <T = any>(url: string, data?: any, config?: AxiosRequestConfig): Promise<T> => {
  return request<T>({ ...config, method: 'POST', url, data })
}

export const put = <T = any>(url: string, data?: any, config?: AxiosRequestConfig): Promise<T> => {
  return request<T>({ ...config, method: 'PUT', url, data })
}

export const del = <T = any>(url: string, config?: AxiosRequestConfig): Promise<T> => {
  return request<T>({ ...config, method: 'DELETE', url })
}

export default apiClient
