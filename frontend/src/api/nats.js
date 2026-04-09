import axios from 'axios'

const ACTIVE_CONNECTION_KEY = 'nats-ui-active-connection'
const CONNECTION_EVENT = 'nats-ui-connection-changed'
const API_BASE_URL = resolveAPIBaseURL()

function resolveAPIBaseURL() {
  if (typeof window !== 'undefined') {
    const desktopBaseURL = window.__NATS_UI_DESKTOP__?.apiBaseUrl
    if (desktopBaseURL) {
      return desktopBaseURL
    }
  }
  return import.meta.env.VITE_API_BASE_URL || '/api/v1'
}

const http = axios.create({
  baseURL: API_BASE_URL,
  timeout: 8000,
})

http.interceptors.request.use((config) => {
  const connectionId = getActiveConnectionId()
  if (connectionId) {
    config.params = { ...(config.params || {}), connectionId }
  }
  return config
})

http.interceptors.response.use(
  (response) => response.data?.data,
  (error) => {
    const message = error.response?.data?.message || error.message || '请求失败'
    return Promise.reject(new Error(message))
  },
)

export function getActiveConnectionId() {
  return window.localStorage.getItem(ACTIVE_CONNECTION_KEY) || ''
}

export function setActiveConnectionId(connectionId) {
  if (connectionId) {
    window.localStorage.setItem(ACTIVE_CONNECTION_KEY, connectionId)
  } else {
    window.localStorage.removeItem(ACTIVE_CONNECTION_KEY)
  }
  window.dispatchEvent(new CustomEvent(CONNECTION_EVENT, { detail: connectionId }))
}

export function onConnectionChanged(listener) {
  window.addEventListener(CONNECTION_EVENT, listener)
  return () => window.removeEventListener(CONNECTION_EVENT, listener)
}

export function getConnections(params) {
  return http.get('/connections', { params })
}

export function createConnection(payload) {
  return http.post('/connections', payload)
}

export function importConnections(items) {
  return http.post('/connections/import', items)
}

export function previewImportConnections(items) {
  return http.post('/connections/import-preview', { items })
}

export function updateConnection(id, payload) {
  return http.put(`/connections/${id}`, payload)
}

export function deleteConnection(id) {
  return http.delete(`/connections/${id}`)
}

export function batchDeleteConnections(ids) {
  return http.post('/connections/batch-delete', { ids })
}

export function activateConnection(id) {
  return http.post(`/connections/${id}/activate`)
}

export function testConnection(id) {
  return http.post(`/connections/${id}/test`)
}

export function probeConnection(payload) {
  return http.post('/connections/test', payload)
}

export function discoverMonitorEndpoints(natsUrls) {
  return http.post('/connections/discover-monitors', { natsUrls })
}

export function getClusterOverview() {
  return http.get('/cluster/overview')
}

export function getStreams(params) {
  return http.get('/streams', { params })
}

export function getStreamDetail(name) {
  return http.get(`/streams/${name}`)
}

export function createStream(payload) {
  return http.post('/streams', payload)
}

export function deleteStream(name) {
  return http.delete(`/streams/${name}`)
}

export function batchDeleteStreams(names) {
  return http.post('/streams/batch-delete', { names })
}

export function getBuckets(params) {
  return http.get('/kv/buckets', { params })
}

export function createBucket(payload) {
  return http.post('/kv/buckets', payload)
}

export function deleteBucket(name) {
  return http.delete(`/kv/buckets/${name}`)
}

export function batchDeleteBuckets(names) {
  return http.post('/kv/buckets/batch-delete', { names })
}

export function getBucketEntries(name, params) {
  return http.get(`/kv/buckets/${name}/entries`, { params })
}

export function putBucketEntry(bucket, key, value) {
  return http.put(`/kv/buckets/${bucket}/entries/${key}`, { value })
}

export function deleteBucketEntry(bucket, key) {
  return http.delete(`/kv/buckets/${bucket}/entries/${key}`)
}

export function batchDeleteBucketEntries(bucket, keys) {
  return http.post(`/kv/buckets/${bucket}/entries/batch-delete`, { keys })
}
