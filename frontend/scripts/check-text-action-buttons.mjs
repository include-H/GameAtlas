import { readdir, readFile } from 'node:fs/promises'
import path from 'node:path'
import process from 'node:process'

const ROOT_DIR = path.resolve(import.meta.dirname, '..')
const SRC_DIR = path.join(ROOT_DIR, 'src')

async function collectVueFiles(dir) {
  const entries = await readdir(dir, { withFileTypes: true })
  const files = []

  for (const entry of entries) {
    const entryPath = path.join(dir, entry.name)

    if (entry.isDirectory()) {
      files.push(...await collectVueFiles(entryPath))
      continue
    }

    if (entry.isFile() && entry.name.endsWith('.vue')) {
      files.push(entryPath)
    }
  }

  return files
}

function getLineNumber(source, index) {
  return source.slice(0, index).split('\n').length
}

function findMissingTextActionButtons(source) {
  const missing = []
  const tagPattern = /<a-button\b[\s\S]*?>/g

  for (const match of source.matchAll(tagPattern)) {
    const tag = match[0]
    const index = match.index ?? 0
    const isTextButton = /\btype\s*=\s*["']text["']/.test(tag)
    const hasTextActionClass = /app-text-action-btn/.test(tag)

    if (isTextButton && !hasTextActionClass) {
      missing.push({
        line: getLineNumber(source, index),
        snippet: tag.replace(/\s+/g, ' ').trim(),
      })
    }
  }

  return missing
}

async function main() {
  const vueFiles = await collectVueFiles(SRC_DIR)
  const violations = []

  for (const file of vueFiles) {
    const source = await readFile(file, 'utf8')
    const missingButtons = findMissingTextActionButtons(source)

    for (const item of missingButtons) {
      violations.push({
        file,
        line: item.line,
        snippet: item.snippet,
      })
    }
  }

  if (violations.length === 0) {
    console.log('Text action button policy check passed.')
    return
  }

  console.error('Found text buttons missing `app-text-action-btn`:')

  for (const violation of violations) {
    const relativePath = path.relative(ROOT_DIR, violation.file)
    console.error(`- ${relativePath}:${violation.line}`)
    console.error(`  ${violation.snippet}`)
  }

  process.exitCode = 1
}

main().catch((error) => {
  console.error('Failed to run text action button policy check.')
  console.error(error)
  process.exitCode = 1
})
