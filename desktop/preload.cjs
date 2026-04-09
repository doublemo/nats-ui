const { contextBridge } = require('electron')

const prefix = '--nats-ui-api-base-url='
const apiBaseURL = process.argv.find((item) => item.startsWith(prefix))?.slice(prefix.length) || 'http://127.0.0.1:31880/api/v1'

contextBridge.exposeInMainWorld('__NATS_UI_DESKTOP__', Object.freeze({
  apiBaseUrl: apiBaseURL,
  isElectron: true,
  platform: process.platform,
}))
