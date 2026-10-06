/*
Copyright (C) 2023-2026 QuantumNous

This program is free software: you can redistribute it and/or modify
it under the terms of the GNU Affero General Public License as
published by the Free Software Foundation, either version 3 of the
License, or (at your option) any later version.

This program is distributed in the hope that it will be useful,
but WITHOUT ANY WARRANTY; without even the implied warranty of
MERCHANTABILITY or FITNESS FOR A PARTICULAR PURPOSE. See the
GNU Affero General Public License for more details.

You should have received a copy of the GNU Affero General Public License
along with this program. If not, see <https://www.gnu.org/licenses/>.

For commercial licensing, please contact support@quantumnous.com
*/

/**
 * GA4 & Growth Attribution Client for Tora AI.
 *
 * Implements strict privacy rules (Section 3):
 * - Never sends prompts, completions, tokens, keys, passwords, emails, names, or raw IDs.
 * - Filters developer and staging traffic to avoid polluting production analytics (Section 16).
 * - Enforces single-dispatch per navigation to eliminate double-pageview counting (Section 18).
 */

declare global {
  interface Window {
    dataLayer?: unknown[]
    gtag?: (...args: unknown[]) => void
  }
}

// Patterns of sensitive keys that MUST be redacted or dropped before sending to GA4
const SENSITIVE_KEY_PATTERNS = [
  /email/i,
  /password/i,
  /token/i,
  /secret/i,
  /key/i,
  /prompt/i,
  /response/i,
  /completion/i,
  /chat_?content/i,
  /user_?id/i,
  /card/i,
  /byok/i,
  /auth/i,
]

let currentMeasurementId: string | null = null
let isInitialized = false

export function isGA4Initialized(): boolean {
  return isInitialized
}

/**
 * Returns true if analytics tracking is authorized on the current hostname.
 * Silently drops telemetry on localhost, private IPs, and staging environments (Section 16).
 */
export function isTrackingAllowed(): boolean {
  if (typeof window === 'undefined') return false
  const host = window.location.hostname
  if (
    host === 'localhost' ||
    host === '127.0.0.1' ||
    host.endsWith('.local') ||
    host.includes('staging')
  ) {
    return false
  }
  return true
}

/**
 * Sanitizes event parameters, dropping any keys matching sensitive patterns (Section 3).
 */
export function sanitizeEventParams(
  params?: Record<string, unknown>
): Record<string, unknown> {
  if (!params) return {}
  const clean: Record<string, unknown> = {}

  for (const [k, v] of Object.entries(params)) {
    if (v === undefined || v === null) continue
    const isSensitiveKey = SENSITIVE_KEY_PATTERNS.some((pat) => pat.test(k))
    if (isSensitiveKey) {
      continue
    }

    // Drop string values that look like tokens or emails
    if (typeof v === 'string') {
      if (v.includes('@') && v.includes('.')) {
        continue // Drop email-like string
      }
      if (v.length > 50 && (v.startsWith('sk-') || v.startsWith('ey'))) {
        continue // Drop token-like string
      }
      clean[k] = v.substring(0, 150)
    } else if (
      typeof v === 'number' ||
      typeof v === 'boolean'
    ) {
      clean[k] = v
    }
  }

  return clean
}

/**
 * Initializes GA4 with the provided Measurement ID if not already initialized.
 * Injects gtag.js with { send_page_view: false } to prevent automatic double counting (Section 18).
 */
export const DEFAULT_GA4_MEASUREMENT_ID = 'G-HL2E9QVEBR'

export function initGA4(measurementId?: string): void {
  if (typeof window === 'undefined') return
  if (!isTrackingAllowed()) return

  const targetId =
    measurementId?.trim() ||
    (typeof window !== 'undefined' &&
      (window as unknown as { __GA4_ID__?: string }).__GA4_ID__) ||
    DEFAULT_GA4_MEASUREMENT_ID

  if (!targetId || targetId === currentMeasurementId) return
  currentMeasurementId = targetId

  // Ensure window.dataLayer & gtag function exist
  window.dataLayer = window.dataLayer || []
  if (typeof window.gtag !== 'function') {
    window.gtag = function (...args: unknown[]) {
      window.dataLayer?.push(args)
    }
  }

  // Load gtag script if not already in document
  const existingScript = document.querySelector(
    `script[src*="googletagmanager.com/gtag/js?id=${targetId}"]`
  )
  if (!existingScript) {
    const script = document.createElement('script')
    script.async = true
    script.src = `https://www.googletagmanager.com/gtag/js?id=${targetId}`
    document.head.appendChild(script)
  }

  window.gtag('js', new Date())
  // Critical for SPA: send_page_view: false prevents auto-tracking collision
  window.gtag('config', targetId, { send_page_view: false })
  isInitialized = true
}

let lastTrackedPath = ''

/**
 * Tracks a page_view event explicitly for SPA route transitions (Section 4 & 18).
 * Guards against duplicate reporting on unchanged paths.
 */
export function trackPageView(
  pagePath: string,
  pageTitle?: string,
  customParams?: Record<string, unknown>
): void {
  if (typeof window === 'undefined') return
  if (!isTrackingAllowed()) return

  // Prevent double-counting the exact same pathname/query consecutively
  if (pagePath === lastTrackedPath) {
    return
  }
  lastTrackedPath = pagePath

  if (typeof window.gtag === 'function') {
    const payload = sanitizeEventParams({
      page_path: pagePath,
      page_title: pageTitle || document.title,
      page_location: window.location.href,
      ...customParams,
    })
    window.gtag('event', 'page_view', payload)
  }
}

/**
 * Generic privacy-safe event dispatcher.
 */
export function trackEvent(
  eventName: string,
  params?: Record<string, unknown>
): void {
  if (typeof window === 'undefined') return
  if (!isTrackingAllowed()) return

  if (typeof window.gtag === 'function') {
    const cleanParams = sanitizeEventParams(params)
    window.gtag('event', eventName, cleanParams)
  }
}

// ==========================================
// Product Funnel Events (Section 7)
// ==========================================

export function trackSignUp(method = 'credentials'): void {
  trackEvent('sign_up', { method })
}

export function trackLogin(method = 'credentials'): void {
  trackEvent('login', { method })
}

export function trackPricingView(): void {
  trackEvent('pricing_view', { page_type: 'pricing' })
}

export function trackChatStarted(): void {
  trackEvent('chat_started', { feature: 'playground' })
}

let hasTrackedFirstChat = false
export function trackFirstSuccessfulChat(): void {
  if (hasTrackedFirstChat) return
  try {
    const stored = localStorage.getItem('tora_chat_first_done')
    if (!stored) {
      localStorage.setItem('tora_chat_first_done', 'true')
      hasTrackedFirstChat = true
      trackEvent('first_successful_chat', { feature: 'playground' })
    }
  } catch {
    hasTrackedFirstChat = true
    trackEvent('first_successful_chat', { feature: 'playground' })
  }
}

export function trackCheckoutStarted(paymentMethod = 'stripe'): void {
  trackEvent('begin_checkout', {
    payment_method: paymentMethod,
    currency: 'THB',
  })
}

export function trackSubscriptionSuccess(): void {
  trackEvent('purchase', {
    currency: 'THB',
    affiliation: 'Tora AI Web',
  })
}

// ==========================================
// News CTA & Interaction Events (Section 6)
// ==========================================

export function trackNewsCTAClick(
  sourcePostId: number,
  ctaType: string,
  destinationType: string
): void {
  trackEvent('news_cta_click', {
    source_post_id: sourcePostId,
    cta_type: ctaType,
    destination_type: destinationType,
  })
}

export function trackNewsInternalLinkClick(
  sourcePostId: number,
  anchorText: string,
  destination: string
): void {
  trackEvent('news_internal_link_click', {
    source_post_id: sourcePostId,
    anchor_text: anchorText.substring(0, 80),
    destination,
  })
}

// ==========================================
// Studio Conversion Funnel Events (Queue 3)
// ==========================================

export function trackStudioInsufficientCredit(
  toolId: string,
  requiredCredits: number,
  currentCredits: number
): void {
  trackEvent('studio_insufficient_credit', {
    tool_id: toolId,
    required_credits: requiredCredits,
    current_credits: currentCredits,
    missing_credits: Math.max(0, requiredCredits - currentCredits),
  })
}

export function trackStudioBuyCreditClick(
  toolId: string,
  requiredCredits: number
): void {
  trackEvent('studio_buy_credit_click', {
    tool_id: toolId,
    required_credits: requiredCredits,
  })
}

export function trackStudioPurchaseReturn(toolId: string): void {
  trackEvent('studio_purchase_return', {
    tool_id: toolId,
  })
}

export function trackStudioGenerationAfterPurchase(
  toolId: string,
  credits: number
): void {
  trackEvent('studio_generation_after_purchase', {
    tool_id: toolId,
    credits,
  })
}

