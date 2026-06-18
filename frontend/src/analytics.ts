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

  // Cookieless via Consent Mode: analytics runs with no first-party cookie
  // (sends cookieless pings, no consent banner required), ad signals fully off.
  window.gtag('consent', 'default', {
    ad_storage: 'denied',
    ad_user_data: 'denied',
    ad_personalization: 'denied',
    analytics_storage: 'denied',
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
