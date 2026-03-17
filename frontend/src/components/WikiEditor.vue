<template>
  <div class="wiki-editor">
    <!-- Toolbar -->
    <div class="wiki-editor__toolbar">
      <a-space :size="4">
        <a-button-group>
          <a-button @click="insertMarkdown('**', '**')">
            <template #icon>
              <icon-bold />
            </template>
          </a-button>
          <a-button @click="insertMarkdown('*', '*')">
            <template #icon>
              <icon-italic />
            </template>
          </a-button>
          <a-button @click="insertMarkdown('__', '__')">
            <template #icon>
              <icon-minus />
            </template>
          </a-button>
          <a-button @click="insertMarkdown('~~', '~~')">
            <template #icon>
              <icon-highlight />
            </template>
          </a-button>
        </a-button-group>

        <a-divider direction="vertical" :margin="8" />

        <a-button-group>
          <a-button @click="insertLine('# ')">
            <template #icon>
              <icon-h1 />
            </template>
          </a-button>
          <a-button @click="insertLine('## ')">
            <span class="toolbar-h2">H2</span>
          </a-button>
          <a-button @click="insertLine('### ')">
            <span class="toolbar-h3">H3</span>
          </a-button>
        </a-button-group>

        <a-divider direction="vertical" :margin="8" />

        <a-button-group>
          <a-button @click="insertLink">
            <template #icon>
              <icon-link />
            </template>
          </a-button>
          <a-button @click="insertImage">
            <template #icon>
              <icon-image />
            </template>
          </a-button>
          <a-button @click="insertCode">
            <template #icon>
              <icon-code />
            </template>
          </a-button>
        </a-button-group>

        <a-divider direction="vertical" :margin="8" />

        <a-button-group>
          <a-button @click="insertLine('- ')">
            <template #icon>
              <icon-unordered-list />
            </template>
          </a-button>
          <a-button @click="insertLine('1. ')">
            <template #icon>
              <icon-ordered-list />
            </template>
          </a-button>
          <a-button @click="insertLine('> ')">
            <template #icon>
              <icon-text />
            </template>
          </a-button>
        </a-button-group>
      </a-space>

      <a-button
        :type="showPreview ? 'primary' : 'secondary'"
        @click="showPreview = !showPreview"
      >
        <template #icon>
          <icon-eye-invisible v-if="showPreview" />
          <icon-eye v-else />
        </template>
        {{ showPreview ? 'Edit' : 'Preview' }}
      </a-button>
    </div>

    <!-- Editor / Preview -->
    <div class="wiki-editor__content">
      <!-- Editor -->
      <textarea
        v-show="!showPreview"
        ref="editorRef"
        v-model="content"
        placeholder="Write your wiki content in Markdown..."
        class="wiki-editor__textarea"
        @keydown.tab.prevent="handleTab"
      />

      <!-- Preview -->
      <div v-show="showPreview" class="wiki-editor__preview">
        <markdown-renderer v-if="content" :content="content" />
        <div v-else class="wiki-editor__preview-empty">
          <icon-file class="wiki-editor__preview-icon" />
          <p>Start typing to see preview</p>
        </div>
      </div>
    </div>

    <!-- Word count -->
    <div class="wiki-editor__footer">
      <a-tag size="small">
        {{ wordCount }} words
      </a-tag>
      <a-tag size="small" class="wiki-editor__footer-tag">
        {{ characterCount }} characters
      </a-tag>
    </div>
  </div>
</template>

<script setup lang="ts">
import { ref, computed, watch } from 'vue'
import MarkdownRenderer from './MarkdownRenderer.vue'
import {
  IconBold,
  IconItalic,
  IconLink,
  IconImage,
  IconCode,
  IconUnorderedList,
  IconOrderedList,
  IconEye,
  IconEyeInvisible,
  IconFile,
  IconHighlight,
  IconH1,
  IconMinus
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

// Sync with parent
watch(content, (value) => {
  emit('update:modelValue', value)
})

watch(() => props.modelValue, (value) => {
  content.value = value
})

const wordCount = computed(() => {
  return content.value.trim().split(/\s+/).filter(w => w).length
})

const characterCount = computed(() => {
  return content.value.length
})

const insertMarkdown = (before: string, after: string) => {
  const textarea = editorRef.value
  if (!textarea) return

  const start = textarea.selectionStart
  const end = textarea.selectionEnd
  const scrollTop = textarea.scrollTop
  const selectedText = content.value.substring(start, end)

  const newText = content.value.substring(0, start) +
    before + selectedText + after +
    content.value.substring(end)

  content.value = newText

  // Set cursor position
  setTimeout(() => {
    textarea.focus()
    textarea.scrollTop = scrollTop
    textarea.setSelectionRange(
      start + before.length,
      end + before.length
    )
  }, 0)
}

const insertLine = (prefix: string) => {
  const textarea = editorRef.value
  if (!textarea) return

  const start = textarea.selectionStart
  const end = textarea.selectionEnd
  const scrollTop = textarea.scrollTop

  // Find the start of the current line
  const lineStart = content.value.lastIndexOf('\n', start - 1) + 1

  const newText = content.value.substring(0, lineStart) +
    prefix +
    content.value.substring(lineStart)

  content.value = newText

  setTimeout(() => {
    textarea.focus()
    textarea.scrollTop = scrollTop
    textarea.setSelectionRange(
      start + prefix.length,
      end + prefix.length
    )
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

const insertCode = () => {
  const textarea = editorRef.value
  if (!textarea) return

  const start = textarea.selectionStart
  const end = textarea.selectionEnd
  const scrollTop = textarea.scrollTop
  const selectedText = content.value.substring(start, end) || 'code'

  const newText = content.value.substring(0, start) +
    '```\n' + selectedText + '\n```' +
    content.value.substring(end)

  content.value = newText

  setTimeout(() => {
    textarea.focus()
    textarea.scrollTop = scrollTop
    const selectionStart = start + 4
    const selectionEnd = selectionStart + selectedText.length
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
  border: 1px solid var(--color-border-2);
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

.wiki-editor__content {
  flex: 1;
  min-height: 0;
  overflow: hidden;
  background: var(--color-fill-1);
}

.wiki-editor__textarea {
  height: 100%;
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
  font-family: 'Fira Code', 'Consolas', monospace;
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
</style>
