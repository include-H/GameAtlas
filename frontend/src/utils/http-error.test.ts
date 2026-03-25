import axios, { AxiosError } from 'axios'
import { describe, expect, it } from 'vitest'

import { getHttpErrorData, getHttpErrorMessage, getHttpStatus } from './http-error'

describe('http-error helpers', () => {
  it('returns fallback values for non-axios errors', () => {
    const error = new Error('plain error')

    expect(getHttpStatus(error)).toBe(0)
    expect(getHttpErrorData(error)).toBeUndefined()
    expect(getHttpErrorMessage(error)).toBe('plain error')
  })

  it('extracts status data and message from axios errors', () => {
    const error = new AxiosError(
      'request failed',
      'ERR_BAD_REQUEST',
      undefined,
      undefined,
      {
        status: 400,
        statusText: 'Bad Request',
        headers: {},
        config: { headers: axios.AxiosHeaders.from({}) },
        data: {
          error: '接口失败',
          data: {
            field: 'title',
          },
        },
      },
    )

    expect(getHttpStatus(error)).toBe(400)
    expect(getHttpErrorData<{ field: string }>(error)).toEqual({ field: 'title' })
    expect(getHttpErrorMessage(error)).toBe('接口失败')
  })

  it('falls back to response message and custom fallback when needed', () => {
    const responseMessageError = new AxiosError(
      'request failed',
      'ERR_BAD_REQUEST',
      undefined,
      undefined,
      {
        status: 422,
        statusText: 'Unprocessable Entity',
        headers: {},
        config: { headers: axios.AxiosHeaders.from({}) },
        data: {
          message: '字段校验失败',
        },
      },
    )

    const emptyAxiosError = new AxiosError('')

    expect(getHttpErrorMessage(responseMessageError)).toBe('字段校验失败')
    expect(getHttpErrorMessage(emptyAxiosError, '兜底提示')).toBe('兜底提示')
  })
})
