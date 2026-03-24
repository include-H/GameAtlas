import axios from 'axios'

export interface HttpErrorEnvelope<TData = unknown> {
  success?: boolean
  error?: string
  message?: string
  data?: TData
}

export const getHttpStatus = (error: unknown): number => {
  if (!axios.isAxiosError(error)) {
    return 0
  }
  return Number(error.response?.status || 0)
}

export const getHttpErrorData = <TData = unknown>(error: unknown): TData | undefined => {
  if (!axios.isAxiosError<HttpErrorEnvelope<TData>>(error)) {
    return undefined
  }
  return error.response?.data?.data
}

export const getHttpErrorMessage = (error: unknown, fallback = '未知错误') => {
  if (axios.isAxiosError<HttpErrorEnvelope>(error)) {
    return error.response?.data?.error || error.response?.data?.message || error.message || fallback
  }
  if (error instanceof Error && error.message) {
    return error.message
  }
  return fallback
}
