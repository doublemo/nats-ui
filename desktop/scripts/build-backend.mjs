import { mkdirSync } from 'node:fs'
import path from 'node:path'
import { fileURLToPath } from 'node:url'
import { spawnSync } from 'node:child_process'

const __dirname = path.dirname(fileURLToPath(import.meta.url))
const projectRoot = path.resolve(__dirname, '..', '..')
const outputDir = path.join(projectRoot, 'desktop', 'bin')

const DEFAULT_TARGETS = [
  { platform: 'win32', arch: 'x64' },
  { platform: 'darwin', arch: 'x64' },
  { platform: 'darwin', arch: 'arm64' },
  { platform: 'linux', arch: 'x64' },
  { platform: 'linux', arch: 'arm64' },
]

const NODE_TO_GOOS = {
  win32: 'windows',
  darwin: 'darwin',
  linux: 'linux',
}

const NODE_TO_GOARCH = {
  x64: 'amd64',
  arm64: 'arm64',
}

function parseTargets(argv) {
  const targets = []

  for (let index = 0; index < argv.length; index += 1) {
    const token = argv[index]

    if (token === '--all') {
      return DEFAULT_TARGETS
    }

    if (token === '--target') {
      const value = argv[index + 1]
      if (value) {
        const [platform, arch] = value.split(':')
        targets.push({ platform, arch })
        index += 1
      }
      continue
    }

    if (token.startsWith('--target=')) {
      const value = token.slice('--target='.length)
      const [platform, arch] = value.split(':')
      targets.push({ platform, arch })
    }
  }

  if (targets.length > 0) {
    return targets
  }

  return [{ platform: process.platform, arch: process.arch }]
}

function validateTarget(target) {
  if (!NODE_TO_GOOS[target.platform]) {
    throw new Error(`unsupported platform: ${target.platform}`)
  }
  if (!NODE_TO_GOARCH[target.arch]) {
    throw new Error(`unsupported arch: ${target.arch}`)
  }
}

function outputNameFor(target) {
  const extension = target.platform === 'win32' ? '.exe' : ''
  return `nats-ui-backend-${target.platform}-${target.arch}${extension}`
}

function buildTarget(target) {
  validateTarget(target)

  const goos = NODE_TO_GOOS[target.platform]
  const goarch = NODE_TO_GOARCH[target.arch]
  const outputPath = path.join(outputDir, outputNameFor(target))

  console.log(`building backend for ${target.platform}/${target.arch} -> ${outputPath}`)

  const result = spawnSync(
    'go',
    ['build', '-buildvcs=false', '-o', outputPath, './cmd/server'],
    {
      cwd: projectRoot,
      stdio: 'inherit',
      env: {
        ...process.env,
        CGO_ENABLED: '0',
        GOOS: goos,
        GOARCH: goarch,
        GOCACHE: path.join(projectRoot, '.gocache'),
      },
    },
  )

  if (result.status !== 0) {
    throw new Error(`go build failed for ${target.platform}/${target.arch}`)
  }
}

function main() {
  const targets = parseTargets(process.argv.slice(2))

  mkdirSync(outputDir, { recursive: true })

  for (const target of targets) {
    buildTarget(target)
  }
}

main()
