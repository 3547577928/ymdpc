import { afterEach, describe, expect, it, vi } from 'vitest'
import { AUTH_EXPIRED_EVENT, getCurrentUser, getPosts } from './api'

function mockResponse(status: number, payload: unknown): Response {
  return { ok: status >= 200 && status < 300, status, json: async () => payload } as Response
}

describe('api session handling', () => {
  afterEach(() => {
    vi.unstubAllGlobals()
  })

  it('broadcasts session expiry for protected requests', async () => {
    const fetchMock = vi.fn().mockResolvedValue(mockResponse(401, { code: 401, message: '登录已过期' }))
    vi.stubGlobal('fetch', fetchMock)
    const onExpired = vi.fn()
    window.addEventListener(AUTH_EXPIRED_EVENT, onExpired)

    await expect(getPosts()).rejects.toThrow('登录已过期')

    expect(fetchMock).toHaveBeenCalledOnce()
    expect(onExpired).toHaveBeenCalledOnce()
    window.removeEventListener(AUTH_EXPIRED_EVENT, onExpired)
  })

  it('does not redirect for the expected unauthenticated current-user check', async () => {
    vi.stubGlobal('fetch', vi.fn().mockResolvedValue(mockResponse(401, { code: 401, message: '未登录' })))
    const onExpired = vi.fn()
    window.addEventListener(AUTH_EXPIRED_EVENT, onExpired)

    await expect(getCurrentUser()).rejects.toThrow('未登录')

    expect(onExpired).not.toHaveBeenCalled()
    window.removeEventListener(AUTH_EXPIRED_EVENT, onExpired)
  })
})
