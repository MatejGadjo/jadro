/* jadro.mk service worker — Web Push only (no offline caching).
 * Receives the daily "Денес во Македонија" push and opens the story on tap. */

self.addEventListener('install', () => {
  // Activate this worker immediately rather than waiting for old tabs to close.
  self.skipWaiting()
})

self.addEventListener('activate', (event) => {
  event.waitUntil(self.clients.claim())
})

self.addEventListener('push', (event) => {
  let data = {}
  try {
    data = event.data ? event.data.json() : {}
  } catch (_e) {
    data = {}
  }

  const title = data.title || 'Јадро'
  const options = {
    body: data.body || '',
    icon: data.icon || '/favicon-32x32.png',
    badge: '/favicon-32x32.png',
    image: data.image || undefined,
    tag: data.tag || 'jadro-daily',
    renotify: true,
    data: { url: data.url || '/' },
  }

  event.waitUntil(self.registration.showNotification(title, options))
})

self.addEventListener('notificationclick', (event) => {
  event.notification.close()
  const target = (event.notification.data && event.notification.data.url) || '/'

  event.waitUntil(
    self.clients.matchAll({ type: 'window', includeUncontrolled: true }).then((clients) => {
      // Focus an existing tab if one is already open, else open a new one.
      for (const client of clients) {
        if ('focus' in client) {
          client.navigate(target)
          return client.focus()
        }
      }
      return self.clients.openWindow(target)
    }),
  )
})
