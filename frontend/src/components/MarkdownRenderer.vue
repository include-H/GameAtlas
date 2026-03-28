<template>
  <div
    class="markdown-renderer"
    :class="{ 'markdown-renderer--compact': compact }"
    v-html="renderedHtml"
  />
</template>

<script setup lang="ts">
import { computed } from 'vue'
import MarkdownIt from 'markdown-it'

interface Props {
  content: string
  compact?: boolean
}

const props = withDefaults(defineProps<Props>(), {
  compact: false,
})

const md = new MarkdownIt({
  html: false,
  linkify: true,
  typographer: true,
  breaks: true,
})

const epigraphBlockPattern = /(^|\n):::epigraph[ \t]*\n([\s\S]*?)\n:::(?=\n|$)/g
const epigraphTokenPrefix = '@@EPIGRAPH_BLOCK_'

function classifyEpigraphLine(line: string): 'cn' | 'en' {
  const asciiLetters = (line.match(/[A-Za-z]/g) || []).length
  const cjkChars = (line.match(/[\u3400-\u9fff]/g) || []).length

  if (asciiLetters > cjkChars) {
    return 'en'
  }

  return 'cn'
}

function splitCnEpigraphSegments(line: string): string[] {
  const normalized = line.replace(/\s+/g, '')
  const segments = normalized
    .split(/(?<=[，。！？；：、,.!?;:])/u)
    .map((segment) => segment.trim())
    .filter((segment) => segment.length > 0)

  return segments.length > 0 ? segments : [normalized]
}

function renderCnEpigraphSegment(segment: string): string {
  const match = segment.match(/^(.*?)([，。！？；：、,.!?;:]+)?$/u)

  if (!match) {
    return md.renderInline(segment)
  }

  const content = match[1] ?? ''
  const punctuation = match[2] ?? ''
  const contentHtml = content ? `<span class="epigraph__segment-text">${md.renderInline(content)}</span>` : ''
  const punctuationHtml = punctuation
    ? `<span class="epigraph__punctuation">${md.renderInline(punctuation)}</span>`
    : ''

  return `${contentHtml}${punctuationHtml}`
}

function getCnEpigraphTypography(line: string): { fontSize: string; letterSpacing: string } {
  const contentLength = line.replace(/[\s，。！？；：、“”‘’「」『』（）()《》〈〉【】〔〕,.!?;:]/gu, '').length

  if (contentLength >= 42) {
    return {
      fontSize: 'clamp(0.92rem, 0.72vw + 0.68rem, 1.08rem)',
      letterSpacing: '0.03em',
    }
  }

  if (contentLength >= 24) {
    return {
      fontSize: 'clamp(0.98rem, 0.9vw + 0.66rem, 1.2rem)',
      letterSpacing: '0.05em',
    }
  }

  return {
    fontSize: 'clamp(1.04rem, 1.15vw + 0.68rem, 1.32rem)',
    letterSpacing: '0.07em',
  }
}

function buildEpigraphHtml(blockContent: string): string {
  const lines = blockContent
    .split('\n')
    .map((line) => line.trim())
    .filter((line) => line.length > 0)

  if (lines.length === 0) {
    return ''
  }

  let author = ''
  const authorLine = lines[lines.length - 1]
  const authorMatch = authorLine.match(/^(?:--|---|——)\s*(.+)$/)

  if (authorMatch) {
    author = authorMatch[1].trim()
    lines.pop()
  }

  if (lines.length === 0) {
    return ''
  }

  const body = lines
    .map((line) => {
      const lineType = classifyEpigraphLine(line)

      if (lineType === 'cn') {
        const segments = splitCnEpigraphSegments(line)
        const typography = getCnEpigraphTypography(line)
        const segmentHtml = segments
          .map((segment) => `<span class="epigraph__segment">${renderCnEpigraphSegment(segment)}</span>`)
          .join('')

        return `<p class="epigraph__line epigraph__line--${lineType}" style="font-size: ${typography.fontSize}; letter-spacing: ${typography.letterSpacing};">${segmentHtml}</p>`
      }

      return `<p class="epigraph__line epigraph__line--${lineType}">${md.renderInline(line)}</p>`
    })
    .join('')

  const caption = author
    ? `<figcaption class="epigraph__author">—— ${md.renderInline(author)}</figcaption>`
    : ''

  return `<figure class="epigraph"><div class="epigraph__content"><blockquote class="epigraph__body">${body}</blockquote>${caption}</div></figure>`
}

function renderEpigraphBlocks(content: string): string {
  const replacements: Array<{ token: string; html: string }> = []
  let index = 0

  const markdownWithTokens = content.replace(epigraphBlockPattern, (_, leadingBreak: string, blockContent: string) => {
    const html = buildEpigraphHtml(blockContent)
    if (!html) {
      return leadingBreak
    }

    const token = `${epigraphTokenPrefix}${index}@@`
    index += 1
    replacements.push({ token, html })
    return `${leadingBreak}${token}\n`
  })

  let rendered = md.render(markdownWithTokens)
  for (const replacement of replacements) {
    rendered = rendered.split(`<p>${replacement.token}</p>\n`).join(`${replacement.html}\n`)
    rendered = rendered.split(`<p>${replacement.token}</p>`).join(replacement.html)
    rendered = rendered.split(replacement.token).join(replacement.html)
  }

  return rendered
}

const renderedHtml = computed(() => {
  if (!props.content) return ''

  try {
    return renderEpigraphBlocks(props.content)
  } catch {
    return props.content
  }
})
</script>

<style scoped>
.markdown-renderer {
  line-height: 1.8;
  color: var(--color-text-1);
  font-size: 15px;
}

.markdown-renderer--compact {
  line-height: 1.5;
  font-size: 0.9em;
}

/* Markdown content styles */
.markdown-renderer :deep(h1),
.markdown-renderer :deep(h2),
.markdown-renderer :deep(h3),
.markdown-renderer :deep(h4),
.markdown-renderer :deep(h5),
.markdown-renderer :deep(h6) {
  margin-top: 1.8em;
  margin-bottom: 0.8em;
  font-weight: 600;
  line-height: 1.3;
  color: var(--color-text-1);
}

.markdown-renderer :deep(h1) {
  font-size: 1.8em;
  border-bottom: 1px solid var(--color-border-2);
  padding-bottom: 0.4em;
  margin-top: 0;
}

.markdown-renderer :deep(h2) {
  font-size: 1.4em;
  border-bottom: 1px solid var(--color-border-2);
  padding-bottom: 0.3em;
}

.markdown-renderer :deep(h3) {
  font-size: 1.2em;
}

.markdown-renderer :deep(p) {
  margin: 0.8em 0;
}

.markdown-renderer :deep(a) {
  color: #67c1f5;
  text-decoration: none;
}

.markdown-renderer :deep(a:hover) {
  text-decoration: underline;
  color: #8ed4ff;
}

.markdown-renderer :deep(img) {
  max-width: 100%;
  height: auto;
  border-radius: 8px;
  margin: 1em 0;
}

.markdown-renderer :deep(code) {
  background: var(--color-code-bg, rgba(0, 0, 0, 0.05));
  padding: 0.2em 0.4em;
  border-radius: 4px;
  font-family: 'Fira Code', 'Consolas', monospace;
  font-size: 0.9em;
}

.markdown-renderer :deep(pre) {
  background: #1E1E1E;
  padding: 1em;
  border-radius: 8px;
  overflow-x: auto;
  margin: 1em 0;
}

.markdown-renderer :deep(pre code) {
  background: none;
  padding: 0;
}

.markdown-renderer :deep(blockquote) {
  border-left: 4px solid #6200EE;
  padding-left: 1em;
  margin: 1em 0;
  opacity: 0.8;
}

.markdown-renderer :deep(.epigraph) {
  margin: 2em auto;
  max-width: 900px;
  text-align: center;
  color: color-mix(in srgb, var(--color-text-1) 78%, #6f7f93 22%);
}

.markdown-renderer :deep(.epigraph__content) {
  position: relative;
  padding: 1.35em 3.4em;
}

.markdown-renderer :deep(.epigraph__content::before),
.markdown-renderer :deep(.epigraph__content::after) {
  position: absolute;
  font-size: clamp(2.75rem, 5vw, 4.2rem);
  line-height: 1;
  color: rgba(140, 154, 255, 0.7);
  font-family: Georgia, 'Times New Roman', serif;
  pointer-events: none;
}

.markdown-renderer :deep(.epigraph__content::before) {
  content: '“';
  top: 0;
  left: 0;
  transform: translate(0.1em, 0.15em);
}

.markdown-renderer :deep(.epigraph__content::after) {
  content: '”';
  right: 0;
  bottom: 0;
  transform: translate(-0.08em, 0.3em);
}

.markdown-renderer :deep(.epigraph__body) {
  margin: 0;
  padding: 0;
  border: 0;
  opacity: 1;
}

.markdown-renderer :deep(.epigraph__line) {
  margin: 0.1em 0;
}

.markdown-renderer :deep(.epigraph__line--cn) {
  display: flex;
  flex-wrap: wrap;
  justify-content: center;
  align-items: baseline;
  gap: 0.3em 0.7em;
  line-height: 1.7;
}

.markdown-renderer :deep(.epigraph__segment) {
  display: inline-flex;
  align-items: baseline;
  white-space: nowrap;
}

.markdown-renderer :deep(.epigraph__segment-text) {
  letter-spacing: inherit;
}

.markdown-renderer :deep(.epigraph__punctuation) {
  letter-spacing: 0;
  margin-left: -0.08em;
}

.markdown-renderer :deep(.epigraph__line--en) {
  font-size: clamp(0.82rem, 0.72vw + 0.58rem, 1rem);
  line-height: 1.35;
  letter-spacing: 0.01em;
  color: color-mix(in srgb, var(--color-text-1) 58%, #8b97a8 42%);
}

.markdown-renderer :deep(.epigraph__author) {
  margin-top: 1rem;
  text-align: right;
  font-size: clamp(0.84rem, 0.62vw + 0.7rem, 1.02rem);
  line-height: 1.35;
  color: color-mix(in srgb, var(--color-text-1) 56%, #8a96a6 44%);
  opacity: 0.9;
}

.markdown-renderer :deep(ul),
.markdown-renderer :deep(ol) {
  padding-left: 1.5em;
  margin: 0.5em 0;
}

.markdown-renderer :deep(li) {
  margin: 0.25em 0;
}

.markdown-renderer :deep(table) {
  border-collapse: collapse;
  width: 100%;
  margin: 1em 0;
}

.markdown-renderer :deep(th),
.markdown-renderer :deep(td) {
  border: 1px solid var(--color-border, rgba(0, 0, 0, 0.12));
  padding: 0.5em 1em;
  text-align: left;
}

.markdown-renderer :deep(th) {
  background: var(--color-table-header, rgba(0, 0, 0, 0.03));
  font-weight: 600;
}

.markdown-renderer :deep(hr) {
  border: none;
  border-top: 1px solid var(--color-border, rgba(0, 0, 0, 0.1));
  margin: 2em 0;
}

/* Dark theme support */
@media (prefers-color-scheme: dark) {
  .markdown-renderer :deep(h1),
  .markdown-renderer :deep(h2) {
    border-bottom-color: rgba(255, 255, 255, 0.1);
  }

  .markdown-renderer :deep(code) {
    background: rgba(255, 255, 255, 0.1);
  }

  .markdown-renderer :deep(th),
  .markdown-renderer :deep(td) {
    border-color: rgba(255, 255, 255, 0.12);
  }

  .markdown-renderer :deep(th) {
    background: rgba(255, 255, 255, 0.05);
  }

  .markdown-renderer :deep(hr) {
    border-top-color: rgba(255, 255, 255, 0.1);
  }
}

/* Arco Design dark theme support */
body.arco-layout-sidemenu-dark .markdown-renderer :deep(h1),
body.arco-layout-sidemenu-dark .markdown-renderer :deep(h2) {
  border-bottom-color: rgba(255, 255, 255, 0.1);
}

body.arco-layout-sidemenu-dark .markdown-renderer :deep(code) {
  background: rgba(255, 255, 255, 0.1);
}

body.arco-layout-sidemenu-dark .markdown-renderer :deep(th),
body.arco-layout-sidemenu-dark .markdown-renderer :deep(td) {
  border-color: rgba(255, 255, 255, 0.12);
}

body.arco-layout-sidemenu-dark .markdown-renderer :deep(th) {
  background: rgba(255, 255, 255, 0.05);
}

body.arco-layout-sidemenu-dark .markdown-renderer :deep(hr) {
  border-top-color: rgba(255, 255, 255, 0.1);
}

@media (max-width: 768px) {
  .markdown-renderer {
    font-size: 14px;
    line-height: 1.75;
  }

  .markdown-renderer :deep(h1) {
    font-size: 1.45em;
    line-height: 1.25;
  }

  .markdown-renderer :deep(h2) {
    font-size: 1.24em;
  }

  .markdown-renderer :deep(blockquote) {
    padding-left: 0.85em;
  }

  .markdown-renderer :deep(.epigraph) {
    margin: 1.6em auto;
  }

  .markdown-renderer :deep(.epigraph__content) {
    padding: 1.1em 1.8em 0.9em;
  }

  .markdown-renderer :deep(.epigraph__line--cn) {
    gap: 0.18em 0.36em;
    line-height: 1.6;
  }

  .markdown-renderer :deep(.epigraph__segment) {
    white-space: normal;
  }
}
</style>
