<template>
  <div class="pending-center">
    <div class="pending-center__header page-hero">
      <div class="page-hero__content">
        <h1 class="pending-center__title page-hero__title text-gradient">待处理工作台</h1>
        <p class="pending-center__subtitle page-hero__subtitle">
          队列视图：后端按待处理规则分页，每页 {{ PENDING_WORKBENCH_PAGE_SIZE }} 条，直到清空待处理游戏。
        </p>
      </div>

      <a-space>
        <a-button class="app-text-action-btn" type="text" @click="refreshWorkbench">
          <template #icon>
            <icon-refresh />
          </template>
          刷新队列
        </a-button>
      </a-space>
    </div>

    <div class="pending-center__stats">
      <a-card class="stat-card stat-card--total" :bordered="false">
        <div class="stat-card__label">待处理总数</div>
        <div class="stat-card__value">{{ totalPendingCount }}</div>
        <div class="stat-card__hint">当前第 {{ currentPage }} 页，本页 {{ currentBatchCount }} 条</div>
      </a-card>
      <a-card
        v-for="definition in pendingIssueDefinitions"
        :key="definition.key"
        class="stat-card stat-card--issue"
        :class="{ 'stat-card--active': selectedIssue === definition.key }"
        :bordered="false"
        @click="toggleIssueFilter(definition.key)"
      >
        <div class="stat-card__label">{{ definition.label }}</div>
        <div class="stat-card__value">{{ issueCounts[definition.key] || 0 }}</div>
        <div class="stat-card__hint">{{ definition.description }}</div>
      </a-card>
    </div>

    <a-card class="pending-center__filters" :bordered="false">
      <a-row :gutter="[12, 12]">
        <a-col :xs="24" :sm="12" :md="8" :lg="8">
          <a-input
            v-model="searchQuery"
            placeholder="筛选当前页游戏"
            allow-clear
          >
            <template #prefix>
              <icon-search />
            </template>
          </a-input>
        </a-col>
        <a-col :xs="24" :sm="12" :md="6" :lg="5">
          <a-select v-model="selectedIssue" placeholder="问题类型" allow-clear>
            <a-option
              v-for="definition in pendingIssueDefinitions"
              :key="definition.key"
              :value="definition.key"
            >
              {{ definition.label }}
            </a-option>
          </a-select>
        </a-col>
        <a-col :xs="24" :sm="12" :md="5" :lg="5">
          <a-select v-model="sortBy" placeholder="排序">
            <a-option value="issue-count">问题数最多优先</a-option>
            <a-option value="created-desc">最新添加优先</a-option>
            <a-option value="updated-asc">最久未更新优先</a-option>
            <a-option value="downloads-desc">下载量高优先</a-option>
          </a-select>
        </a-col>
        <a-col :xs="12" :sm="6" :md="3" :lg="3">
          <div class="filter-toggle">
            <span>仅严重项</span>
            <a-switch v-model="onlySevere" />
          </div>
        </a-col>
        <a-col :xs="12" :sm="6" :md="2" :lg="3">
          <div class="filter-toggle">
            <span>近 7 天</span>
            <a-switch v-model="onlyRecent" />
          </div>
        </a-col>
        <a-col :xs="12" :sm="6" :md="3" :lg="3">
          <div class="filter-toggle">
            <span>显示已忽略</span>
            <a-switch v-model="showIgnored" />
          </div>
        </a-col>
      </a-row>
    </a-card>

    <div class="pending-center__result-meta">
      <span>
        当前页筛选显示 {{ filteredGames.length }} 条，待处理总量 {{ totalPendingCount }} 条，已忽略 {{ ignoredOverridesCount }} 个问题
      </span>
      <div class="pending-center__result-actions">
        <a-button class="app-text-action-btn" type="text" size="small" @click="resetFilters">重置筛选</a-button>
      </div>
    </div>

    <div v-if="isLoading" class="pending-center__loading">
      <a-spin :size="24" />
      <p>正在整理待处理队列...</p>
    </div>

    <a-empty v-else-if="filteredGames.length === 0" class="pending-center__empty">
      <template #description>
        <div>
          <h3>没有符合条件的待处理项</h3>
          <p>可以尝试放宽筛选，或者先去添加新的游戏。</p>
        </div>
      </template>
    </a-empty>

    <div v-else class="pending-center__content">
      <div class="pending-center__list">
        <div
          v-for="game in filteredGames"
          :key="game.id"
          class="pending-game"
          :class="{ 'pending-game--active': activeGame?.id === game.id }"
          @click="selectGame(game)"
        >
          <div class="pending-game__media">
            <img :src="getDisplayImage(game)" :alt="game.title" />
          </div>

          <div class="pending-game__main">
            <div class="pending-game__top">
              <div>
                <h3 class="pending-game__title">{{ game.title }}</h3>
                <p class="pending-game__meta">
                  {{ formatDate(game.updated_at) }} 更新
                  <span v-if="game.release_date"> · {{ formatDate(game.release_date) }} 发售</span>
                </p>
              </div>
              <a-space size="small">
                <a-tag v-if="getIgnoredIssueDetails(game).length > 0" color="gray">
                  已忽略 {{ getIgnoredIssueDetails(game).length }} 项
                </a-tag>
                <a-tag v-if="isSevereGame(game)" color="orangered">严重</a-tag>
              </a-space>
            </div>

            <a-space wrap size="small" class="pending-game__detail-tags">
              <a-tag
                v-for="detail in getVisibleIssueDetails(game)"
                :key="detail"
                bordered
              >
                {{ getPendingIssueDetailLabel(detail) }}
              </a-tag>
              <a-tag
                v-for="detail in getIgnoredIssueDetails(game)"
                :key="`ignored-${detail}`"
                color="gray"
              >
                已忽略 {{ getPendingIssueDetailLabel(detail) }}
              </a-tag>
            </a-space>
          </div>
        </div>
        <div v-if="totalPages > 1" class="pending-center__pagination">
          <a-pagination
            :current="currentPage"
            :total="totalPendingCount"
            :page-size="PENDING_WORKBENCH_PAGE_SIZE"
            show-total
            show-jumper
            @change="changePage"
          />
        </div>
      </div>

      <a-card class="pending-center__detail" :bordered="false">
        <template #title>
          <div class="pending-center__detail-title">
            <span>待处理详情</span>
            <span v-if="activeGame" class="pending-center__detail-game">{{ activeGame.title }}</span>
          </div>
        </template>

        <div v-if="activeGame" class="detail-panel">
          <div class="detail-panel__hero">
            <img
              :src="getDetailHeroImage(activeGame)"
              :alt="activeGame.title"
              class="detail-panel__hero-image"
              :class="{ 'detail-panel__hero-image--contain': detailHeroFit === 'contain' }"
              @load="updateDetailHeroFit"
            />
          </div>

          <div class="detail-panel__section">
            <div class="detail-panel__section-title">问题概览</div>
            <a-space wrap size="small">
              <a-tag
                v-for="issue in getVisibleIssueGroups(activeGame)"
                :key="issue"
                color="arcoblue"
              >
                {{ getPendingIssueLabel(issue) }}
              </a-tag>
              <a-tag
                v-for="detail in getIgnoredIssueDetails(activeGame)"
                :key="`active-ignored-${detail}`"
                color="gray"
              >
                已忽略 {{ getPendingIssueDetailLabel(detail) }}
              </a-tag>
            </a-space>
          </div>

          <div class="detail-panel__section">
            <div class="detail-panel__section-title">缺失项清单</div>
            <div class="detail-checklist">
              <div
                v-for="detail in activeGameDetails"
                :key="detail.key"
                class="detail-checklist__item"
                :class="{ 'detail-checklist__item--ignored': detail.ignored }"
              >
                <div class="detail-checklist__main">
                  <span>{{ detail.label }}</span>
                  <span v-if="detail.reason" class="detail-checklist__reason">{{ detail.reason }}</span>
                </div>
                <div class="detail-checklist__side">
                  <span class="detail-checklist__group">{{ getPendingIssueLabel(detail.group) }}</span>
                  <a-button
                    v-if="!detail.ignored"
                    size="mini"
                    type="text"
                    status="warning"
                    @click="ignoreIssue(activeGame, detail.key)"
                  >
                    忽略
                  </a-button>
                  <a-button
                    v-else
                    size="mini"
                    type="text"
                    @click="restoreIssue(activeGame, detail.key)"
                  >
                    恢复
                  </a-button>
                </div>
              </div>
            </div>
          </div>

          <div class="detail-panel__section">
            <div class="detail-panel__section-title">当前状态</div>
            <div class="detail-overview">
              <div class="detail-overview__item">
                <span>文件</span>
                <strong>{{ activeGame.file_count || 0 }}</strong>
              </div>
              <div class="detail-overview__item">
                <span>截图</span>
                <strong>{{ activeGame.screenshot_count || 0 }}</strong>
              </div>
              <div class="detail-overview__item">
                <span>开发商</span>
                <strong>{{ activeGame.developer_count || 0 }}</strong>
              </div>
              <div class="detail-overview__item">
                <span>平台</span>
                <strong>{{ activeGame.platform_count || 0 }}</strong>
              </div>
            </div>
          </div>

          <div class="detail-panel__section">
            <div class="detail-panel__section-title">快捷处理</div>
            <a-space wrap>
              <a-button type="primary" @click="openEdit(activeGame)">
                <template #icon>
                  <icon-edit />
                </template>
                编辑资料
              </a-button>
              <a-button type="text" @click="openWiki(activeGame)">
                <template #icon>
                  <icon-book />
                </template>
                编辑 Wiki
              </a-button>
              <a-button type="text" @click="viewGame(activeGame)">
                <template #icon>
                  <icon-right />
                </template>
                游戏详情
              </a-button>
            </a-space>
          </div>
        </div>

        <a-empty v-else description="选择左侧一条游戏，查看待处理详情。" />
      </a-card>
    </div>

    <edit-game-modal
      v-model:visible="showEditModal"
      :game="editingGame"
      @success="handleEditSuccess"
    />
  </div>
</template>

<script setup lang="ts">
import { computed, onMounted, ref, watch } from 'vue'
import { useRoute, useRouter } from 'vue-router'
import { useUiStore } from '@/stores/ui'
import gamesService from '@/services/games.service'
import type { GameDetail, GameListItem } from '@/services/types'
import EditGameModal from '@/components/EditGameModal.vue'
import {
  getPendingIssueDetailLabel,
  getPendingIssueLabel,
  pendingIssueDefinitions,
  pendingIssueDetailDefinitions,
  type PendingIssueDetailKey,
  type PendingIssueKey,
} from '@/utils/pendingIssues'
import { createDetailRouteQuery } from '@/utils/navigation'
import { PENDING_WORKBENCH_PAGE_SIZE, usePendingWorkbench } from '@/composables/usePendingWorkbench'
import { IconBook, IconEdit, IconRefresh, IconRight, IconSearch } from '@arco-design/web-vue/es/icon'

defineOptions({
  name: 'PendingCenterView',
})

const route = useRoute()
const router = useRouter()
const uiStore = useUiStore()

const editingGame = ref<GameDetail | null>(null)
const showEditModal = ref(false)
const detailHeroFit = ref<'cover' | 'contain'>('cover')

const {
  activeGame,
  changePage,
  currentBatchCount,
  currentPage,
  filteredGames,
  getIssueEvaluation,
  getIgnoredIssueDetails,
  getVisibleIssueGroups,
  getVisibleIssueDetails,
  ignoredOverridesCount,
  ignoreIssue,
  isSevereGame,
  isLoading,
  issueCounts,
  loadWorkbenchGames,
  onlyRecent,
  onlySevere,
  resetFilters,
  restoreIssue,
  reviewOverrideMap,
  searchQuery,
  selectedIssue,
  showIgnored,
  sortBy,
  totalPages,
  totalPendingCount,
} = usePendingWorkbench({
  addAlert: (message, type) => uiStore.addAlert(message, type),
})

const placeholderImage = 'data:image/svg+xml,%3Csvg xmlns="http://www.w3.org/2000/svg" viewBox="0 0 24 24"%3E%3Cpath fill="%23424242" d="M12 2C6.48 2 2 6.48 2 12s4.48 10 10 10 10-4.48 10-10S17.52 2 12 2zm-1 17.93c-3.95-.49-7-3.85-7-7.93 0-.62.08-1.21.21-1.79L9 15v1c0 1.1.9 2 2 2v1.93zm6.9-2.54c-.26-.81-1-1.39-1.9-1.39h-1v-3c0-.55-.45-1-1-1H8v-2h2c.55 0 1-.45 1-1V7h2c1.1 0 2-.9 2-2v-.41c2.93 1.19 5 4.06 5 7.41 0 2.08-.8 3.97-2.1 5.39z"/%3E%3C/svg%3E'

type PendingIssueDetailDefinition = (typeof pendingIssueDetailDefinitions)[number]
const pendingIssueDetailDefinitionMap = pendingIssueDetailDefinitions.reduce<
  Record<PendingIssueDetailKey, PendingIssueDetailDefinition>
>((acc, item) => {
  acc[item.key] = item
  return acc
}, {} as Record<PendingIssueDetailKey, PendingIssueDetailDefinition>)

watch(
  activeGame,
  () => {
    detailHeroFit.value = activeGame.value?.banner_image ? 'cover' : 'contain'
  },
  { immediate: true },
)

const activeGameDetails = computed(() => {
  if (!activeGame.value) return []
  const activeOverrides = reviewOverrideMap.value[String(activeGame.value.id)] || []
  const activeOverrideReasonMap = Object.fromEntries(activeOverrides.map((item) => [item.issue_key, item.reason || '']))
  const evaluation = getIssueEvaluation(activeGame.value)

  return [
    ...evaluation.details.map((key) => {
      const definition = pendingIssueDetailDefinitionMap[key]
      return definition ? { ...definition, ignored: false, reason: '' } : null
    }),
    ...evaluation.ignoredDetails.map((key) => {
      const definition = pendingIssueDetailDefinitionMap[key]
      return definition ? { ...definition, ignored: true, reason: activeOverrideReasonMap[key] || '' } : null
    }),
  ].filter((item): item is NonNullable<typeof item> => Boolean(item))
})

const getDisplayImage = (game: GameListItem) => {
  return game.cover_image || game.banner_image || game.primary_screenshot || placeholderImage
}

const getDetailHeroImage = (game: GameListItem) => {
  return game.banner_image || game.cover_image || game.primary_screenshot || placeholderImage
}

const updateDetailHeroFit = (event: Event) => {
  const target = event.target as HTMLImageElement | null
  if (!target?.naturalWidth || !target.naturalHeight) {
    detailHeroFit.value = 'cover'
    return
  }

  const aspectRatio = target.naturalWidth / target.naturalHeight
  detailHeroFit.value = aspectRatio >= 1.5 ? 'cover' : 'contain'
}

const formatDate = (value?: string | null) => {
  if (!value) return '未知时间'
  const date = new Date(value)
  if (Number.isNaN(date.getTime())) return value
  return `${date.getFullYear()}-${String(date.getMonth() + 1).padStart(2, '0')}-${String(date.getDate()).padStart(2, '0')}`
}

const selectGame = (game: GameListItem) => {
  activeGame.value = game
}

const toggleIssueFilter = (key: PendingIssueKey) => {
  selectedIssue.value = selectedIssue.value === key ? undefined : key
}

const openEdit = async (game: GameListItem) => {
  if (!game.public_id) return
  try {
    editingGame.value = await gamesService.getGame(game.public_id)
    showEditModal.value = true
  } catch {
    uiStore.addAlert('加载游戏详情失败', 'error')
  }
}

const openWiki = (game: GameListItem) => {
  if (!game.public_id) return
  router.push({
    name: 'wiki-edit',
    params: { publicId: game.public_id },
    query: createDetailRouteQuery(route),
  })
}

const viewGame = (game: GameListItem) => {
  if (!game.public_id) return
  router.push({
    name: 'game-detail',
    params: { publicId: game.public_id },
    query: createDetailRouteQuery(route),
  })
}

const refreshWorkbench = async () => {
  await loadWorkbenchGames()
}

const handleEditSuccess = async () => {
  showEditModal.value = false
  await loadWorkbenchGames()
}

onMounted(async () => {
  await loadWorkbenchGames()
})
</script>

<style scoped>
.pending-center {
  display: flex;
  flex-direction: column;
  gap: 16px;
  min-height: calc(100vh - 120px);
}

.pending-center__header {
  align-items: flex-start;
}

.pending-center__title {
  margin: 0;
}

.pending-center__subtitle {
  margin: 0;
}

.pending-center__stats {
  display: grid;
  grid-template-columns: repeat(5, minmax(0, 1fr));
  gap: 16px;
}

.stat-card {
  border-radius: 18px;
  cursor: pointer;
  transition: transform 0.2s ease, box-shadow 0.2s ease;
}

.stat-card:hover {
  transform: translateY(-2px);
  box-shadow: 0 16px 36px rgba(15, 23, 42, 0.08);
}

.stat-card--total {
  cursor: default;
  background:
    linear-gradient(135deg, rgba(26, 159, 255, 0.14), rgba(0, 180, 42, 0.1)),
    var(--app-card-surface);
}

.stat-card--issue {
  background: var(--app-card-surface);
  border: 1px solid var(--app-card-border);
  backdrop-filter: blur(var(--app-card-backdrop-blur));
  -webkit-backdrop-filter: blur(var(--app-card-backdrop-blur));
}

.stat-card--active {
  box-shadow: inset 0 0 0 2px rgb(var(--arcoblue-6));
}

.stat-card__label {
  color: var(--color-text-2);
  font-size: 13px;
}

.stat-card__value {
  margin-top: 6px;
  font-size: 30px;
  font-weight: 700;
  color: var(--color-text-1);
}

.stat-card__hint {
  margin-top: 8px;
  color: var(--color-text-3);
  font-size: 12px;
}

.pending-center__filters {
  border-radius: 18px;
}

.filter-toggle {
  height: 100%;
  min-height: 32px;
  display: flex;
  align-items: center;
  justify-content: space-between;
  gap: 12px;
  padding: 0 4px;
}

.pending-center__result-meta {
  display: flex;
  justify-content: space-between;
  align-items: center;
  gap: 12px;
  color: var(--color-text-3);
}

.pending-center__result-actions {
  display: flex;
  align-items: center;
  gap: 12px;
}

.pending-center__loading {
  min-height: 320px;
  display: flex;
  flex-direction: column;
  align-items: center;
  justify-content: center;
  gap: 12px;
  color: var(--color-text-3);
}

.pending-center__content {
  display: grid;
  grid-template-columns: minmax(0, 1fr) clamp(360px, 28vw, 460px);
  gap: 10px;
  align-items: start;
}

.pending-center__list {
  display: flex;
  flex-direction: column;
  gap: 12px;
}

.pending-center__pagination {
  display: flex;
  justify-content: flex-end;
  padding-top: 4px;
}

.pending-game {
  display: grid;
  grid-template-columns: 112px minmax(0, 1fr);
  gap: 10px;
  padding: 14px;
  border-radius: 18px;
  border: 1px solid var(--app-card-border);
  background: var(--app-card-surface);
  backdrop-filter: blur(var(--app-card-backdrop-blur));
  -webkit-backdrop-filter: blur(var(--app-card-backdrop-blur));
  cursor: pointer;
  transition: border-color 0.2s ease, box-shadow 0.2s ease, transform 0.2s ease;
}

.pending-game:hover,
.pending-game--active {
  border-color: rgba(22, 93, 255, 0.32);
  box-shadow: 0 18px 36px rgba(15, 23, 42, 0.08);
  transform: translateY(-1px);
}

.pending-game__media {
  width: 112px;
  height: 148px;
  border-radius: 14px;
  overflow: hidden;
  background: var(--color-fill-2);
}

.pending-game__media img {
  width: 100%;
  height: 100%;
  object-fit: cover;
}

.pending-game__main {
  min-width: 0;
  display: flex;
  flex-direction: column;
  gap: 10px;
}

.pending-game__top {
  display: flex;
  justify-content: space-between;
  align-items: flex-start;
  gap: 12px;
}

.pending-game__title {
  margin: 0;
  font-size: 18px;
  color: var(--color-text-1);
}

.pending-game__meta {
  margin: 4px 0 0;
  color: var(--color-text-3);
  font-size: 12px;
}

.pending-game__detail-tags {
  display: flex;
}

.pending-center__detail {
  position: sticky;
  top: 12px;
  width: 100%;
  max-width: 460px;
  justify-self: end;
  border-radius: 18px;
}

.pending-center__detail-title {
  display: flex;
  justify-content: space-between;
  gap: 12px;
}

.pending-center__detail-game {
  color: var(--color-text-3);
  font-size: 13px;
}

.detail-panel {
  display: flex;
  flex-direction: column;
  gap: 10px;
}

.detail-panel__hero {
  width: 100%;
  aspect-ratio: 16 / 9;
  max-height: 300px;
  border-radius: 16px;
  overflow: hidden;
  background: var(--color-fill-2);
  display: flex;
  align-items: center;
  justify-content: center;
}

.detail-panel__hero-image {
  width: 100%;
  height: 100%;
  object-fit: cover;
  display: block;
}

.detail-panel__hero-image--contain {
  width: auto;
  max-width: 100%;
  height: auto;
  max-height: 100%;
  object-fit: contain;
  padding: 12px;
}

.detail-panel__section {
  display: flex;
  flex-direction: column;
  gap: 10px;
}

.detail-panel__section-title {
  font-size: 14px;
  font-weight: 600;
  color: var(--color-text-1);
}

.detail-checklist {
  display: flex;
  flex-direction: column;
  gap: 8px;
}

.detail-checklist__item {
  display: flex;
  justify-content: space-between;
  gap: 12px;
  padding: 10px 12px;
  border-radius: 12px;
  background: var(--color-fill-1);
  color: var(--color-text-2);
}

.detail-checklist__item--ignored {
  opacity: 0.72;
}

.detail-checklist__main {
  display: flex;
  flex-direction: column;
  gap: 4px;
}

.detail-checklist__reason {
  color: var(--color-text-3);
  font-size: 12px;
}

.detail-checklist__side {
  display: flex;
  align-items: center;
  gap: 10px;
}

.detail-checklist__group {
  color: var(--color-text-3);
  font-size: 12px;
}

.detail-overview {
  display: grid;
  grid-template-columns: repeat(2, minmax(0, 1fr));
  gap: 10px;
}

.detail-overview__item {
  padding: 12px;
  border-radius: 12px;
  background: var(--color-fill-1);
  display: flex;
  flex-direction: column;
  gap: 6px;
  color: var(--color-text-3);
}

.detail-overview__item strong {
  color: var(--color-text-1);
  font-size: 22px;
}

@media (max-width: 1100px) {
  .pending-center__stats {
    grid-template-columns: repeat(3, minmax(0, 1fr));
  }

  .pending-center__filters :deep(.arco-row) {
    row-gap: 8px;
  }

  .pending-center__content {
    grid-template-columns: 1fr;
  }

  .pending-center__detail {
    position: static;
    max-width: none;
    justify-self: stretch;
  }
}

@media (max-width: 768px) {
  .pending-center__header,
  .pending-center__result-meta {
    flex-direction: column;
    align-items: stretch;
  }

  .pending-center__filters {
    border-radius: 16px;
  }

  .pending-center__result-actions {
    justify-content: space-between;
  }

  .pending-center__stats {
    grid-template-columns: repeat(2, minmax(0, 1fr));
  }

  .pending-game {
    grid-template-columns: 88px minmax(0, 1fr);
  }

  .pending-game__top,
  .pending-center__detail-title {
    flex-direction: column;
    align-items: flex-start;
  }

  .filter-toggle {
    min-height: 36px;
    padding: 0;
  }

  .pending-game__media {
    width: 88px;
    height: 118px;
  }
}

@media (max-width: 576px) {
  .pending-center__stats {
    grid-template-columns: 1fr;
  }

  .pending-game {
    grid-template-columns: 1fr;
    gap: 12px;
  }

  .pending-game__media {
    width: 100%;
    height: auto;
    aspect-ratio: 16 / 9;
  }

  .pending-center__result-actions {
    flex-direction: column;
    align-items: stretch;
  }
}
</style>
