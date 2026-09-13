const fs = require('fs')
const path = require('path')

const root = path.resolve(__dirname, '..')
const source = path.join(root, 'miniprogram', '.env.local')
const destination = path.join(root, 'miniprogram', 'utils', 'config.runtime.js')

if (!fs.existsSync(source)) {
  console.error('Missing miniprogram/.env.local. Copy miniprogram/.env.local.example first.')
  process.exit(1)
}

const values = {}
for (const rawLine of fs.readFileSync(source, 'utf8').split(/\r?\n/)) {
  const line = rawLine.trim()
  if (!line || line.startsWith('#')) continue
  const separator = line.indexOf('=')
  if (separator < 1) continue
  values[line.slice(0, separator).trim()] = line.slice(separator + 1).trim().replace(/^['"]|['"]$/g, '')
}

const required = ['MINIPROGRAM_ENVIRONMENT', 'MINIPROGRAM_BASE_URL', 'MINIPROGRAM_SUBSCRIBE_TEMPLATE_ID']
for (const key of required) {
  if (!values[key]) {
    console.error(`Missing ${key} in miniprogram/.env.local`)
    process.exit(1)
  }
}

const runtimeConfig = `// Generated file — do not edit directly.\nmodule.exports = ${JSON.stringify({
  environment: values.MINIPROGRAM_ENVIRONMENT,
  baseURL: values.MINIPROGRAM_BASE_URL,
  subscribeTemplateID: values.MINIPROGRAM_SUBSCRIBE_TEMPLATE_ID
}, null, 2)}\n`
fs.writeFileSync(destination, runtimeConfig)
console.log(`Generated ${path.relative(root, destination)} for ${values.MINIPROGRAM_ENVIRONMENT}`)
