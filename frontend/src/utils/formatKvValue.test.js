import assert from 'node:assert/strict'
import { test } from 'node:test'
import { formatKvValue } from './formatKvValue.js'

test('formats JSON values without losing their content', () => {
  assert.deepEqual(formatKvValue('{"name":"订单","count":2}'), {
    text: '{\n  "name": "订单",\n  "count": 2\n}',
    format: 'JSON',
  })
  assert.deepEqual(formatKvValue('[1,2]'), { text: '[\n  1,\n  2\n]', format: 'JSON' })
})

test('shows plain or malformed values unchanged', () => {
  assert.deepEqual(formatKvValue('hello'), { text: 'hello', format: 'String' })
  assert.deepEqual(formatKvValue('{broken'), { text: '{broken', format: 'String' })
  assert.deepEqual(formatKvValue(''), { text: '', format: 'String' })
})

test('expands JSON stored as a string and nested payload JSON', () => {
  assert.equal(formatKvValue(JSON.stringify('{"ok":true}')).text, '{\n  "ok": true\n}')
  assert.equal(formatKvValue('{"payload":"{\\"id\\":7}"}').text, '{\n  "payload": {\n    "id": 7\n  }\n}')
})

test('expands Base64-wrapped JSON payloads while preserving unrelated text', () => {
  const encoded = Buffer.from('{"id":7}').toString('base64')
  assert.equal(formatKvValue(JSON.stringify({ payload: encoded })).text, '{\n  "payload": {\n    "id": 7\n  }\n}')
  assert.equal(formatKvValue('{"payload":"hello"}').text, '{\n  "payload": "hello"\n}')
})
