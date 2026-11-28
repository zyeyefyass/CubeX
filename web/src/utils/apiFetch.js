// Unified lightweight API fetch wrapper for CubeX Web Console

const clientConfig = {
  onUnauthorized: null,
  onError: null,
  defaultTimeoutMs: 15000,
}

export function configureApiClient({ onUnauthorized, onError, defaultTimeoutMs } = {}) {
  if (typeof onUnauthorized === 'function') clientConfig.onUnauthorized = onUnauthorized
  if (typeof onError === 'function') clientConfig.onError = onError
  if (typeof defaultTimeoutMs === 'number' && defaultTimeoutMs > 0) {
    clientConfig.defaultTimeoutMs = defaultTimeoutMs
  }
}

export async function apiFetch(url, options = {}) {
  const {
    timeoutMs = clientConfig.defaultTimeoutMs,
    skipWhenHidden = false,
    notifyError = false,
    json: jsonBody,
    headers: rawHeaders = {},
    ...fetchOpts
  } = options

  if (skipWhenHidden && typeof document !== 'undefined' && document.visibilityState === 'hidden') {
    return new Response(JSON.stringify({ ok: false, hidden: true }), {
      status: 200,
      headers: { 'Content-Type': 'application/json' },
    })
  }

  const headers = { ...rawHeaders }
  let body = fetchOpts.body
  if (jsonBody !== undefined) {
    if (!headers['Content-Type']) headers['Content-Type'] = 'application/json'
    body = JSON.stringify(jsonBody)
  }

  let controller = null
  let timer = null
  let signal = fetchOpts.signal
  if (!signal && timeoutMs > 0 && typeof AbortController !== 'undefined') {
    controller = new AbortController()
    signal = controller.signal
    timer = setTimeout(() => controller.abort(), timeoutMs)
  }

  const requester = (typeof window !== 'undefined' && window.__cubexNativeFetch)
    ? window.__cubexNativeFetch
    : fetch

  try {
    const res = await requester(url, { ...fetchOpts, headers, body, signal })
    if (res.status === 401 && !String(url).includes('/api/v1/auth/') && clientConfig.onUnauthorized) {
      try { clientConfig.onUnauthorized() } catch (_) {}
    }
    if (!res.ok && notifyError && clientConfig.onError) {
      clientConfig.onError(`请求失败 (${res.status})`)
    }
    return res
  } catch (err) {
    if (notifyError && clientConfig.onError && err?.name !== 'AbortError') {
      clientConfig.onError(err?.message || '网络请求异常')
    }
    throw err
  } finally {
    if (timer) clearTimeout(timer)
  }
}

export async function apiJson(url, options = {}) {
  const res = await apiFetch(url, options)
  const data = await res.json().catch(() => null)
  return { ok: res.ok, status: res.status, data, response: res }
}
