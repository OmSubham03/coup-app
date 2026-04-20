const CACHE_NAME = 'coup-v2';
const PRECACHE = [
  '/textures/duke.jpg',
  '/textures/assassin.jpg',
  '/textures/captain.jpg',
  '/textures/ambassador.jpg',
  '/textures/contessa.jpg',
  '/textures/inquisitor.png',
  '/textures/card-back.svg',
  '/icons/icon-192.png',
  '/icons/icon-512.png'
];

self.addEventListener('install', e => {
  e.waitUntil(
    caches.open(CACHE_NAME).then(cache => cache.addAll(PRECACHE))
  );
  self.skipWaiting();
});

self.addEventListener('activate', e => {
  e.waitUntil(
    caches.keys().then(keys =>
      Promise.all(keys.filter(k => k !== CACHE_NAME).map(k => caches.delete(k)))
    )
  );
  self.clients.claim();
});

self.addEventListener('fetch', e => {
  const url = new URL(e.request.url);
  // Only cache static assets (textures, icons) — never JS, CSS, or HTML
  if (url.pathname === '/' || url.pathname.startsWith('/ws') || url.pathname.startsWith('/api/') ||
      url.pathname.startsWith('/js/') || url.pathname.startsWith('/css/')) {
    return;
  }
  // Cache-first for static assets (textures, icons)
  e.respondWith(
    caches.match(e.request).then(cached => cached || fetch(e.request).then(res => {
      if (res.ok) {
        const clone = res.clone();
        caches.open(CACHE_NAME).then(cache => cache.put(e.request, clone));
      }
      return res;
    }))
  );
});
