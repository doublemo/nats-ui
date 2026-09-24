import assert from 'node:assert/strict'
import { test } from 'node:test'
import { parseMessage } from './parseMessage.js'

const preview = (body) => ({ rawBase64: Buffer.from(body).toString('base64'), preview: body })

test('formats UTF-8 strings and JSON from raw message bytes', () => {
  const message = preview('{"name":"订单"}')
  assert.equal(parseMessage(message, 'string'), '{"name":"订单"}')
  assert.equal(parseMessage(message, 'json'), '{\n  "name": "订单"\n}')
})

test('decodes Base64-wrapped string and JSON when explicitly selected', () => {
  const wrapped = preview(Buffer.from('{"ok":true}').toString('base64'))
  assert.equal(parseMessage(wrapped, 'base64-string'), '{"ok":true}')
  assert.equal(parseMessage(wrapped, 'base64-json'), '{\n  "ok": true\n}')
})

test('reports invalid formats rather than displaying Base64 as decoded content', () => {
  assert.throws(() => parseMessage(preview('plain text'), 'json'), SyntaxError)
  assert.throws(() => parseMessage(preview('not base64!'), 'base64-string'))
  assert.throws(() => parseMessage({ rawBase64: '/w==', preview: 'FF' }, 'string'))
})
