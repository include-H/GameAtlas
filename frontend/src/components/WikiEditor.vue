<template>
  <div class="wiki-editor">
    <div class="wiki-editor__toolbar">
      <div class="wiki-editor__toolbar-groups">
        <a-button-group class="wiki-editor__toolbar-group">
          <a-button type="text" @click="insertMarkdown('**', '**')">
            <template #icon>
              <icon-bold />
            </template>
          </a-button>
          <a-button type="text" @click="insertMarkdown('*', '*')">
            <template #icon>
              <icon-italic />
            </template>
          </a-button>
          <a-button type="text" @click="insertMarkdown('`', '`')">
            <template #icon>
              <icon-code />
            </template>
          </a-button>
          <a-button type="text" @click="insertMarkdown('~~', '~~')">
            <template #icon>
              <icon-highlight />
            </template>
          </a-button>
        </a-button-group>

        <a-button-group class="wiki-editor__toolbar-group">
          <a-button type="text" @click="insertLine('# ')">
            <template #icon>
              <icon-h1 />
            </template>
          </a-button>
          <a-button type="text" @click="insertLine('## ')">
            <span class="toolbar-h2">H2</span>
          </a-button>
          <a-button type="text" @click="insertLine('### ')">
            <span class="toolbar-h3">H3</span>
          </a-button>
          <a-button type="text" @click="insertDivider">
            <template #icon>
              <icon-minus />
            </template>
          </a-button>
        </a-button-group>

        <a-button-group class="wiki-editor__toolbar-group">
          <a-button type="text" @click="insertLink">
            <template #icon>
              <icon-link />
            </template>
          </a-button>
          <a-button type="text" @click="insertImage">
            <template #icon>
              <icon-image />
            </template>
          </a-button>
          <a-button type="text" @click="insertCodeBlock">
            <template #icon>
              <icon-code-block />
            </template>
          </a-button>
          <a-button type="text" @click="insertTable">
            <template #icon>
              <icon-list />
            </template>
          </a-button>
        </a-button-group>

        <a-button-group class="wiki-editor__toolbar-group">
          <a-button type="text" @click="insertLine('- ')">
            <template #icon>
              <icon-unordered-list />
            </template>
          </a-button>
          <a-button type="text" @click="insertLine('1. ')">
            <template #icon>
              <icon-ordered-list />
            </template>
          </a-button>
          <a-button type="text" @click="insertLine('- [ ] ')">
            <template #icon>
              <icon-check-square />
            </template>
          </a-button>
          <a-button type="text" @click="insertLine('> ')">
            <template #icon>
              <icon-quote />
            </template>
          </a-button>
        </a-button-group>
      </div>

      <div class="wiki-editor__toolbar-actions">
        <a-button :type="showPreview ? 'primary' : 'secondary'" @click="showPreview = !showPreview">
          <template #icon>
            <icon-eye-invisible v-if="showPreview" />
            <icon-eye v-else />
          </template>
          {{ showPreview ? '返回编辑' : '预览' }}
        </a-button>
      </div>
    </div>

    <div class="wiki-editor__content">
      <textarea
        v-show="!showPreview"
        ref="editorRef"
        v-model="content"
        placeholder="使用 Markdown 编写 Wiki 内容..."
        class="wiki-editor__textarea"
        @keydown.tab.prevent="handleTab"
      />

      <div v-show="showPreview" class="wiki-editor__preview">
        <markdown-renderer v-if="content" :content="content" />
        <div v-else class="wiki-editor__preview-empty">
          <icon-file class="wiki-editor__preview-icon" />
          <p>开始输入后可在这里预览</p>
        </div>
      </div>
    </div>

    <div class="wiki-editor__footer">
      <a-tag size="small">
        {{ wordCount }} 词
      </a-tag>
      <a-tag size="small" class="wiki-editor__footer-tag">
        {{ characterCount }} 字
      </a-tag>
    </div>
  </div>
</template>

<script setup lang="ts">
import { computed, ref, watch } from 'vue'
import MarkdownRenderer from './MarkdownRenderer.vue'
import {
  IconBold,
  IconCode,
  IconCodeBlock,
  IconCheckSquare,
  IconEye,
  IconEyeInvisible,
  IconFile,
  IconH1,
  IconHighlight,
  IconImage,
  IconItalic,
  IconLink,
  IconList,
  IconMinus,
  IconOrderedList,
  IconQuote,
  IconUnorderedList,
} from '@arco-design/web-vue/es/icon'

interface Props {
  modelValue: string
}

const props = defineProps<Props>()

const emit = defineEmits<{
  'update:modelValue': [value: string]
}>()

const content = ref(props.modelValue)
const showPreview = ref(false)
const editorRef = ref<HTMLTextAreaElement | null>(null)

watch(content, (value) => {
  emit('update:modelValue', value)
})

watch(() => props.modelValue, (value) => {
  content.value = value
})

const wordCount = computed(() => content.value.trim().split(/\s+/).filter(Boolean).length)
const characterCount = computed(() => content.value.length)

const insertMarkdown = (before: string, after: string) => {
  const textarea = editorRef.value
  if (!textarea) return

  const start = textarea.selectionStart
  const end = textarea.selectionEnd
  const scrollTop = textarea.scrollTop
  const selectedText = content.value.substring(start, end)

  content.value = `${content.value.substring(0, start)}${before}${selectedText}${after}${content.value.substring(end)}`

  setTimeout(() => {
    textarea.focus()
    textarea.scrollTop = scrollTop
    textarea.setSelectionRange(start + before.length, end + before.length)
  }, 0)
}

const insertLine = (prefix: string) => {
  const textarea = editorRef.value
  if (!textarea) return

  const start = textarea.selectionStart
  const end = textarea.selectionEnd
  const scrollTop = textarea.scrollTop
  const lineStart = content.value.lastIndexOf('\n', start - 1) + 1

  content.value = `${content.value.substring(0, lineStart)}${prefix}${content.value.substring(lineStart)}`

  setTimeout(() => {
    textarea.focus()
    textarea.scrollTop = scrollTop
    textarea.setSelectionRange(start + prefix.length, end + prefix.length)
  }, 0)
}

const insertLink = () => {
  const url = prompt('Enter URL:')
  if (!url) return
  insertMarkdown('[', `](${url})`)
}

const insertImage = () => {
  const url = prompt('Enter image URL:')
  if (!url) return
  insertMarkdown('![', `](${url})`)
}

const insertCodeBlock = () => {
  const textarea = editorRef.value
  if (!textarea) return

  const start = textarea.selectionStart
  const end = textarea.selectionEnd
  const scrollTop = textarea.scrollTop
  const selectedText = content.value.substring(start, end) || 'code'

  content.value = `${content.value.substring(0, start)}\`\`\`\n${selectedText}\n\`\`\`${content.value.substring(end)}`

  setTimeout(() => {
    textarea.focus()
    textarea.scrollTop = scrollTop
    const selectionStart = start + 4
    const selectionEnd = selectionStart + selectedText.length
    textarea.setSelectionRange(selectionStart, selectionEnd)
  }, 0)
}

const insertDivider = () => {
  const textarea = editorRef.value
  if (!textarea) return

  const start = textarea.selectionStart
  const scrollTop = textarea.scrollTop
  const lineStart = content.value.lastIndexOf('\n', start - 1) + 1
  const prefix = lineStart === 0 ? '' : '\n'
  const suffix = start >= content.value.length ? '' : '\n'

  content.value = `${content.value.substring(0, lineStart)}${prefix}---${suffix}${content.value.substring(lineStart)}`

  setTimeout(() => {
    textarea.focus()
    textarea.scrollTop = scrollTop
    const cursor = lineStart + prefix.length + 3 + suffix.length
    textarea.setSelectionRange(cursor, cursor)
  }, 0)
}

const insertTable = () => {
  const textarea = editorRef.value
  if (!textarea) return

  const start = textarea.selectionStart
  const scrollTop = textarea.scrollTop
  const lineStart = content.value.lastIndexOf('\n', start - 1) + 1
  const prefix = lineStart === 0 ? '' : '\n'
  const table = `${prefix}| 列 1 | 列 2 | 列 3 |\n| --- | --- | --- |\n| 内容 | 内容 | 内容 |\n`

  content.value = `${content.value.substring(0, lineStart)}${table}${content.value.substring(lineStart)}`

  setTimeout(() => {
    textarea.focus()
    textarea.scrollTop = scrollTop
    const selectionStart = lineStart + prefix.length + 2
    const selectionEnd = selectionStart + 3
    textarea.setSelectionRange(selectionStart, selectionEnd)
  }, 0)
}

const handleTab = () => {
  insertMarkdown('  ', '')
}
</script>

<style scoped>
.wiki-editor {
  display: flex;
  flex-direction: column;
  border-radius: 8px;
  overflow: hidden;
  border: 1px solid var(--app-card-border);
  background: var(--app-card-surface);
  backdrop-filter: blur(var(--app-card-backdrop-blur));
  -webkit-backdrop-filter: blur(var(--app-card-backdrop-blur));
  height: 100%;
  min-height: 0;
}

.wiki-editor__toolbar {
  position: sticky;
  top: 0;
  z-index: 2;
  flex-shrink: 0;
  display: flex;
  align-items: center;
  justify-content: space-between;
  padding: 8px 12px;
  background: var(--color-fill-2);
  border-bottom: 1px solid var(--color-border-2);
  gap: 8px;
}

.wiki-editor__toolbar-groups {
  display: flex;
  align-items: center;
  gap: 8px;
  min-width: 0;
  overflow-x: auto;
  padding-bottom: 2px;
  scrollbar-width: thin;
}

.wiki-editor__toolbar-group {
  flex-shrink: 0;
}

.wiki-editor__toolbar-actions {
  flex-shrink: 0;
  display: flex;
  align-items: center;
  gap: 10px;
}

.wiki-editor :deep(.arco-btn-group) {
  border-radius: 10px;
  overflow: hidden;
  border: 1px solid var(--color-border-2);
  background: color-mix(in srgb, var(--color-bg-2) 85%, transparent);
}

.wiki-editor :deep(.arco-btn-group .arco-btn) {
  min-width: auto;
}

.wiki-editor__content {
  flex: 1;
  min-height: 0;
  overflow: hidden;
  background: var(--color-fill-1);
}

.wiki-editor__textarea {
  width: 100%;
  display: block;
  padding: 16px 18px;
  background: transparent;
  color: var(--color-text-1);
  border: 0;
  outline: none;
  box-sizing: border-box;
  height: 100%;
  min-height: 100%;
  resize: none;
  overflow-y: auto;
  font-family: var(--font-family-base);
  font-size: 14px;
  line-height: 1.6;
}

.wiki-editor__textarea::placeholder {
  color: var(--color-text-4);
}

.wiki-editor__preview {
  height: 100%;
  overflow-y: auto;
  padding: 16px;
  background: var(--color-fill-1);
  box-sizing: border-box;
}

.wiki-editor__preview-empty {
  text-align: center;
  color: var(--color-text-3);
  padding: 32px;
}

.wiki-editor__preview-icon {
  font-size: 48px;
  margin-bottom: 16px;
}

.wiki-editor__preview-empty p {
  margin: 0;
}

.wiki-editor__footer {
  flex-shrink: 0;
  display: flex;
  align-items: center;
  gap: 8px;
  padding: 8px 16px;
  background: var(--color-fill-2);
  border-top: 1px solid var(--color-border-2);
}

.wiki-editor__footer-tag {
  margin-left: 4px;
}

.toolbar-h2,
.toolbar-h3 {
  font-weight: 700;
  font-size: 12px;
}

@media (max-width: 768px) {
  .wiki-editor__toolbar {
    padding: 8px 10px;
    align-items: flex-start;
    flex-direction: column;
  }

  .wiki-editor__toolbar-actions {
    width: 100%;
    justify-content: space-between;
  }

  .wiki-editor__textarea,
  .wiki-editor__preview {
    padding: 14px;
  }

  .wiki-editor__footer {
    padding: 8px 12px;
    flex-wrap: wrap;
  }
}

@media (max-width: 576px) {
  .wiki-editor__toolbar {
    gap: 10px;
  }

  .wiki-editor__toolbar-groups {
    width: 100%;
  }

  .wiki-editor__textarea {
    padding: 12px;
    font-size: 13px;
  }

  .wiki-editor__preview {
    padding: 12px;
  }
}
</style>
