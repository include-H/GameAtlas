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
  html: true,
  linkify: true,
  typographer: true,
  breaks: true,
})

const renderedHtml = computed(() => {
  if (!props.content) return ''

  try {
    return md.render(props.content)
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
</style>
