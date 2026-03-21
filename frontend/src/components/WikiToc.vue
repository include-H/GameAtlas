<template>
  <div class="wiki-toc">
    <div class="wiki-toc__title">目录</div>
    <nav v-if="headings.length > 0" class="wiki-toc__nav">
      <a
        v-for="heading in headings"
        :key="heading.id"
        :href="`#${heading.id}`"
        class="wiki-toc__item"
        :class="{
          'wiki-toc__item--active': activeId === heading.id,
          [`wiki-toc__item--level-${heading.level}`]: true
        }"
        @click.prevent="scrollToHeading(heading.id)"
      >
        {{ heading.text }}
      </a>
    </nav>
    <div v-else class="wiki-toc__empty">暂无目录</div>
  </div>
</template>

<script setup lang="ts">
import { ref, onMounted, onUnmounted, nextTick } from 'vue'

interface Heading {
  id: string
  text: string
  level: number
}

const headings = ref<Heading[]>([])
const activeId = ref<string>('')

// 生成 ID 的辅助函数
const generateId = (text: string, index: number): string => {
  const baseId = text
    .toLowerCase()
    .replace(/[^\w\u4e00-\u9fa5]+/g, '-')
    .replace(/^-+|-+$/g, '')
  return index > 0 ? `${baseId}-${index}` : baseId
}

// 从已渲染的 DOM 中提取标题
const extractHeadingsFromDOM = () => {
  nextTick(() => {
    const wikiContent = document.querySelector('.markdown-renderer')
    if (!wikiContent) {
      headings.value = []
      return
    }

    const hElements = wikiContent.querySelectorAll('h1, h2, h3')
    const result: Heading[] = []
    const idMap: Record<string, number> = {}

    hElements.forEach((el) => {
      const text = el.textContent?.trim() || ''
      if (!text) return

      const level = parseInt(el.tagName[1])
      const baseId = generateId(text, 0)

      const count = idMap[baseId] || 0
      idMap[baseId] = count + 1
      const id = count > 0 ? `${baseId}-${count}` : baseId

      el.id = id
      result.push({ id, text, level })
    })

    headings.value = result

    if (result.length > 0) {
      activeId.value = result[0].id
      handleScroll()
    }
  })
}

// 滚动到指定标题
const scrollToHeading = (id: string) => {
  const element = document.getElementById(id)
  if (element) {
    activeId.value = id
    element.scrollIntoView({ behavior: 'smooth', block: 'start' })

    setTimeout(() => {
      const offset = 80
      const elementPosition = element.getBoundingClientRect().top + window.pageYOffset
      window.scrollTo({
        top: elementPosition - offset,
        behavior: 'auto'
      })
    }, 400)
  }
}

// 监听滚动，高亮当前标题并滚动目录
const handleScroll = () => {
  if (headings.value.length === 0) return

  // 使用视口中心作为判断基准
  const viewportCenter = window.innerHeight / 2
  let closestHeading: Heading | null = null
  let minDistance = Infinity

  // 找到距离视口中心最近的标题
  for (const heading of headings.value) {
    const element = document.getElementById(heading.id)
    if (element) {
      const rect = element.getBoundingClientRect()
      const distance = Math.abs(rect.top - viewportCenter)

      if (distance < minDistance) {
        minDistance = distance
        closestHeading = heading
      }
    }
  }

  // 更新激活项
  if (closestHeading) {
    const changed = closestHeading.id !== activeId.value
    activeId.value = closestHeading.id
    // 只在激活项变化时滚动目录
    if (changed) {
      nextTick(() => {
        scrollToActiveItem()
      })
    }
  }
}

// 滚动目录使激活项保持在可视区域
const scrollToActiveItem = () => {
  const tocContainer = document.querySelector('.wiki-toc')
  const activeItem = document.querySelector('.wiki-toc__item--active')

  if (!tocContainer || !activeItem) return

  // 使用 offsetTop 获取相对于滚动容器的位置
  const itemTop = (activeItem as HTMLElement).offsetTop
  const itemBottom = itemTop + (activeItem as HTMLElement).offsetHeight
  const containerTop = tocContainer.scrollTop
  const containerBottom = containerTop + tocContainer.clientHeight

  // 如果激活项在可视区域上方，滚动到顶部
  if (itemTop < containerTop) {
    tocContainer.scrollTo({
      top: itemTop - 10, // 留一点边距
      behavior: 'smooth'
    })
  }
  // 如果激活项在可视区域下方，滚动到底部
  else if (itemBottom > containerBottom) {
    tocContainer.scrollTo({
      top: itemBottom - tocContainer.clientHeight + 10, // 留一点边距
      behavior: 'smooth'
    })
  }
}

// 获取滚动容器
let scrollContainer: HTMLElement | null = null
let contentObserver: MutationObserver | null = null

onMounted(() => {
  // 找到实际的滚动容器（.content），而不是 window
  scrollContainer = document.querySelector('.content')

  setTimeout(() => {
    extractHeadingsFromDOM()
    const wikiContent = document.querySelector('.markdown-renderer')
    if (wikiContent && typeof MutationObserver !== 'undefined') {
      contentObserver = new MutationObserver(() => {
        extractHeadingsFromDOM()
      })
      contentObserver.observe(wikiContent, {
        childList: true,
        subtree: true,
        characterData: true,
      })
    }
  }, 200)

  // 绑定到正确的滚动容器
  if (scrollContainer) {
    scrollContainer.addEventListener('scroll', handleScroll, { passive: true })
  }
})

onUnmounted(() => {
  if (scrollContainer) {
    scrollContainer.removeEventListener('scroll', handleScroll)
  }
  if (contentObserver) {
    contentObserver.disconnect()
    contentObserver = null
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
  padding: 12px;
  position: sticky;
  top: 80px;
  max-height: calc(100vh - 100px);
  overflow-y: auto;
  flex: 0 0 25.5%;
}

.wiki-toc__title {
  font-size: 14px;
  font-weight: 600;
  color: var(--color-text-1);
  margin-bottom: 12px;
  padding-bottom: 8px;
  border-bottom: 1px solid var(--color-border-2);
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

.wiki-toc::-webkit-scrollbar {
  width: 4px;
}

.wiki-toc::-webkit-scrollbar-track {
  background: transparent;
}

.wiki-toc::-webkit-scrollbar-thumb {
  background: var(--color-border-2);
  border-radius: 2px;
}

.wiki-toc::-webkit-scrollbar-thumb:hover {
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
    position: static;
    top: auto;
    max-height: none;
    width: 100%;
    flex: none;
  }
}
</style>
