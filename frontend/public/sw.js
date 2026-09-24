// 离线稍后读：缓存已访问的页面与静态资源，收藏的文章离线可看。
// 请求优先从缓存返回（Stale-while-revalidate），后台更新缓存
const CACHE_NAME = 'qs-v1'

self.addEventListener('install', () => { self.skipWaiting() })

self.addEventListener('activate', (event) => {
  event.waitUntil(
    caches.keys().then((names) => Promise.all(names.map((name) => name !== CACHE_NAME ? caches.delete(name) : undefined)))
  )
  self.clients.claim()
})

self.addEventListener('fetch', (event) => {
  const { request } = event
  if (request.method !== 'GET') return
  // 仅缓存同源请求、导航请求（HTML）与静态资源
  if (!request.url.startsWith(self.location.origin)) return

  event.respondWith(
    caches.open(CACHE_NAME).then(async (cache) => {
      const cached = await cache.match(request)
      const fetched = fetch(request).catch(() => cached)
      if (fetched) {
        fetched.then((response) => {
          if (response.ok && (response.headers.get('Content-Type')?.includes('text/html') || /.(js|css|png|jpg|svg|woff2|webp)$/.test(request.url))) {
            cache.put(request, response.clone())
          }
        })
      }
      return cached || fetched
    })
  )
})
