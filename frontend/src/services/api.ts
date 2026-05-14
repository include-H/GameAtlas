import axios, { type AxiosInstance, type AxiosRequestConfig, type AxiosError } from 'axios'

// Axios requests read the API base from this file only.
// Non-axios URLs such as download href/action must use buildApiUrl().
export const API_BASE_URL = import.meta.env.VITE_API_BASE_URL || '/api'

// Create axios instance
const apiClient: AxiosInstance = axios.create({
  baseURL: API_BASE_URL,
  withCredentials: true,
  timeout: 30000,
  headers: {
    'Content-Type': 'application/json',
  },
})

// Response interceptor
apiClient.interceptors.response.use(
  (response) => response,
  (error: AxiosError) => {
    if (error.response) {
      switch (error.response.status) {
        case 404:
          // 2026-05-13: GET 404s from page-level API calls redirect to the not-found page.
          // Non-GET requests (POST/PUT/DELETE) and non-page resources are rejected normally.
          if (error.config?.method === 'get') {
            import('@/router').then(({ default: router }) => {
              if (router.currentRoute.value.name !== 'not-found') {
                router.replace({ name: 'not-found' })
              }
            })
          }
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
const request = async <T = unknown>(config: AxiosRequestConfig): Promise<T> => {
  const response = await apiClient.request<T>(config)
  return response.data
}

// HTTP method helpers
export const get = <T = unknown>(url: string, config?: AxiosRequestConfig): Promise<T> => {
  return request<T>({ ...config, method: 'GET', url })
}

export const post = <T = unknown, D = unknown>(url: string, data?: D, config?: AxiosRequestConfig<D>): Promise<T> => {
  return request<T>({ ...config, method: 'POST', url, data })
}

export const put = <T = unknown, D = unknown>(url: string, data?: D, config?: AxiosRequestConfig<D>): Promise<T> => {
  return request<T>({ ...config, method: 'PUT', url, data })
}

export const del = <T = unknown>(url: string, config?: AxiosRequestConfig): Promise<T> => {
  return request<T>({ ...config, method: 'DELETE', url })
}

export default apiClient
