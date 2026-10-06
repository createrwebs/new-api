import { describe, it, expect, vi, beforeEach } from 'vitest'
import {
  sanitizeEventParams,
  trackPageView,
  initGA4,
} from '../analytics'

describe('analytics / GA4 tracking', () => {
  beforeEach(() => {
    vi.clearAllMocks()
    Object.defineProperty(window, 'location', {
      value: {
        hostname: 'www.toraapi.com',
        href: 'https://www.toraapi.com/news',
      },
      writable: true,
    })
    window.gtag = vi.fn()
    window.dataLayer = []
  })

  it('sanitizes event params dropping sensitive credentials and tokens', () => {
    const raw = {
      token: 'secret-token-123',
      password: 'password123',
      prompt: 'hello world',
      user_id: 42,
      page_path: '/news/ai-models',
      item_count: 5,
    }
    const clean = sanitizeEventParams(raw)
    expect(clean).toEqual({
      page_path: '/news/ai-models',
      item_count: 5,
    })
    expect(clean).not.toHaveProperty('token')
    expect(clean).not.toHaveProperty('password')
    expect(clean).not.toHaveProperty('prompt')
    expect(clean).not.toHaveProperty('user_id')
  })

  it('tracks page view with sanitized parameters and avoids immediate duplicates', () => {
    trackPageView('/test-page-1')
    expect(window.gtag).toHaveBeenCalledTimes(1)
    expect(window.gtag).toHaveBeenCalledWith('event', 'page_view', expect.objectContaining({
      page_path: '/test-page-1',
    }))

    // Duplicate call with same path should be skipped
    trackPageView('/test-page-1')
    expect(window.gtag).toHaveBeenCalledTimes(1)

    // Different path should track
    trackPageView('/test-page-2?query=test')
    expect(window.gtag).toHaveBeenCalledTimes(2)
  })

  it('initializes GA4 correctly', () => {
    initGA4('G-TEST123456')
    expect(window.gtag).toHaveBeenCalledWith('config', 'G-TEST123456', { send_page_view: false })
  })
})
