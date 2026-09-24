function parseJson(text) {
  try {
    return JSON.parse(text)
  } catch {
    return undefined
  }
}

function parseJsonPayload(text) {
  const parsed = parseJson(text)
  if (parsed && typeof parsed === 'object') return parsed
  try {
    const compact = text.replace(/\s/g, '')
    if (!/^[A-Za-z0-9+/_-]+={0,2}$/.test(compact) || compact.length % 4 === 1) return undefined
    const encoded = compact.replace(/-/g, '+').replace(/_/g, '/').padEnd(Math.ceil(compact.length / 4) * 4, '=')
    const bytes = Uint8Array.from(atob(encoded), (character) => character.charCodeAt(0))
    const decoded = new TextDecoder('utf-8', { fatal: true }).decode(bytes)
    const inner = parseJson(decoded)
    return inner && typeof inner === 'object' ? inner : undefined
  } catch {
    return undefined
  }
}

function expandPayload(value, depth = 0) {
  if (!value || typeof value !== 'object' || depth >= 4) return value
  if (Array.isArray(value)) return value.map((item) => expandPayload(item, depth + 1))
  return Object.fromEntries(Object.entries(value).map(([key, item]) => {
    if (key.toLowerCase() === 'payload' && typeof item === 'string') {
      const parsed = parseJsonPayload(item)
      if (parsed !== undefined) return [key, expandPayload(parsed, depth + 1)]
    }
    return [key, expandPayload(item, depth + 1)]
  }))
}

export function formatKvValue(value) {
  const raw = String(value ?? '')
  let parsed = parseJson(raw)
  if (parsed === undefined) parsed = parseJsonPayload(raw)
  if (parsed === undefined) return { text: raw, format: 'String' }

  // Some producers store the whole JSON document as a JSON string.
  for (let i = 0; i < 3 && typeof parsed === 'string'; i += 1) {
    const inner = parseJson(parsed)
    if (inner === undefined) break
    parsed = inner
  }
  return { text: JSON.stringify(expandPayload(parsed), null, 2), format: 'JSON' }
}
