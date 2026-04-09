const { app, BrowserWindow, dialog } = require('electron')
const http = require('node:http')
const net = require('node:net')
const fs = require('node:fs')
const path = require('node:path')
const { spawn } = require('node:child_process')

const HOST = '127.0.0.1'
const PORT_START = 31880
const PORT_END = 31920

let mainWindow = null
let backendProcess = null
let backendBaseURL = ''
let isQuitting = false

function isDevelopment() {
  return !app.isPackaged
}

function ensureDir(dirPath) {
  fs.mkdirSync(dirPath, { recursive: true })
}

function getFrontendEntry() {
  if (isDevelopment()) {
    return path.join(app.getAppPath(), 'frontend', 'dist', 'index.html')
  }
  return path.join(process.resourcesPath, 'frontend', 'dist', 'index.html')
}

function resolveBackendBinaryName() {
  const extension = process.platform === 'win32' ? '.exe' : ''
  return `nats-ui-backend-${process.platform}-${process.arch}${extension}`
}

function getBackendBinaryPath() {
  if (isDevelopment()) {
    return path.join(app.getAppPath(), 'desktop', 'bin', resolveBackendBinaryName())
  }
  return path.join(process.resourcesPath, 'bin', resolveBackendBinaryName())
}

function getBackendDataDir() {
  return path.join(app.getPath('userData'), 'backend')
}

function getBackendLogPath() {
  return path.join(getBackendDataDir(), 'backend.log')
}

function checkFileExists(filePath, label) {
  if (!fs.existsSync(filePath)) {
    throw new Error(`${label} 不存在: ${filePath}`)
  }
}

function isPortAvailable(port) {
  return new Promise((resolve) => {
    const server = net.createServer()

    server.once('error', () => resolve(false))
    server.once('listening', () => {
      server.close(() => resolve(true))
    })

    server.listen(port, HOST)
  })
}

async function pickAvailablePort() {
  for (let port = PORT_START; port <= PORT_END; port += 1) {
    if (await isPortAvailable(port)) {
      return port
    }
  }
  throw new Error(`无法在 ${PORT_START}-${PORT_END} 之间找到可用端口`)
}

function probeBackend(url) {
  return new Promise((resolve, reject) => {
    const req = http.get(url, (res) => {
      const ok = res.statusCode && res.statusCode < 500
      res.resume()
      if (ok) {
        resolve()
        return
      }
      reject(new Error(`backend probe failed: ${res.statusCode}`))
    })

    req.on('error', reject)
    req.setTimeout(1500, () => {
      req.destroy(new Error('backend probe timeout'))
    })
  })
}

async function waitForBackendReady(baseURL) {
  const probeURL = `${baseURL}/api/v1/connections`
  const startedAt = Date.now()
  let lastError = null

  while (Date.now() - startedAt < 15000) {
    try {
      await probeBackend(probeURL)
      return
    } catch (error) {
      lastError = error
      await new Promise((resolve) => setTimeout(resolve, 400))
    }
  }

  throw lastError || new Error('backend startup timed out')
}

function createBackendLogStream() {
  const logPath = getBackendLogPath()
  return fs.createWriteStream(logPath, { flags: 'a' })
}

async function startBackend() {
  if (backendProcess) {
    return backendBaseURL
  }

  ensureDir(getBackendDataDir())

  const frontendEntry = getFrontendEntry()
  const backendBinaryPath = getBackendBinaryPath()

  checkFileExists(frontendEntry, '前端构建文件')
  checkFileExists(backendBinaryPath, '桌面端后端二进制')

  const port = await pickAvailablePort()
  backendBaseURL = `http://${HOST}:${port}`

  const env = {
    ...process.env,
    HTTP_ADDR: `${HOST}:${port}`,
    NATS_CONNECTION_STORE: path.join(getBackendDataDir(), 'connections.json'),
    NATS_SECRET_KEY_FILE: path.join(getBackendDataDir(), 'secret.key'),
  }

  const logStream = createBackendLogStream()
  backendProcess = spawn(backendBinaryPath, {
    cwd: path.dirname(backendBinaryPath),
    env,
    stdio: ['ignore', 'pipe', 'pipe'],
  })

  backendProcess.stdout.on('data', (chunk) => {
    process.stdout.write(chunk)
    logStream.write(chunk)
  })

  backendProcess.stderr.on('data', (chunk) => {
    process.stderr.write(chunk)
    logStream.write(chunk)
  })

  backendProcess.once('exit', (code, signal) => {
    backendProcess = null
    if (!logStream.closed) {
      logStream.end()
    }
    if (!isQuitting) {
      dialog.showErrorBox(
        'NATS UI 后端已退出',
        `桌面端内置后端已退出，code=${code ?? 'null'} signal=${signal ?? 'null'}。\n日志文件: ${getBackendLogPath()}`,
      )
    }
  })

  await waitForBackendReady(backendBaseURL)
  return backendBaseURL
}

function stopBackend() {
  if (!backendProcess) {
    return
  }

  const child = backendProcess
  backendProcess = null

  if (process.platform === 'win32') {
    spawn('taskkill', ['/pid', String(child.pid), '/t', '/f'])
    return
  }

  child.kill('SIGTERM')
}

async function createMainWindow() {
  const apiBaseURL = await startBackend()

  mainWindow = new BrowserWindow({
    width: 1440,
    height: 920,
    minWidth: 1080,
    minHeight: 720,
    show: false,
    backgroundColor: '#f5f7fb',
    autoHideMenuBar: true,
    webPreferences: {
      preload: path.join(__dirname, 'preload.cjs'),
      contextIsolation: true,
      nodeIntegration: false,
      additionalArguments: [`--nats-ui-api-base-url=${apiBaseURL}/api/v1`],
    },
  })

  mainWindow.once('ready-to-show', () => {
    mainWindow.show()
  })

  mainWindow.on('closed', () => {
    mainWindow = null
  })

  await mainWindow.loadFile(getFrontendEntry())
}

app.whenReady().then(async () => {
  try {
    await createMainWindow()
  } catch (error) {
    dialog.showErrorBox('NATS UI 启动失败', String(error?.message || error))
    app.quit()
    return
  }

  app.on('activate', async () => {
    if (BrowserWindow.getAllWindows().length === 0) {
      await createMainWindow()
    }
  })
})

app.on('window-all-closed', () => {
  if (process.platform !== 'darwin') {
    app.quit()
  }
})

app.on('before-quit', () => {
  isQuitting = true
  stopBackend()
})
