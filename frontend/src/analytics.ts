const GA_ID = import.meta.env.VITE_GA_MEASUREMENT_ID

declare global {
  interface Window {
    dataLayer: unknown[]
    gtag: (...args: unknown[]) => void
  }
}

export function initAnalytics(): void {
  if (!GA_ID) return

  const script = document.createElement('script')
  script.async = true
  script.src = `https://www.googletagmanager.com/gtag/js?id=${GA_ID}`
  document.head.appendChild(script)

  window.dataLayer = window.dataLayer || []
  // Must push the `arguments` object (not a spread array) — gtag.js only
  // recognizes queued commands in that exact shape.
  window.gtag = function () {
    // eslint-disable-next-line prefer-rest-params
    window.dataLayer.push(arguments)
  }

  // Analytics-only: GA4 uses its first-party `_ga` cookie so hits actually
  // register (Realtime/DebugView/reports). All advertising signals stay off.
  // NOTE: analytics_storage 'denied' = cookieless consent-mode pings, which
  // are modeling-only and never show for a small/low-traffic property.
  window.gtag('consent', 'default', {
    ad_storage: 'denied',
    ad_user_data: 'denied',
    ad_personalization: 'denied',
    analytics_storage: 'granted',
  })
  window.gtag('js', new Date())
  window.gtag('config', GA_ID, {
    anonymize_ip: true,
  })
}

export function trackEvent(name: string, params?: Record<string, string>): void {
  if (!GA_ID || typeof window.gtag !== 'function') return
  window.gtag('event', name, params)
}
