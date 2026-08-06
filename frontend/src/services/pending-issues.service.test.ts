import { beforeEach, describe, expect, it, vi } from 'vitest'

const { getMock } = vi.hoisted(() => ({
  getMock: vi.fn(),
}))

vi.mock('./api', () => ({
  get: getMock,
}))

import pendingIssuesService from './pending-issues.service'

describe('pending issues service', () => {
  beforeEach(() => {
    getMock.mockReset()
  })

  it('caches successful catalogs and retries failed requests', async () => {
    const catalog = {
      groups: [{ key: 'missing-assets', label: '缺素材', description: '缺少素材' }],
      details: [{ key: 'missing-cover', label: '缺封面', group: 'missing-assets' }],
    }
    getMock
      .mockRejectedValueOnce(new Error('network'))
      .mockResolvedValueOnce({ data: catalog })

    await expect(pendingIssuesService.getCatalog()).rejects.toThrow('network')
    expect(getMock).toHaveBeenCalledTimes(1)

    const first = pendingIssuesService.getCatalog()
    const second = pendingIssuesService.getCatalog()
    expect(second).toBe(first)
    await expect(first).resolves.toEqual(catalog)
    expect(getMock).toHaveBeenCalledTimes(2)

    await expect(pendingIssuesService.getCatalog()).resolves.toEqual(catalog)
    expect(getMock).toHaveBeenCalledTimes(2)
  })
})
