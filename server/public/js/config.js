globalThis.TIMECRUSHER_CONFIG = { backendURL: '' };

function getBackendURL() {
  const configured = globalThis.TIMECRUSHER_CONFIG.backendURL;
  const isLocal = location.port === '8080' || location.hostname === 'localhost';
  const fallback = location.protocol + '//' + (isLocal ? location.hostname + ':8080' : location.host);
  const url = new URL(configured || fallback);
  if (!['http:', 'https:'].includes(url.protocol) || url.username || url.password ||
      url.pathname !== '/' || url.search || url.hash ||
      (location.protocol === 'https:' && url.protocol !== 'https:')) {
    throw new Error('TimeCrusher backendURL must be an HTTP(S) origin and use HTTPS on a secure page.');
  }
  return url;
}
