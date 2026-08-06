<template>
  <a-modal
    v-model:visible="visible"
    class="edit-game-modal"
    title="编辑游戏信息"
    :width="modalWidth"
    :footer="false"
    :align-center="false"
    @cancel="handleCancel"
  >
    <a-form ref="formRef" :model="form" :rules="rules" layout="vertical" @submit="handleSubmit">
      <a-row :gutter="12">
        <a-col :span="12">
          <a-form-item field="title">
            <template #label>
              <div class="field-label-action">
                <span>游戏名称</span>
                <a-button
                  class="app-text-action-btn"
                  type="text"
                  size="mini"
                  html-type="button"
                  :disabled="!hasParsableWikiContent"
                  :loading="isPreparingWikiMetadataCandidates"
                  @click="importMetadataFromWiki"
                >
                  从 Wiki 提取
                </a-button>
              </div>
            </template>
            <a-input v-model="form.title" placeholder="请输入游戏名称" />
          </a-form-item>
        </a-col>
        <a-col :span="12">
          <a-form-item label="别名/英文名">
            <a-input v-model="form.title_alt" placeholder="请输入别名" />
          </a-form-item>
        </a-col>
      </a-row>

      <a-row :gutter="12">
        <a-col :span="12">
          <a-form-item label="开发商">
            <a-select
              :model-value="form.developer_ids"
              placeholder="选择开发商（可多选）"
              multiple
              allow-clear
              allow-search
              :loading="isSearchingDevelopers || isCreatingDevelopers"
              :filter-option="false"
              @search="handleDeveloperSearch"
              @update:model-value="handleDeveloperSelection"
            >
              <a-option
                v-for="d in filteredDeveloperOptions"
                :key="d.id"
                :value="d.id"
                :label="d.name"
              >
                {{ d.name }}
              </a-option>
              <a-option
                v-if="canCreateDeveloperOption"
                :value="CREATE_DEVELOPER_OPTION_VALUE"
                :label="`创建“${developerSearchQuery.trim()}”`"
              >
                创建“{{ developerSearchQuery.trim() }}”
              </a-option>
            </a-select>
          </a-form-item>
        </a-col>
        <a-col :span="12">
          <a-form-item label="发行商">
            <a-select
              :model-value="form.publisher_ids"
              placeholder="选择发行商（可多选）"
              multiple
              allow-clear
              allow-search
              :loading="isSearchingPublishers || isCreatingPublishers"
              :filter-option="false"
              @search="handlePublisherSearch"
              @update:model-value="handlePublisherSelection"
            >
              <a-option
                v-for="p in filteredPublisherOptions"
                :key="p.id"
                :value="p.id"
                :label="p.name"
              >
                {{ p.name }}
              </a-option>
              <a-option
                v-if="canCreatePublisherOption"
                :value="CREATE_PUBLISHER_OPTION_VALUE"
                :label="`创建“${publisherSearchQuery.trim()}”`"
              >
                创建“{{ publisherSearchQuery.trim() }}”
              </a-option>
            </a-select>
          </a-form-item>
        </a-col>
      </a-row>

      <a-row :gutter="12">
        <a-col :span="8">
          <a-form-item label="系列">
            <a-select
              :model-value="form.series_id"
              v-model:input-value="seriesSearchQuery"
              placeholder="选择系列"
              allow-clear
              allow-search
              :loading="isSearchingSeries || isCreatingSeries"
              :filter-option="false"
              @keydown.enter.capture.stop.prevent="handleSeriesEnter"
              @search="handleSeriesSearch"
              @update:model-value="handleSeriesSelection"
            >
              <a-option
                v-for="s in filteredSeriesOptions"
                :key="s.id"
                :value="s.id"
                :label="s.name"
              >
                {{ s.name }}
              </a-option>
              <a-option
                v-if="canCreateSeriesOption"
                :value="CREATE_SERIES_OPTION_VALUE"
                :label="`创建“${seriesSearchQuery.trim()}”`"
              >
                创建“{{ seriesSearchQuery.trim() }}”
              </a-option>
            </a-select>
          </a-form-item>
        </a-col>
        <a-col :span="8">
          <a-form-item label="发行日期">
            <a-date-picker
              v-model="releaseDate"
              :min-year="1950"
              :max-year="2100"
              placeholder="选择发行日期"
              class="w-full"
            />
          </a-form-item>
        </a-col>
        <a-col :span="8">
          <a-form-item label="可见性">
            <a-radio-group v-model="form.visibility" type="button">
              <a-radio value="public">公开</a-radio>
              <a-radio value="private">私有</a-radio>
            </a-radio-group>
          </a-form-item>
        </a-col>
      </a-row>

      <a-form-item>
        <template #label>
          <div class="summary-label">
            <span>简介</span>
            <a-button
              class="app-text-action-btn"
              type="text"
              size="mini"
              html-type="button"
              @click="showSummarySelector = true"
            >
              从 Steam 导入
            </a-button>
          </div>
        </template>
        <a-textarea
          v-model="form.summary"
          placeholder="简短描述..."
          :auto-size="{ minRows: 3, maxRows: 6 }"
          show-word-limit
        />
      </a-form-item>

      <game-file-paths-section
        :file-paths="form.file_paths"
        @update-item="handleFilePathItemUpdate"
        @add="addFilePath"
        @remove="removeFilePath"
        @browse="openFileBrowser"
      />

      <a-form-item>
        <a-space class="edit-modal-footer">
          <a-button
            class="app-text-action-btn"
            type="text"
            html-type="button"
            @click="openStandaloneMediaPage"
          >
            <template #icon><icon-launch /></template>
            素材管理
          </a-button>
          <a-button class="app-text-action-btn" type="text" html-type="button" @click="handleCancel">
            取消
          </a-button>
          <a-button type="primary" html-type="submit" :loading="isSubmitting">
            保存
          </a-button>
        </a-space>
      </a-form-item>
    </a-form>

    <!-- File Browser Modal -->
    <file-browser-modal
      v-model:visible="showFileBrowser"
      :initial-path="initialPath"
      @select="handleFileSelect"
    />

    <summary-import-modal
      :visible="showSummarySelector"
      :steam-summary-search-query="steamSummarySearchQuery"
      :is-searching-steam-summary="isSearchingSteamSummary"
      :steam-summary-search-results="steamSummarySearchResults"
      :selected-steam-summary-game="selectedSteamSummaryGame"
      :steam-summary-preview="steamSummaryPreview"
      @update:visible="showSummarySelector = $event"
      @update:steam-summary-search-query="steamSummarySearchQuery = $event"
      @search-summary="searchSteamForSummary"
      @clear-summary="handleSummarySearchClear"
      @select-summary="selectSteamSummaryGame"
      @back-summary="backToSummarySearch"
      @confirm-summary-import="confirmSummaryImport"
    />

    <edit-game-wiki-metadata-picker-modal
      :visible="wikiMetadataPickerVisible"
      :candidates="wikiMetadataCandidates"
      :is-applying-wiki-metadata="isApplyingWikiMetadata"
      @update:visible="wikiMetadataPickerVisible = $event"
      @selection-change="handleWikiMetadataCandidateSelectionChange($event.key, $event.selected)"
      @apply="applySelectedWikiMetadata"
    />
  </a-modal>
</template>

<script setup lang="ts">
import { ref } from 'vue'
import { useRouter } from 'vue-router'
import { IconLaunch } from '@arco-design/web-vue/es/icon'
import { useUiStore } from '@/stores/ui'
import type { AdminGameDetail } from '@/services/types'
import FileBrowserModal from '@/components/FileBrowserModal.vue'
import GameFilePathsSection from '@/components/edit-game/GameFilePathsSection.vue'
import EditGameWikiMetadataPickerModal from '@/components/edit-game/EditGameWikiMetadataPickerModal.vue'
import SummaryImportModal from '@/components/edit-game/import-modals/SummaryImportModal.vue'
import { useEditGameModal } from '@/composables/useEditGameModal'

interface Props {
  visible: boolean
  game: AdminGameDetail | null
}

const props = defineProps<Props>()

const emit = defineEmits<{
  'update:visible': [value: boolean]
  'success': []
  'sync': []
}>()

const uiStore = useUiStore()
const router = useRouter()
const formRef = ref()
const isSubmitting = ref(false)
const activeTab = ref('info')

const {
  CREATE_DEVELOPER_OPTION_VALUE,
  CREATE_PUBLISHER_OPTION_VALUE,
  CREATE_SERIES_OPTION_VALUE,
  addFilePath,
  applySelectedWikiMetadata,
  backToSummarySearch,
  canCreateDeveloperOption,
  canCreatePublisherOption,
  canCreateSeriesOption,
  confirmSummaryImport,
  developerSearchQuery,
  filteredDeveloperOptions,
  filteredPublisherOptions,
  filteredSeriesOptions,
  form,
  handleCancel,
  handleDeveloperSearch,
  handleDeveloperSelection,
  handleFilePathItemUpdate,
  handleFileSelect,
  handlePublisherSearch,
  handlePublisherSelection,
  handleSeriesEnter,
  handleSeriesSearch,
  handleSeriesSelection,
  handleSubmit,
  handleSummarySearchClear,
  handleWikiMetadataCandidateSelectionChange,
  hasParsableWikiContent,
  importMetadataFromWiki,
  initialPath,
  isApplyingWikiMetadata,
  isCreatingDevelopers,
  isCreatingPublishers,
  isCreatingSeries,
  isPreparingWikiMetadataCandidates,
  isSearchingDevelopers,
  isSearchingPublishers,
  isSearchingSteamSummary,
  isSearchingSeries,
  modalWidth,
  openFileBrowser,
  publisherSearchQuery,
  releaseDate,
  removeFilePath,
  rules,
  searchSteamForSummary,
  selectSteamSummaryGame,
  selectedSteamSummaryGame,
  seriesSearchQuery,
  showFileBrowser,
  showSummarySelector,
  steamSummaryPreview,
  steamSummarySearchQuery,
  steamSummarySearchResults,
  visible,
  wikiMetadataCandidates,
  wikiMetadataPickerVisible,
} = useEditGameModal({
  props,
  emit,
  uiStore,
  formRef,
  isSubmitting,
  activeTab,
})

const openStandaloneMediaPage = () => {
  if (!props.game?.public_id) return
  visible.value = false
  void router.push({ name: 'game-media', params: { publicId: props.game.public_id } })
}
</script>

<style scoped src="./edit-game/EditGameModal.css"></style>
