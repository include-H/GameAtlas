<template>
  <div class="pending-center">
    <div class="pending-center__header page-hero">
      <div class="page-hero__content">
        <h1 class="pending-center__title page-hero__title text-gradient">待处理工作台</h1>
        <p class="pending-center__subtitle page-hero__subtitle">
          队列视图：后端原生处理待处理筛选、排序与分页，每页 {{ PENDING_WORKBENCH_PAGE_SIZE }} 条。
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
        <div class="stat-card__hint">当前第 {{ currentPage }} 页，本页 {{ pageGameCount }} 条</div>
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
        <div class="stat-card__value">{{ pendingIssueCounts[definition.key] || 0 }}</div>
        <div class="stat-card__hint">{{ definition.description }}</div>
      </a-card>
    </div>

    <a-card class="pending-center__filters" :bordered="false">
      <a-row :gutter="[12, 12]">
        <a-col :xs="24" :sm="12" :md="8" :lg="8">
          <a-input
            v-model="searchQuery"
            placeholder="搜索待处理队列"
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
        当前页返回 {{ pendingGames.length }} 条，待处理总量 {{ totalPendingCount }} 条，已忽略 {{ pendingIssueIgnoredTotal }} 个问题
      </span>
      <div class="pending-center__result-actions">
        <a-button class="app-text-action-btn" type="text" size="small" @click="resetFilters">重置筛选</a-button>
      </div>
    </div>

    <div v-if="isLoading" class="pending-center__loading">
      <a-spin :size="24" />
      <p>正在整理待处理队列...</p>
    </div>

    <a-empty v-else-if="pendingGames.length === 0" class="pending-center__empty">
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
          v-for="game in pendingGames"
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
                :key="detail.key"
                bordered
              >
                {{ getPendingIssueDetailLabel(detail.key) }}
              </a-tag>
              <a-tag
                v-for="detail in getIgnoredIssueDetails(game)"
                :key="`ignored-${detail.key}`"
                color="gray"
              >
                已忽略 {{ getPendingIssueDetailLabel(detail.key) }}
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
            <div
              class="detail-panel__hero-backdrop"
              :style="{ backgroundImage: detailHeroSrc ? `url(${detailHeroSrc})` : 'none' }"
            />
            <img
              :src="detailHeroSrc"
              :alt="activeGame.title"
              class="detail-panel__hero-image"
              :class="{ 'detail-panel__hero-image--contain': detailHeroFit === 'contain' }"
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
                :key="`active-ignored-${detail.key}`"
                color="gray"
              >
                已忽略 {{ getPendingIssueDetailLabel(detail.key) }}
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
                    class="app-text-action-btn"
                    size="mini"
                    type="text"
                    status="warning"
                    @click="ignoreIssue(activeGame, detail.key)"
                  >
                    忽略
                  </a-button>
                  <a-button
                    v-else
                    class="app-text-action-btn"
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
              <a-button class="app-text-action-btn" type="text" @click="openWiki(activeGame)">
                <template #icon>
                  <icon-book />
                </template>
                编辑 Wiki
              </a-button>
              <a-button class="app-text-action-btn" type="text" @click="viewGame(activeGame)">
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
import { useRouter } from 'vue-router'
import { useUiStore } from '@/stores/ui'
import EditGameModal from '@/components/EditGameModal.vue'
import { PENDING_WORKBENCH_PAGE_SIZE } from '@/composables/usePendingWorkbench'
import { usePendingCenterView } from '@/composables/usePendingCenterView'
import { IconBook, IconEdit, IconRefresh, IconRight, IconSearch } from '@arco-design/web-vue/es/icon'

defineOptions({
  name: 'PendingCenterView',
})

const router = useRouter()
const uiStore = useUiStore()

const {
  activeGame,
  activeGameDetails,
  changePage,
  pageGameCount,
  currentPage,
  detailHeroFit,
  detailHeroSrc,
  editingGame,
  pendingGames,
  formatDate,
  getDisplayImage,
  getIgnoredIssueDetails,
  getPendingIssueDetailLabel,
  getPendingIssueLabel,
  getVisibleIssueGroups,
  getVisibleIssueDetails,
  handleEditSuccess,
  pendingIssueIgnoredTotal,
  ignoreIssue,
  isSevereGame,
  isLoading,
  pendingIssueCounts,
  onlyRecent,
  onlySevere,
  openEdit,
  openWiki,
  pendingIssueDefinitions,
  refreshWorkbench,
  resetFilters,
  restoreIssue,
  searchQuery,
  selectGame,
  selectedIssue,
  showEditModal,
  showIgnored,
  sortBy,
  toggleIssueFilter,
  totalPages,
  totalPendingCount,
  viewGame,
} = usePendingCenterView({
  router,
  uiStore,
})
</script>
<style scoped src="./PendingCenterView.css"></style>
