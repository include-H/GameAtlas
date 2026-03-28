<template>
  <aside class="wiki-toc" aria-labelledby="wiki-toc-title">
    <div id="wiki-toc-title" class="wiki-toc__header">目录</div>
    <div ref="tocBodyRef" class="wiki-toc__body">
      <nav v-if="headings.length > 0" class="wiki-toc__nav" aria-label="Wiki 目录">
        <a
          v-for="heading in headings"
          :key="heading.id"
          :href="`#${heading.id}`"
          class="wiki-toc__item"
          :class="{
            'wiki-toc__item--active': activeId === heading.id,
            [`wiki-toc__item--level-${heading.level}`]: true
          }"
          :aria-current="activeId === heading.id ? 'location' : undefined"
          @click.prevent="scrollToHeading(heading.id)"
        >
          {{ heading.text }}
        </a>
      </nav>
      <div v-else class="wiki-toc__empty">暂无目录</div>
    </div>
  </aside>
</template>

<script setup lang="ts">
import { nextTick, onMounted, onUnmounted, ref } from 'vue'
import { useRouter } from 'vue-router'

interface Heading {
  id: string
  text: string
  level: number
}

const headings = ref<Heading[]>([])
const activeId = ref('')
const tocBodyRef = ref<HTMLElement | null>(null)
const router = useRouter()

const HEADING_SELECTOR = 'h1, h2, h3'
const HEADING_SCROLL_OFFSET = 76

let scrollContainer: HTMLElement | null = null
let contentObserver: MutationObserver | null = null
let headingObserver: IntersectionObserver | null = null
let wikiCardObserver: MutationObserver | null = null
let observedWikiContent: HTMLElement | null = null
let isPageUnloading = false

const generateId = (text: string, index: number): string => {
  const baseId = text
    .toLowerCase()
    .replace(/[^\w\u4e00-\u9fa5]+/g, '-')
    .replace(/^-+|-+$/g, '')
  return index > 0 ? `${baseId}-${index}` : baseId
}

const getWikiContent = () => {
  return document.querySelector('.game-detail__wiki-card .markdown-renderer') as HTMLElement | null
}

const getHeadingElements = () => {
  return headings.value
    .map((heading) => document.getElementById(heading.id))
    .filter((element): element is HTMLElement => Boolean(element))
}

const updateActiveId = (nextId: string) => {
  if (!nextId || nextId === activeId.value) return

  activeId.value = nextId
  nextTick(() => {
    scrollToActiveItem()
  })
}

const computeActiveHeading = () => {
  const elements = getHeadingElements()
  if (elements.length === 0) return

  const rootTop = scrollContainer?.getBoundingClientRect().top ?? 0
  const containerHeight = scrollContainer?.clientHeight ?? window.innerHeight
  const readingLine = rootTop + Math.min(containerHeight * 0.32, 220) + HEADING_SCROLL_OFFSET

  let candidate: HTMLElement | null = null

  for (const element of elements) {
    const rect = element.getBoundingClientRect()
    if (rect.top <= readingLine) {
      candidate = element
      continue
    }

    if (!candidate) {
      candidate = element
    }
    break
  }

  updateActiveId(candidate?.id || elements[0].id)
}

const disconnectHeadingObserver = () => {
  if (headingObserver) {
    headingObserver.disconnect()
    headingObserver = null
  }
}

const setupHeadingObserver = () => {
  disconnectHeadingObserver()

  const elements = getHeadingElements()
  if (elements.length === 0 || typeof IntersectionObserver === 'undefined') return

  headingObserver = new IntersectionObserver(
    () => {
      computeActiveHeading()
    },
    {
      root: scrollContainer,
      rootMargin: `-${HEADING_SCROLL_OFFSET}px 0px -55% 0px`,
      threshold: [0, 0.2, 0.6, 1],
    },
  )

  for (const element of elements) {
    headingObserver.observe(element)
  }
}

const extractHeadingsFromDOM = async () => {
  await nextTick()

  const wikiContent = getWikiContent()
  if (!wikiContent) {
    headings.value = []
    activeId.value = ''
    disconnectHeadingObserver()
    return
  }

  const hElements = Array.from(wikiContent.querySelectorAll(HEADING_SELECTOR)) as HTMLElement[]
  const result: Heading[] = []
  const idMap: Record<string, number> = {}

  for (const element of hElements) {
    const text = element.textContent?.trim() || ''
    if (!text) continue

    const level = Number.parseInt(element.tagName[1] || '1', 10)
    const baseId = generateId(text, 0)
    const count = idMap[baseId] || 0
    idMap[baseId] = count + 1

    const id = count > 0 ? `${baseId}-${count}` : baseId
    element.id = id
    element.style.scrollMarginTop = `${HEADING_SCROLL_OFFSET}px`
    result.push({ id, text, level })
  }

  headings.value = result
  activeId.value = result[0]?.id || ''

  setupHeadingObserver()
  computeActiveHeading()
}

const scrollToHeading = (id: string) => {
  const element = document.getElementById(id)
  if (!element) return

  updateActiveId(id)

  if (scrollContainer) {
    const containerRect = scrollContainer.getBoundingClientRect()
    const targetRect = element.getBoundingClientRect()
    const top = scrollContainer.scrollTop + targetRect.top - containerRect.top - HEADING_SCROLL_OFFSET
    scrollContainer.scrollTo({
      top: Math.max(top, 0),
      behavior: 'smooth',
    })
  } else {
    element.scrollIntoView({ behavior: 'smooth', block: 'start' })
  }

  replaceUrlHash(id)
}

const markPageUnloading = () => {
  isPageUnloading = true
}

const replaceUrlHash = (id: string) => {
  if (typeof window === 'undefined') return
  if (isPageUnloading || document.visibilityState === 'hidden') return

  const targetHash = `#${id}`
  if (window.location.hash === targetHash) return

  void router.replace({ hash: targetHash }).then(
    () => undefined,
    () => undefined,
  )
}

const scrollToActiveItem = () => {
  const tocBody = tocBodyRef.value
  if (!tocBody) return

  const activeItem = tocBody.querySelector('.wiki-toc__item--active') as HTMLElement | null
  if (!activeItem) return

  const itemTop = activeItem.offsetTop
  const itemBottom = itemTop + activeItem.offsetHeight
  const containerTop = tocBody.scrollTop
  const containerBottom = containerTop + tocBody.clientHeight

  if (itemTop < containerTop + 8) {
    tocBody.scrollTo({
      top: Math.max(itemTop - 8, 0),
      behavior: 'smooth',
    })
    return
  }

  if (itemBottom > containerBottom - 8) {
    tocBody.scrollTo({
      top: itemBottom - tocBody.clientHeight + 8,
      behavior: 'smooth',
    })
  }
}

const setupContentObserver = () => {
  const wikiContent = getWikiContent()
  if (!wikiContent || typeof MutationObserver === 'undefined') return
  if (observedWikiContent === wikiContent && contentObserver) return

  contentObserver?.disconnect()
  observedWikiContent = wikiContent
  contentObserver = new MutationObserver(() => {
    void extractHeadingsFromDOM()
  })
  contentObserver.observe(wikiContent, {
    childList: true,
    subtree: true,
    characterData: true,
  })
}

const setupWikiCardObserver = () => {
  if (typeof MutationObserver === 'undefined') return

  const wikiCard = document.querySelector('.game-detail__wiki-card') as HTMLElement | null
  if (!wikiCard) return

  wikiCardObserver?.disconnect()
  wikiCardObserver = new MutationObserver(() => {
    setupContentObserver()
    void extractHeadingsFromDOM()
  })
  wikiCardObserver.observe(wikiCard, {
    childList: true,
    subtree: true,
  })
}

const handleResize = () => {
  computeActiveHeading()
}

onMounted(() => {
  isPageUnloading = false
  scrollContainer = document.querySelector('.content')
  setupWikiCardObserver()
  setupContentObserver()
  void extractHeadingsFromDOM()

  if (scrollContainer) {
    scrollContainer.addEventListener('scroll', computeActiveHeading, { passive: true })
  }
  if (typeof window !== 'undefined') {
    window.addEventListener('resize', handleResize, { passive: true })
    window.addEventListener('beforeunload', markPageUnloading)
    window.addEventListener('pagehide', markPageUnloading)
  }
})

onUnmounted(() => {
  disconnectHeadingObserver()
  contentObserver?.disconnect()
  contentObserver = null
  observedWikiContent = null
  wikiCardObserver?.disconnect()
  wikiCardObserver = null

  if (scrollContainer) {
    scrollContainer.removeEventListener('scroll', computeActiveHeading)
  }
  if (typeof window !== 'undefined') {
    window.removeEventListener('resize', handleResize)
    window.removeEventListener('beforeunload', markPageUnloading)
    window.removeEventListener('pagehide', markPageUnloading)
  }
})
</script>

<style scoped>
.wiki-toc {
  background: var(--app-card-surface);
  border: 1px solid var(--app-card-border);
  border-radius: var(--radius-lg);
  backdrop-filter: blur(var(--app-card-backdrop-blur));
  -webkit-backdrop-filter: blur(var(--app-card-backdrop-blur));
  position: sticky;
  top: 80px;
  max-height: calc(100vh - 100px);
  overflow: hidden;
  flex: 0 0 25.5%;
  display: flex;
  flex-direction: column;
}

.wiki-toc__header {
  font-size: 16px;
  font-weight: 600;
  color: var(--color-text-1);
  padding: 12px 20px;
  border-bottom: 1px solid var(--color-border-2);
  flex: 0 0 auto;
}

.wiki-toc__body {
  padding: 20px;
  overflow-y: auto;
  scroll-behavior: smooth;
  flex: 1 1 auto;
  min-height: 0;
}

.wiki-toc__nav {
  display: flex;
  flex-direction: column;
  gap: 4px;
}

.wiki-toc__item {
  display: block;
  padding: 6px 8px;
  font-size: 13px;
  color: var(--color-text-2);
  text-decoration: none;
  border-radius: 4px;
  transition: all 0.2s ease;
  line-height: 1.4;
  cursor: pointer;
}

.wiki-toc__item:hover {
  color: var(--color-text-1);
  background: var(--color-fill-2);
}

.wiki-toc__item--active {
  color: #1a9fff;
  background: rgba(26, 159, 255, 0.1);
  font-weight: 500;
}

.wiki-toc__item--level-1 {
  font-weight: 500;
}

.wiki-toc__item--level-2 {
  padding-left: 16px;
}

.wiki-toc__item--level-3 {
  padding-left: 28px;
  font-size: 12px;
}

.wiki-toc__body::-webkit-scrollbar {
  width: 4px;
}

.wiki-toc__body::-webkit-scrollbar-track {
  background: transparent;
}

.wiki-toc__body::-webkit-scrollbar-thumb {
  background: var(--color-border-2);
  border-radius: 2px;
}

.wiki-toc__body::-webkit-scrollbar-thumb:hover {
  background: var(--color-border-3);
}

.wiki-toc__empty {
  font-size: 12px;
  color: var(--color-text-3);
  text-align: center;
  padding: 16px 0;
}

@media (max-width: 992px) {
  .wiki-toc {
    display: none;
  }
}
</style>
