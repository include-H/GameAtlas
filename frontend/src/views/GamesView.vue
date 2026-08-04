<template>
  <div class="games-view">
    <!-- Header -->
    <div class="view-header">
      <div class="view-header-title-group">
        <h1 class="view-title text-gradient">{{ pageTitle }}</h1>
        <p class="view-subtitle">你收藏的每一款，都在这里。</p>
      </div>

      <a-space>
        <a-radio-group v-model="viewMode" type="button" size="medium">
          <a-radio value="grid">
            <icon-apps />
          </a-radio>
          <a-radio value="list">
            <icon-list />
          </a-radio>
        </a-radio-group>

        <a-button v-if="isAdmin" type="primary" @click="handleAddGame">
          <template #icon>
            <icon-plus />
          </template>
          添加游戏
        </a-button>
      </a-space>
    </div>

    <!-- Search and Filters -->
    <a-card class="mb-4 search-card app-glass-surface" :bordered="false">
      <a-row :gutter="[12, 12]" class="games-filters-row">
        <!-- Search -->
        <a-col :xs="24" :sm="12" :md="6" :lg="6" :xl="6" :xxl="5" class="games-filters-col games-filters-col--search">
          <div class="app-input-action-row">
            <a-input
              v-model="searchQuery"
              class="app-input-action-row__field"
              placeholder="搜索游戏"
              allow-clear
              @press-enter="handleSearch"
            >
              <template #prefix>
                <icon-search />
              </template>
            </a-input>
          </div>
        </a-col>

        <!-- Sort -->
        <a-col :xs="24" :sm="8" :md="6" :lg="6" :xl="6" :xxl="6" class="games-filters-col games-filters-col--sort">
          <a-select
            v-model="sortBy"
            :options="sortOptions"
            placeholder="排序"
          >
            <template #prefix>
              <icon-sort />
            </template>
          </a-select>
        </a-col>

        <!-- Items Per Page -->
        <a-col :xs="24" :sm="8" :md="3" :lg="3" :xl="3" :xxl="3" class="games-filters-col games-filters-col--page-size">
          <a-select
            v-model="itemsPerPage"
            :options="itemsPerPageOptions"
          />
        </a-col>

        <!-- Favorites Toggle -->
        <a-col :xs="24" :sm="8" :md="1" :lg="1" :xl="1" :xxl="1" class="games-filters-col games-filters-col--favorite">
          <a-tag
            checkable
            :checked="filterFavorites"
            color="red"
            class="favorite-toggle-tag"
            @click="updateRoute({ favorite: filterFavorites ? undefined : 'true' })"
          >
            <icon-heart-fill v-if="filterFavorites" />
            <icon-heart v-else />
            收藏
          </a-tag>
        </a-col>

        <!-- Private Toggle (admin only) -->
        <a-col v-if="isAdmin" :xs="24" :sm="8" :md="1" :lg="1" :xl="1" :xxl="1" class="games-filters-col games-filters-col--favorite">
          <a-tag
            checkable
            :checked="filterPrivate"
            color="orangered"
            class="favorite-toggle-tag"
            @click="updateRoute({ visibility: filterPrivate ? undefined : 'private' })"
          >
            <icon-lock />
            私有
          </a-tag>
        </a-col>
      </a-row>

      <!-- Active Filters -->
      <a-row v-if="hasActiveFilters" class="mt-3">
        <a-col :span="24">
          <a-space wrap>
            <span class="filter-label">当前筛选:</span>
            <a-tag
              v-if="route.query.search"
              closable
              @close="updateRoute({ search: undefined })"
            >
              搜索: {{ route.query.search }}
            </a-tag>
            <a-tag
              v-if="filterFavorites"
              closable
              @close="updateRoute({ favorite: undefined })"
            >
              仅收藏
            </a-tag>
            <a-tag
              v-if="filterPrivate && isAdmin"
              closable
              @close="updateRoute({ visibility: undefined })"
            >
              仅私有
            </a-tag>
            <a-button
              class="app-text-action-btn"
              size="small"
              type="text"
              @click="clearFilters"
            >
              清除全部
            </a-button>
          </a-space>
        </a-col>
      </a-row>
    </a-card>

    <!-- Results Count -->
    <div v-if="!hasLoadFailure" class="results-info">
      <span class="results-count">
        显示 {{ games?.length || 0 }} / {{ pagination?.total || 0 }} 个游戏
      </span>
    </div>

    <!-- Loading State -->
    <div v-if="isLoading" class="loading-container">
      <a-spin :size="24" />
      <p class="loading-text">加载中...</p>
    </div>

    <!-- Games Grid/List -->
    <div v-else-if="games && games.length > 0">
      <!-- Grid View -->
      <div v-if="viewMode === 'grid'" class="games-grid">
        <div
          v-for="game in games"
          :key="game.id"
          class="games-grid__item"
        >
          <game-card
            :game="game"
            @view="viewGame"
            @view-series="viewSeries"
            @toggle-favorite="toggleFavorite"
            @delete="handleDelete($event, game.title)"
          />
        </div>
      </div>

      <!-- List View -->
      <a-row v-else :gutter="16">
        <a-col
          v-for="game in games"
          :key="game.id"
          :span="24"
        >
          <game-card
            :game="game"
            is-list
            @view="viewGame"
            @view-series="viewSeries"
            @toggle-favorite="toggleFavorite"
            @delete="handleDelete($event, game.title)"
          />
        </a-col>
      </a-row>

      <!-- Pagination -->
      <div v-if="totalPages > 1" class="pagination-container">
        <a-pagination
          v-model:current="currentPage"
          v-model:page-size="itemsPerPage"
          :total="pagination?.total || 0"
          :page-size-options="itemsPerPageOptions.map((item) => item.value)"
          show-total
          show-jumper
          show-page-size
        />
      </div>
    </div>

    <a-empty v-else-if="hasLoadFailure" class="empty-state">
      <template #image>
        <icon-trophy :style="{ fontSize: '96px', color: 'var(--color-text-3)' }" />
      </template>
      <template #description>
        <div class="empty-description">
          <h3>加载游戏失败</h3>
          <p>当前列表请求没有成功返回，请稍后重试。</p>
        </div>
      </template>
    </a-empty>

    <!-- Empty State -->
    <a-empty v-else class="empty-state">
      <template #image>
        <icon-trophy :style="{ fontSize: '96px', color: 'var(--color-text-3)' }" />
      </template>
      <template #description>
        <div class="empty-description">
          <h3>暂无游戏</h3>
          <p>尝试调整筛选条件或搜索关键词</p>
        </div>
      </template>
      <a-button
        v-if="hasActiveFilters"
        class="app-text-action-btn"
        type="text"
        @click="clearFilters"
      >
        清除筛选
      </a-button>
    </a-empty>

    <!-- Add Game Modal -->
    <add-game-modal
      v-model:visible="showAddModal"
      :submitting="addGameSubmitting"
      @submit="handleAddGameSubmit"
    />
  </div>
</template>

<script setup lang="ts">
import { storeToRefs } from 'pinia'
import { useRoute, useRouter } from 'vue-router'
import { useGamesStore } from '@/stores/games'
import { useAuthStore } from '@/stores/auth'
import { useUiStore } from '@/stores/ui'
import GameCard from '@/components/GameCard.vue'
import AddGameModal from '@/components/AddGameModal.vue'
import { useGamesView } from '@/composables/useGamesView'
import { IconApps, IconHeart, IconHeartFill, IconList, IconLock, IconPlus, IconSearch, IconSort, IconTrophy } from '@arco-design/web-vue/es/icon'

defineOptions({
  name: 'GamesView',
})

const route = useRoute()
const router = useRouter()
const gamesStore = useGamesStore()
const authStore = useAuthStore()
const uiStore = useUiStore()
const { isAdmin } = storeToRefs(authStore)

const {
  clearFilters,
  currentPage,
  addGameSubmitting,
  filterFavorites,
  filterPrivate,
  games,
  handleAddGame,
  handleAddGameSubmit,
  handleDelete,
  handleSearch,
  hasActiveFilters,
  hasLoadFailure,
  isLoading,
  itemsPerPage,
  itemsPerPageOptions,
  pageTitle,
  pagination,
  searchQuery,
  showAddModal,
  sortBy,
  sortOptions,
  toggleFavorite,
  totalPages,
  updateRoute,
  viewGame,
  viewSeries,
  viewMode,
} = useGamesView({
  route,
  router,
  gamesStore,
  uiStore,
  isAdmin,
})
</script>

<style scoped src="./GamesView.css"></style>
