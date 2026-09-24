const decoder = new TextDecoder('utf-8', { fatal: true })

function decodeBytes(rawBase64) {
  const binary = atob(rawBase64 || '')
  return Uint8Array.from(binary, (character) => character.charCodeAt(0))
}

function decodeText(bytes) {
  return decoder.decode(bytes)
}

function decodeInnerBase64(text) {
  const compact = text.replace(/\s/g, '').replace(/-/g, '+').replace(/_/g, '/')
  if (!compact || !/^[A-Za-z0-9+/]*={0,2}$/.test(compact) || compact.length % 4 === 1) {
    throw new Error('Invalid Base64 text')
  }
  const padded = compact.padEnd(Math.ceil(compact.length / 4) * 4, '=')
  return decodeText(decodeBytes(padded))
}

export function parseMessage(preview, mode) {
  if (mode === 'auto') return preview.preview || ''
  const raw = decodeText(decodeBytes(preview.rawBase64))
  const text = mode.startsWith('base64-') ? decodeInnerBase64(raw) : raw
  if (mode === 'string' || mode === 'base64-string') return text
  if (mode === 'json' || mode === 'base64-json') return JSON.stringify(JSON.parse(text), null, 2)
  throw new Error('Unsupported message format')
}
