<template>
  <a-modal
    v-model:visible="visible"
    title="编辑游戏信息"
    :width="800"
    :footer="false"
    @cancel="handleCancel"
  >
    <a-form ref="formRef" :model="form" :rules="rules" layout="vertical" @submit="handleSubmit">
      <a-row :gutter="16">
        <a-col :span="12">
          <a-form-item field="title" label="游戏名称">
            <a-input v-model="form.title" placeholder="请输入游戏名称" />
          </a-form-item>
        </a-col>
        <a-col :span="12">
          <a-form-item label="别名/英文名">
            <a-input v-model="form.title_alt" placeholder="请输入别名" />
          </a-form-item>
        </a-col>
      </a-row>

      <a-row :gutter="16">
        <a-col :span="12">
          <a-form-item label="开发商">
            <a-select
              v-model="form.developer"
              placeholder="选择开发商"
              allow-clear
              allow-search
              allow-create
              :loading="isSearchingDevelopers"
              :remote-search="true"
              :on-search="handleDeveloperSearch"
            >
              <a-option
                v-for="d in filteredDeveloperOptions"
                :key="d.id"
                :value="d.id"
                :label="d.name"
              >
                {{ d.name }}
              </a-option>
            </a-select>
          </a-form-item>
        </a-col>
        <a-col :span="12">
          <a-form-item label="发行商">
            <a-select
              v-model="form.publisher"
              placeholder="选择发行商"
              allow-clear
              allow-search
              allow-create
              :loading="isSearchingPublishers"
              :remote-search="true"
              :on-search="handlePublisherSearch"
            >
              <a-option
                v-for="p in filteredPublisherOptions"
                :key="p.id"
                :value="p.id"
                :label="p.name"
              >
                {{ p.name }}
              </a-option>
            </a-select>
          </a-form-item>
        </a-col>
      </a-row>

      <a-row :gutter="16">
        <a-col :span="8">
          <a-form-item label="发行日期">
            <a-date-picker
              v-model="releaseDate"
              :min-year="1950"
              :max-year="2100"
              placeholder="选择发行日期"
              class="w-full"
              @change="handleDateChange"
            />
          </a-form-item>
        </a-col>
        <a-col :span="8">
          <a-form-item label="游戏引擎">
            <a-input v-model="form.engine" placeholder="如：Unity, Unreal" />
          </a-form-item>
        </a-col>
        <a-col :span="8">
          <a-form-item label="平台">
            <a-select
              v-model="form.platform"
              placeholder="选择或输入平台（可多选）"
              multiple
              allow-clear
              allow-search
              allow-create
            >
              <a-option
                v-for="p in platformOptions"
                :key="p.id"
                :value="p.id"
                :label="p.name"
              >
                {{ p.name }}
              </a-option>
            </a-select>
          </a-form-item>
        </a-col>
      </a-row>

      <a-form-item label="系列">
        <a-select
          v-model="form.series"
          placeholder="选择系列"
          allow-clear
          allow-search
          allow-create
          :loading="isSearchingSeries"
          :remote-search="true"
          :on-search="handleSeriesSearch"
        >
          <a-option
            v-for="s in filteredSeriesOptions"
            :key="s.id"
            :value="s.id"
            :label="s.name"
          >
            {{ s.name }}
          </a-option>
        </a-select>
      </a-form-item>

      <a-form-item>
        <template #label>
          <div class="summary-label">
            <span>简介</span>
            <a-button
              type="text"
              size="mini"
              @click="showSummarySelector = true"
            >
              从 Steam 导入
            </a-button>
          </div>
        </template>
        <a-textarea
          v-model="form.summary"
          placeholder="简短描述..."
          :auto-size="{ minRows: 2, maxRows: 4 }"
          show-word-limit
        />
      </a-form-item>

      <!-- 游戏文件路径 -->
      <a-form-item label="游戏文件路径">
        <div class="file-paths-container">
          <div v-for="(item, index) in form.file_paths" :key="index" class="file-path-item">
            <div class="file-path-row">
              <a-input
                v-model="item.path"
                placeholder="游戏文件路径"
                class="file-path-input"
              >
                <template #prepend>
                  <span class="path-index">{{ index + 1 }}</span>
                </template>
              </a-input>
              <a-input
                v-model="item.label"
                placeholder="版本备注"
                class="file-label-input"
              />
              <a-button @click="openFileBrowser(index)">
                <template #icon>
                  <icon-folder />
                </template>
                浏览
              </a-button>
                <a-button
                  type="text"
                  status="danger"
                  @click="removeFilePath(index)"
                >
                <icon-minus />
              </a-button>
            </div>
          </div>
          
          <a-button
            type="dashed"
            long
            @click="addFilePath"
            :style="{ marginTop: '8px' }"
          >
            <template #icon>
              <icon-plus />
            </template>
            添加文件路径
          </a-button>
        </div>
      </a-form-item>


      <!-- 封面图和截图 -->
      <a-row :gutter="16">
        <!-- 封面图 -->
        <a-col :span="8">
          <a-form-item label="封面图">
            <div class="media-section">
              <div class="media-frame media-frame--cover">
                <div v-if="form.cover_image" class="media-preview">
                  <a-image
                    :src="form.cover_image"
                    :alt="form.title"
                    width="100%"
                    height="100%"
                    fit="cover"
                    hide-footer
                  />
                  <div class="media-overlay">
                    <div class="media-overlay-actions">
                      <button class="media-action-button" type="button" @click.stop="showCoverSelector = true">
                        <icon-image />
                      </button>
                      <button class="media-action-button media-action-button--danger" type="button" @click.stop="removeCover">
                        <icon-delete />
                      </button>
                    </div>
                  </div>
                </div>
                <div
                  v-else
                  class="media-empty-action"
                  role="button"
                  tabindex="0"
                  @click="showCoverSelector = true"
                >
                  <icon-image class="media-empty-icon" />
                  <span class="media-empty-title">未设置封面</span>
                  <span class="media-empty-subtitle">点击选择图片</span>
                </div>
              </div>
            </div>
          </a-form-item>
        </a-col>

        <!-- 横幅图 -->
        <a-col :span="8">
          <a-form-item label="横幅图">
            <div class="media-section">
              <div class="media-frame media-frame--banner">
                <div v-if="form.banner_image" class="media-preview">
                  <a-image
                    :src="form.banner_image"
                    :alt="form.title"
                    width="100%"
                    height="100%"
                    fit="cover"
                    hide-footer
                  />
                  <div class="media-overlay">
                    <div class="media-overlay-actions">
                      <button class="media-action-button" type="button" @click.stop="showBannerSelector = true">
                        <icon-image />
                      </button>
                      <button class="media-action-button media-action-button--danger" type="button" @click.stop="removeBanner">
                        <icon-delete />
                      </button>
                    </div>
                  </div>
                </div>
                <div
                  v-else
                  class="media-empty-action"
                  role="button"
                  tabindex="0"
                  @click="showBannerSelector = true"
                >
                  <icon-image class="media-empty-icon" />
                  <span class="media-empty-title">未设置横幅</span>
                  <span class="media-empty-subtitle">点击选择图片</span>
                </div>
              </div>
            </div>
          </a-form-item>
        </a-col>

        <!-- 截图 -->
        <a-col :span="8">
          <a-form-item label="截图">
            <div class="media-section">
              <div v-if="form.screenshots.length === 0" class="media-frame media-frame--banner">
                <div
                  class="media-empty-action"
                  role="button"
                  tabindex="0"
                  @click="showScreenshotSelector = true"
                >
                  <icon-image class="media-empty-icon" />
                  <span class="media-empty-title">未设置截图</span>
                  <span class="media-empty-subtitle">点击添加截图</span>
                </div>
              </div>
              <a-image-preview-group v-else infinite>
                <div class="screenshots-grid">
                  <div
                    v-for="(url, index) in form.screenshots"
                    :key="index"
                    class="screenshot-thumb"
                  >
                    <a-image
                      :src="url"
                      width="100%"
                      height="100%"
                      fit="cover"
                      hide-footer
                    />
                    <div class="screenshot-overlay">
                      <button class="media-action-button media-action-button--danger" type="button" @click.stop="removeScreenshot(index)">
                        <icon-delete />
                      </button>
                    </div>
                  </div>
                  <div
                    class="screenshot-add-tile"
                    role="button"
                    tabindex="0"
                    @click="showScreenshotSelector = true"
                  >
                    <icon-image class="media-empty-icon" />
                    <span class="media-empty-title">添加截图</span>
                  </div>
                </div>
              </a-image-preview-group>
            </div>
          </a-form-item>
        </a-col>
      </a-row>

      <a-form-item>
        <a-space style="justify-content: flex-end; width: 100%">
          <a-button @click="handleCancel">取消</a-button>
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

    <!-- Summary Selector Modal -->
    <a-modal
      v-model:visible="showSummarySelector"
      title="导入 Steam 简介"
      :width="800"
      :footer="false"
    >
      <div class="cover-selector-content">
        <div class="steam-search-section">
          <a-input-search
            v-model="steamSummarySearchQuery"
            placeholder="搜索 Steam 游戏..."
            :loading="isSearchingSteamSummary"
            @search="searchSteamForSummary"
            @clear="handleSummarySearchClear"
            allow-clear
          >
            <template #prepend>
              <icon-cloud-download />
            </template>
          </a-input-search>

          <div
            v-if="steamSummarySearchResults.length > 0 && !selectedSteamSummaryGame"
            class="steam-search-results"
          >
            <div class="steam-search-title">选择游戏</div>
            <div
              v-for="game in steamSummarySearchResults"
              :key="game.id"
              class="steam-search-result-item"
              @click="selectSteamSummaryGame(game)"
            >
              <img :src="game.tinyImage" :alt="game.name" />
              <div class="steam-result-info">
                <div class="steam-result-name">{{ game.name }}</div>
                <div class="steam-result-meta">{{ game.releaseDate || '' }}</div>
              </div>
            </div>
          </div>

          <div v-if="selectedSteamSummaryGame" class="steam-summary-section">
            <div class="steam-search-title">
              {{ selectedSteamSummaryGame.name }} 的简介
              <a-button type="text" size="mini" @click="backToSummarySearch">返回</a-button>
            </div>

            <div v-if="steamSummaryPreview" class="steam-summary-preview">
              {{ steamSummaryPreview }}
            </div>

            <a-empty
              v-else-if="!isSearchingSteamSummary"
              description="Steam 未返回可用简介"
              class="steam-summary-empty"
            />

            <a-button
              v-if="steamSummaryPreview"
              type="primary"
              long
              @click="confirmSummaryImport"
            >
              导入这段简介
            </a-button>
          </div>
        </div>
      </div>
    </a-modal>

    <!-- Cover Selector Modal -->
    <a-modal
      v-model:visible="showCoverSelector"
      title="选择封面图"
      :width="700"
      :footer="false"
    >
      <div class="cover-selector-content">
        <!-- Steam 搜索 -->
        <div class="steam-search-section">
          <a-input-search
            v-model="steamCoverSearchQuery"
            placeholder="搜索 Steam 游戏..."
            :loading="isSearchingSteamCover"
            @search="searchSteamForCover"
            @clear="handleCoverSearchClear"
            allow-clear
          >
            <template #prepend>
              <icon-cloud-download />
            </template>
          </a-input-search>

          <!-- Steam 搜索结果 -->
          <div v-if="steamCoverSearchResults.length > 0 && !selectedSteamGame" class="steam-search-results">
            <div class="steam-search-title">选择游戏</div>
            <div
              v-for="game in steamCoverSearchResults"
              :key="game.id"
              class="steam-search-result-item"
              @click="selectSteamCoverGame(game)"
            >
              <img :src="game.tinyImage" :alt="game.name" />
              <div class="steam-result-info">
                <div class="steam-result-name">{{ game.name }}</div>
                <div class="steam-result-meta">{{ game.releaseDate || '' }}</div>
              </div>
            </div>
          </div>

          <!-- Steam 封面图片选择 -->
          <div v-if="selectedSteamGame && steamCoverImages.length > 0" class="steam-images-section">
            <div class="steam-search-title">
              {{ selectedSteamGame.name }} 的封面
              <a-button type="text" size="mini" @click="backToCoverGameSearch">返回</a-button>
            </div>
            <div class="steam-images-grid">
              <div
                v-for="(image, index) in steamCoverImages"
                :key="index"
                class="steam-image-item"
                :class="{ 'steam-image-selected': selectedCoverImage === image }"
                @click="selectedCoverImage = image"
              >
                <img :src="image" />
              </div>
            </div>
            <a-button
              v-if="selectedCoverImage"
              type="primary"
              long
              :loading="isSearchingSteamCover"
              @click="downloadSelectedSteamCover"
            >
              下载选中的封面
            </a-button>
          </div>
        </div>

        <a-divider />

        <!-- 本地上传 -->
        <a-upload
          :action="uploadAction"
          :data="uploadData"
          :headers="uploadHeaders"
          :show-file-list="false"
          accept="image/*"
          @success="handleCoverUploadSuccess"
          @error="handleCoverUploadError"
        >
          <a-button type="dashed" long>
            <template #icon>
              <icon-upload />
            </template>
            点击上传本地图片
          </a-button>
        </a-upload>

        <a-divider>或从 URL 加载</a-divider>

        <!-- URL 加载 -->
        <a-input
          v-model="coverSearchUrl"
          placeholder="输入图片 URL..."
          @press-enter="loadCoverFromUrl"
        >
          <template #append>
            <a-button type="primary" @click="loadCoverFromUrl">
              加载
            </a-button>
          </template>
        </a-input>
        <div v-if="coverPreviewUrl" class="cover-preview-large">
          <img :src="coverPreviewUrl" @error="handleCoverError" />
        </div>
        <div class="cover-selector-actions">
          <a-button @click="showCoverSelector = false">取消</a-button>
          <a-button type="primary" :disabled="!coverPreviewUrl" :loading="isDownloadingCover" @click="confirmCoverSelection">
            确定
          </a-button>
        </div>
      </div>
    </a-modal>

    <!-- Banner Selector Modal -->
    <a-modal
      v-model:visible="showBannerSelector"
      title="选择横幅图"
      :width="800"
      :footer="false"
    >
      <div class="cover-selector-content">
        <!-- Steam 搜索 -->
        <div class="steam-search-section">
          <a-input-search
            v-model="steamBannerSearchQuery"
            placeholder="搜索 Steam 游戏..."
            :loading="isSearchingSteamBanner"
            @search="searchSteamForBanner"
            @clear="handleBannerSearchClear"
            allow-clear
          >
            <template #prepend>
              <icon-cloud-download />
            </template>
          </a-input-search>

          <!-- Steam 搜索结果 -->
          <div v-if="steamBannerSearchResults.length > 0 && !selectedSteamBannerGame" class="steam-search-results">
            <div class="steam-search-title">选择游戏</div>
            <div
              v-for="game in steamBannerSearchResults"
              :key="game.id"
              class="steam-search-result-item"
              @click="selectSteamBannerGame(game)"
            >
              <img :src="game.tinyImage" :alt="game.name" />
              <div class="steam-result-info">
                <div class="steam-result-name">{{ game.name }}</div>
                <div class="steam-result-meta">{{ game.releaseDate || '' }}</div>
              </div>
            </div>
          </div>

          <!-- Steam 横幅图片选择 -->
          <div v-if="selectedSteamBannerGame && steamBannerImages.length > 0" class="steam-images-section">
            <div class="steam-search-title">
              {{ selectedSteamBannerGame.name }} 的横幅
              <a-button type="text" size="mini" @click="backToBannerGameSearch">返回</a-button>
            </div>
            <div class="steam-images-grid">
              <div
                v-for="(image, index) in steamBannerImages"
                :key="index"
                class="steam-image-item banner-thumb"
                :class="{ 'steam-image-selected': selectedBannerImage === image }"
                @click="selectedBannerImage = image"
              >
                <img :src="image" />
              </div>
            </div>
            <a-button
              v-if="selectedBannerImage"
              type="primary"
              long
              :loading="isSearchingSteamBanner"
              @click="downloadSelectedSteamBanner"
            >
              下载选中的横幅
            </a-button>
          </div>
        </div>

        <a-divider />

        <!-- 本地上传 -->
        <a-upload
          :action="bannerUploadAction"
          :data="bannerUploadData"
          :headers="uploadHeaders"
          :show-file-list="false"
          accept="image/*"
          @success="handleBannerUploadSuccess"
          @error="handleBannerUploadError"
        >
          <a-button type="dashed" long>
            <template #icon>
              <icon-upload />
            </template>
            点击上传本地图片
          </a-button>
        </a-upload>

        <a-divider>或从 URL 加载</a-divider>

        <!-- URL 加载 -->
        <a-input
          v-model="bannerSearchUrl"
          placeholder="输入图片 URL..."
          @press-enter="loadBannerFromUrl"
        >
          <template #append>
            <a-button type="primary" @click="loadBannerFromUrl">
              加载
            </a-button>
          </template>
        </a-input>
        <div v-if="bannerPreviewUrl" class="cover-preview-large">
          <img :src="bannerPreviewUrl" @error="handleCoverError" />
        </div>
        <div class="cover-selector-actions">
          <a-button @click="showBannerSelector = false">取消</a-button>
          <a-button type="primary" :disabled="!bannerPreviewUrl" :loading="isDownloadingBanner" @click="confirmBannerSelection">
            确定
          </a-button>
        </div>
      </div>
    </a-modal>

    <!-- Screenshot Selector Modal -->
    <a-modal
      v-model:visible="showScreenshotSelector"
      title="添加截图"
      :width="800"
      :footer="false"
    >
      <div class="screenshot-selector-content">
        <!-- Steam 搜索 -->
        <div class="steam-search-section">
          <a-input-search
            v-model="steamScreenshotSearchQuery"
            placeholder="搜索 Steam 游戏..."
            :loading="isSearchingSteamScreenshots"
            @search="searchSteamForScreenshots"
            @clear="handleScreenshotSearchClear"
            allow-clear
          >
            <template #prepend>
              <icon-cloud-download />
            </template>
          </a-input-search>

          <!-- Steam 游戏搜索结果 -->
          <div v-if="steamScreenshotSearchResults.length > 0 && !selectedSteamScreenshotGame" class="steam-search-results">
            <div class="steam-search-title">选择游戏</div>
            <div
              v-for="game in steamScreenshotSearchResults"
              :key="game.id"
              class="steam-search-result-item"
              @click="selectSteamScreenshotGame(game)"
            >
              <img :src="game.tinyImage" :alt="game.name" />
              <div class="steam-result-info">
                <div class="steam-result-name">{{ game.name }}</div>
                <div class="steam-result-meta">{{ game.releaseDate || '' }}</div>
              </div>
            </div>
          </div>

          <!-- Steam 截图选择 -->
          <div v-if="steamScreenshotsData" class="steam-screenshots-section">
            <div class="steam-game-info">
              <img :src="steamScreenshotsData.cover" :alt="steamScreenshotsData.name" />
              <span>{{ steamScreenshotsData.name }}</span>
              <a-button type="text" size="mini" @click="backToScreenshotGameSearch">返回</a-button>
            </div>

            <div v-if="steamScreenshotsData.usedFallbackAssets" class="steam-screenshot-hint">
              Steam 未返回截图，以下为可用商店素材
            </div>

            <div v-if="steamScreenshotsData.screenshots.length > 0" class="steam-screenshots-grid">
              <div
                v-for="(screenshot, index) in steamScreenshotsData.screenshots"
                :key="index"
                class="steam-screenshot-item"
                :class="{ 'steam-screenshot-selected': selectedSteamScreenshots.has(index) }"
                @click="toggleSteamScreenshot(index)"
              >
                <img :src="screenshot" />
                <div v-if="selectedSteamScreenshots.has(index)" class="steam-screenshot-check">
                  <icon-check />
                </div>
              </div>
            </div>

            <a-empty
              v-else
              description="未找到可用截图"
              class="steam-screenshots-empty"
            />

            <a-button
              v-if="selectedSteamScreenshots.size > 0"
              type="primary"
              long
              :loading="isDownloadingSteamScreenshots"
              @click="downloadSelectedSteamScreenshots"
            >
              下载选中的 {{ selectedSteamScreenshots.size }} 张截图
            </a-button>
          </div>
        </div>

        <a-divider />

        <!-- 本地上传 -->
        <a-upload
          :action="screenshotUploadAction"
          :data="screenshotUploadData"
          :headers="uploadHeaders"
          :show-file-list="false"
          accept="image/*"
          @success="handleScreenshotUploadSuccess"
          @error="handleScreenshotUploadError"
        >
          <a-button type="dashed" long>
            <template #icon>
              <icon-upload />
            </template>
            本地上传
          </a-button>
        </a-upload>

        <a-divider>或</a-divider>

        <!-- URL 下载 -->
        <div class="url-input-section">
          <a-input
            v-model="screenshotSearchUrl"
            placeholder="输入图片 URL..."
            @press-enter="loadScreenshotPreview"
          >
            <template #append>
              <a-button type="primary" @click="loadScreenshotPreview">
                加载
              </a-button>
            </template>
          </a-input>

          <!-- 预览区域 -->
          <div v-if="screenshotPreviewUrl" class="cover-preview-section">
            <img :src="screenshotPreviewUrl" class="cover-preview-img" />
          </div>
        </div>

        <!-- 操作按钮 -->
        <div class="cover-selector-actions">
          <a-button @click="showScreenshotSelector = false">取消</a-button>
          <a-button type="primary" :disabled="!screenshotPreviewUrl" :loading="isDownloadingScreenshot" @click="confirmScreenshotSelection">
            确定
          </a-button>
        </div>
      </div>
    </a-modal>
  </a-modal>
</template>

<script setup lang="ts">
import { ref, watch, computed } from 'vue'
import { useUiStore } from '@/stores/ui'
import { deleteAsset, uploadAsset } from '@/services/assets'
import { directoryService } from '@/services/directory.service'
import type { Game } from '@/services/types'
import gamesService from '@/services/games.service'
import FileBrowserModal from '@/components/FileBrowserModal.vue'
import {
  IconFolder,
  IconPlus,
  IconMinus,
  IconImage,
  IconDelete,
  IconUpload,
  IconCloudDownload,
  IconCheck,
} from '@arco-design/web-vue/es/icon'
import steamService from '@/services/steam.service'
import { seriesService } from '@/services/series.service'
import { platformService } from '@/services/platforms.service'
import type { Platform } from '@/services/types'
import type { Series } from '@/services/types'
import type { Developer } from '@/services/types'
import type { Publisher } from '@/services/types'

interface Props {
  visible: boolean
  game: Game | null
}

interface FilePathItem {
  id?: number
  path: string
  label: string
}

interface GameForm {
  title: string
  title_alt: string
  developer: string | number | null
  publisher: string | number | null
  release_date: string | undefined
  engine: string
  platform: (string | number)[]
  series: string | number | null
  summary: string
  cover_image: string
  banner_image: string
  screenshots: string[]
  file_paths: FilePathItem[]
}

const props = defineProps<Props>()

const emit = defineEmits<{
  'update:visible': [value: boolean]
  'success': []
}>()

const uiStore = useUiStore()
const formRef = ref()
const isSubmitting = ref(false)

// Form validation rules
const rules = {
  title: [{ required: true, message: '请输入游戏名称' }]
}

// Steam 搜索状态
const steamCoverSearchQuery = ref('')
const steamCoverSearchResults = ref<any[]>([])
const isSearchingSteamCover = ref(false)
const selectedSteamGame = ref<any>(null)
const steamCoverImages = ref<string[]>([])
const selectedCoverImage = ref('')

const steamScreenshotSearchQuery = ref('')
const steamScreenshotSearchResults = ref<any[]>([])
const selectedSteamScreenshotGame = ref<any>(null)
const steamScreenshotsData = ref<{
  name: string
  cover: string
  screenshots: string[]
  appId: string
  usedFallbackAssets: boolean
} | null>(null)
const selectedSteamScreenshots = ref<Set<number>>(new Set())
const isSearchingSteamScreenshots = ref(false)
const isDownloadingSteamScreenshots = ref(false)
const seriesOptions = ref<Series[]>([])
const platformOptions = ref<Platform[]>([])
const developerOptions = ref<Developer[]>([])
const publisherOptions = ref<Publisher[]>([])
const isSearchingSeries = ref(false)
const isSearchingDevelopers = ref(false)
const isSearchingPublishers = ref(false)

const showSummarySelector = ref(false)
const steamSummarySearchQuery = ref('')
const steamSummarySearchResults = ref<any[]>([])
const selectedSteamSummaryGame = ref<any>(null)
const steamSummaryPreview = ref('')
const isSearchingSteamSummary = ref(false)

const steamBannerSearchQuery = ref('')
const steamBannerSearchResults = ref<any[]>([])
const isSearchingSteamBanner = ref(false)
const selectedSteamBannerGame = ref<any>(null)
const steamBannerImages = ref<string[]>([])
const selectedBannerImage = ref('')

// Files to delete only after successful submit
const pendingDeleteAssets = ref<Array<{ type: 'cover' | 'banner' | 'screenshot'; path: string }>>([])

const filteredSeriesOptions = computed(() => {
  return [...seriesOptions.value].sort((a, b) => a.name.localeCompare(b.name, 'zh-Hans-CN'))
})

const handleSeriesSearch = async (query: string) => {
  if (!query) return
  isSearchingSeries.value = true
  try {
    const results = await seriesService.searchSeries(query)
    // Add results but keep current selection if it exists
    const current = seriesOptions.value.find(s => s.id === form.value.series)
    seriesOptions.value = results
    if (current && !results.find(s => s.id === current.id)) {
      seriesOptions.value.push(current)
    }
  } finally {
    isSearchingSeries.value = false
  }
}

const filteredDeveloperOptions = computed(() => {
  return [...developerOptions.value].sort((a, b) => a.name.localeCompare(b.name, 'zh-Hans-CN'))
})

const handleDeveloperSearch = async (query: string) => {
  if (!query) return
  isSearchingDevelopers.value = true
  try {
    const { developersService } = await import('@/services/developers.service')
    const results = await developersService.searchDevelopers(query)
    const current = developerOptions.value.find(d => d.id === form.value.developer)
    developerOptions.value = results
    if (current && !results.find(d => d.id === current.id)) {
      developerOptions.value.push(current)
    }
  } finally {
    isSearchingDevelopers.value = false
  }
}

const filteredPublisherOptions = computed(() => {
  return [...publisherOptions.value].sort((a, b) => a.name.localeCompare(b.name, 'zh-Hans-CN'))
})

const handlePublisherSearch = async (query: string) => {
  if (!query) return
  isSearchingPublishers.value = true
  try {
    const { publishersService } = await import('@/services/publishers.service')
    const results = await publishersService.searchPublishers(query)
    const current = publisherOptions.value.find(p => p.id === form.value.publisher)
    publisherOptions.value = results
    if (current && !results.find(p => p.id === current.id)) {
      publisherOptions.value.push(current)
    }
  } finally {
    isSearchingPublishers.value = false
  }
}

const form = ref<GameForm>({
  title: '',
  title_alt: '',
  developer: '',
  publisher: '',
  release_date: undefined,
  engine: '',
  platform: [],
  series: null,
  summary: '',
  cover_image: '',
  banner_image: '',
  screenshots: [],
  file_paths: [{ path: '', label: '' }]
})

// Cover image state
const showCoverSelector = ref(false)
const coverSearchUrl = ref('')
const coverPreviewUrl = ref('')
const isDownloadingCover = ref(false)
const uploadAction = computed(() => {
  const baseUrl = import.meta.env.VITE_API_BASE_URL || '/api'
  return `${baseUrl}/assets/cover`
})
const uploadData = computed(() => ({
  game_id: String(props.game?.id || ''),
  sort_order: '0',
}))

// Banner image state
const showBannerSelector = ref(false)
const bannerSearchUrl = ref('')
const bannerPreviewUrl = ref('')
const isDownloadingBanner = ref(false)
const bannerUploadAction = computed(() => {
  const baseUrl = import.meta.env.VITE_API_BASE_URL || '/api'
  return `${baseUrl}/assets/banner`
})
const bannerUploadData = computed(() => ({
  game_id: String(props.game?.id || ''),
  sort_order: '0',
}))

// Screenshot state
const showScreenshotSelector = ref(false)
const screenshotSearchUrl = ref('')
const screenshotPreviewUrl = ref('')
const isDownloadingScreenshot = ref(false)
const screenshotUploadAction = computed(() => {
  const baseUrl = import.meta.env.VITE_API_BASE_URL || '/api'
  return `${baseUrl}/assets/screenshot`
})
const screenshotUploadData = computed(() => ({
  game_id: String(props.game?.id || ''),
  sort_order: String(form.value.screenshots.length),
}))
const uploadHeaders = computed(() => {
  return {}
})

// File browser state
const showFileBrowser = ref(false)
const initialPath = ref('')
const currentFileIndex = ref(-1)

// Release date for date picker (Date object)
const releaseDate = ref<Date | null>(null)

const visible = computed({
  get: () => props.visible,
  set: (value) => emit('update:visible', value)
})

const createEmptyForm = (): GameForm => ({
  title: '',
  title_alt: '',
  developer: '',
  publisher: '',
  release_date: undefined,
  engine: '',
  platform: [],
  series: null,
  summary: '',
  cover_image: '',
  banner_image: '',
  screenshots: [],
  file_paths: [{ path: '', label: '' }],
})

const resetTransientState = () => {
  pendingDeleteAssets.value = []
  showFileBrowser.value = false
  showSummarySelector.value = false
  showCoverSelector.value = false
  showBannerSelector.value = false
  showScreenshotSelector.value = false

  steamSummarySearchQuery.value = ''
  steamSummarySearchResults.value = []
  selectedSteamSummaryGame.value = null
  steamSummaryPreview.value = ''

  steamCoverSearchQuery.value = ''
  steamCoverSearchResults.value = []
  selectedSteamGame.value = null
  steamCoverImages.value = []
  selectedCoverImage.value = ''
  coverSearchUrl.value = ''
  coverPreviewUrl.value = ''

  steamBannerSearchQuery.value = ''
  steamBannerSearchResults.value = []
  selectedSteamBannerGame.value = null
  steamBannerImages.value = []
  selectedBannerImage.value = ''
  bannerSearchUrl.value = ''
  bannerPreviewUrl.value = ''

  steamScreenshotSearchQuery.value = ''
  steamScreenshotSearchResults.value = []
  selectedSteamScreenshotGame.value = null
  steamScreenshotsData.value = null
  selectedSteamScreenshots.value = new Set()
  screenshotSearchUrl.value = ''
  screenshotPreviewUrl.value = ''
}

const hydrateFormFromGame = (game: Game | null) => {
  if (!game) {
    form.value = createEmptyForm()
    releaseDate.value = null
    return
  }

  let platformList: (string | number)[] = []
  if (game.platforms && game.platforms.length > 0) {
    platformList = game.platforms.map((name) => {
      const match = platformOptions.value.find((item) => item.name === name)
      return match ? match.id : name
    })
  } else if (game.platform) {
    const match = platformOptions.value.find((item) => item.name === game.platform)
    platformList = [match ? match.id : game.platform]
  }

  let filePaths: FilePathItem[] = [{ path: '', label: '' }]
  if (game.file_paths && game.file_paths.length > 0) {
    filePaths = game.file_paths.map((p: any) => {
      if (typeof p === 'string') {
        return { path: p, label: '' }
      }
      return { id: p.id, path: p.path || '', label: p.label || '' }
    })
  } else if (game.file_path) {
    filePaths = [{ path: game.file_path, label: '' }]
  }

  const seriesId: string | number | null = game.series && game.series.length > 0
    ? game.series[0].id
    : null
  const developerId: string | number | null = game.developers && game.developers.length > 0
    ? game.developers[0].id
    : null
  const publisherId: string | number | null = game.publishers && game.publishers.length > 0
    ? game.publishers[0].id
    : null

  form.value = {
    title: game.title || '',
    title_alt: game.title_alt || '',
    developer: developerId,
    publisher: publisherId,
    release_date: game.release_date || undefined,
    engine: game.engine || '',
    platform: platformList,
    series: seriesId,
    summary: game.summary || '',
    cover_image: game.cover_image || '',
    banner_image: game.banner_image || '',
    screenshots: game.screenshots || [],
    file_paths: filePaths,
  }

  if (game.release_date) {
    const parts = game.release_date.split('-')
    if (parts.length === 3) {
      releaseDate.value = new Date(parseInt(parts[0]), parseInt(parts[1]) - 1, parseInt(parts[2]))
    } else {
      releaseDate.value = new Date(game.release_date)
    }
  } else {
    releaseDate.value = null
  }
}

// Initialize options with popular items
const initializeOptions = async (currentGame?: Game | null) => {
  // Load series picks
  try {
    const popularSeries = await seriesService.getPopularSeries(50)
    seriesOptions.value = popularSeries
    if (currentGame?.series?.[0]) {
      const existing = popularSeries.find(s => s.id === currentGame.series![0].id)
      if (!existing) {
        seriesOptions.value.push(currentGame.series![0] as any)
      }
    }
  } catch (error) {
    console.error('Failed to load series:', error)
  }

  // Load developer picks
  try {
    const { developersService } = await import('@/services/developers.service')
    const popularDevelopers = await developersService.getPopularDevelopers(50)
    developerOptions.value = popularDevelopers
    if (currentGame?.developers?.[0]) {
      const existing = popularDevelopers.find(d => d.id === currentGame.developers![0].id)
      if (!existing) {
        developerOptions.value.push(currentGame.developers![0] as any)
      }
    }
  } catch (error) {
    console.error('Failed to load developers:', error)
  }

  // Load publisher picks
  try {
    const { publishersService } = await import('@/services/publishers.service')
    const popularPublishers = await publishersService.getPopularPublishers(50)
    publisherOptions.value = popularPublishers
    if (currentGame?.publishers?.[0]) {
      const existing = popularPublishers.find(p => p.id === currentGame.publishers![0].id)
      if (!existing) {
        publisherOptions.value.push(currentGame.publishers![0] as any)
      }
    }
  } catch (error) {
    console.error('Failed to load publishers:', error)
  }

  // Load platform options (usually small enough to load all)
  try {
    const allPlatforms = await platformService.getAllPlatforms()
    platformOptions.value = allPlatforms
  } catch (error) {
    console.error('Failed to load platforms:', error)
  }
}

// Initialize form and options
watch(() => props.game, async (game) => {
  await initializeOptions(game)
  hydrateFormFromGame(game)
}, { immediate: true })

// Reset state when modal opens
watch(visible, async (val) => {
  resetTransientState()
  if (val) {
    await initializeOptions(props.game)
    hydrateFormFromGame(props.game)
  }
})

// Handle date change from date picker
const handleDateChange = (value: Date | number | string | null) => {
  if (value) {
    // Convert to Date object if needed
    const dateObj = value instanceof Date ? value : new Date(value)
    // Set release_date as YYYY-MM-DD format (avoid timezone issues)
    const year = dateObj.getFullYear()
    const month = String(dateObj.getMonth() + 1).padStart(2, '0')
    const day = String(dateObj.getDate()).padStart(2, '0')
    form.value.release_date = `${year}-${month}-${day}`
  } else {
    form.value.release_date = undefined
  }
}

const queueAssetDeletion = (type: 'cover' | 'banner' | 'screenshot', path: string) => {
  if (!path) return
  pendingDeleteAssets.value.push({ type, path })
}

const uploadAssetFromUrl = async (
  url: string,
  assetType: 'cover' | 'banner' | 'screenshot',
  sortOrder = 0,
) => {
  if (!props.game?.id) {
    throw new Error('缺少游戏 ID')
  }

  const response = await fetch(url)
  if (!response.ok) {
    throw new Error(`下载远程图片失败: ${response.status}`)
  }

  const blob = await response.blob()
  const ext = blob.type.split('/')[1] || 'jpg'
  const file = new File([blob], `${assetType}-${Date.now()}.${ext}`, {
    type: blob.type || 'image/jpeg',
  })

  return uploadAsset(assetType, props.game.id, file, sortOrder)
}

// Cover image handlers
const handleCoverError = (e: Event) => {
  const img = e.target as HTMLImageElement
  img.src = 'data:image/svg+xml,%3Csvg xmlns="http://www.w3.org/2000/svg" viewBox="0 0 100 100"%3E%3Crect fill="%23333" width="100" height="100"/%3E%3Ctext x="50%25" y="50%25" text-anchor="middle" dy=".3em" fill="%23666" font-size="12"%3E加载失败%3C/text%3E%3C/svg%3E'
}

const loadCoverFromUrl = () => {
  if (coverSearchUrl.value.trim()) {
    coverPreviewUrl.value = coverSearchUrl.value.trim()
  }
}

const confirmCoverSelection = async () => {
  if (!coverPreviewUrl.value) return
  isDownloadingCover.value = true
  try {
    const path = await uploadAssetFromUrl(coverPreviewUrl.value, 'cover')
    if (form.value.cover_image) {
      queueAssetDeletion('cover', form.value.cover_image)
    }
    form.value.cover_image = path
    showCoverSelector.value = false
    coverSearchUrl.value = ''
    coverPreviewUrl.value = ''
    uiStore.addAlert('封面下载成功', 'success')
  } catch (error: any) {
    uiStore.addAlert('封面下载失败：' + error.message, 'error')
  } finally {
    isDownloadingCover.value = false
  }
}

const handleCoverUploadSuccess = (fileItem: any) => {
  // Arco Upload component returns fileItem with response
  const response = fileItem.response
  if (response?.success && response?.data?.path) {
    if (form.value.cover_image) {
      queueAssetDeletion('cover', form.value.cover_image)
    }
    form.value.cover_image = response.data.path
    showCoverSelector.value = false
    uiStore.addAlert('封面上传成功', 'success')
  } else {
    uiStore.addAlert('上传失败：' + (response?.error || '未知错误'), 'error')
  }
}

const handleCoverUploadError = () => {
  uiStore.addAlert('封面上传失败', 'error')
}

// Screenshot handlers
const loadScreenshotPreview = () => {
  if (screenshotSearchUrl.value.trim()) {
    screenshotPreviewUrl.value = screenshotSearchUrl.value.trim()
  }
}

const pickSteamSearchQuery = () => {
  const preferred = form.value.title_alt?.trim()
  if (preferred) return preferred
  return form.value.title?.trim() || ''
}

const stripHtmlToText = (html: string) => {
  if (!html.trim()) return ''

  if (typeof window !== 'undefined' && typeof DOMParser !== 'undefined') {
    const doc = new DOMParser().parseFromString(html, 'text/html')
    return (doc.body.textContent || '')
      .replace(/\u00a0/g, ' ')
      .replace(/\s+\n/g, '\n')
      .replace(/\n{3,}/g, '\n\n')
      .trim()
  }

  return html
    .replace(/<br\s*\/?>/gi, '\n')
    .replace(/<\/p>/gi, '\n\n')
    .replace(/<[^>]+>/g, ' ')
    .replace(/&nbsp;/gi, ' ')
    .replace(/\s+/g, ' ')
    .trim()
}

const handleSummarySearchClear = () => {
  steamSummarySearchQuery.value = ''
  steamSummarySearchResults.value = []
  selectedSteamSummaryGame.value = null
  steamSummaryPreview.value = ''
}

const searchSteamForSummary = async () => {
  if (!steamSummarySearchQuery.value.trim()) return

  isSearchingSteamSummary.value = true
  try {
    const results = await steamService.searchGames(steamSummarySearchQuery.value.trim())
    steamSummarySearchResults.value = results
    selectedSteamSummaryGame.value = null
    steamSummaryPreview.value = ''
  } catch (error: any) {
    uiStore.addAlert('搜索失败：' + (error?.message || '未知错误'), 'error')
  } finally {
    isSearchingSteamSummary.value = false
  }
}

watch(showSummarySelector, (isOpen) => {
  if (!isOpen) {
    return
  }

  const query = pickSteamSearchQuery()
  if (!query) {
    return
  }

  steamSummarySearchQuery.value = query
  searchSteamForSummary()
})

const selectSteamSummaryGame = async (game: any) => {
  selectedSteamSummaryGame.value = game
  steamSummaryPreview.value = ''
  isSearchingSteamSummary.value = true
  try {
    const details = await steamService.getGameDetails(game.id)
    steamSummaryPreview.value = stripHtmlToText(details.description || '')
  } catch (error: any) {
    uiStore.addAlert('获取简介失败：' + (error?.message || '未知错误'), 'error')
  } finally {
    isSearchingSteamSummary.value = false
  }
}

const backToSummarySearch = () => {
  selectedSteamSummaryGame.value = null
  steamSummaryPreview.value = ''
}

const confirmSummaryImport = () => {
  if (!steamSummaryPreview.value) {
    uiStore.addAlert('当前没有可导入的简介', 'warning')
    return
  }

  form.value.summary = steamSummaryPreview.value
  showSummarySelector.value = false
  uiStore.addAlert(
    `已导入简介：${selectedSteamSummaryGame.value?.name || 'Steam 游戏'}`,
    'success',
  )
}

const confirmScreenshotSelection = async () => {
  if (!screenshotPreviewUrl.value) return
  isDownloadingScreenshot.value = true
  try {
    const path = await uploadAssetFromUrl(screenshotPreviewUrl.value, 'screenshot', form.value.screenshots.length)
    form.value.screenshots.push(path)
    showScreenshotSelector.value = false
    screenshotSearchUrl.value = ''
    screenshotPreviewUrl.value = ''
    uiStore.addAlert('截图下载成功', 'success')
  } catch (error: any) {
    uiStore.addAlert('截图下载失败：' + error.message, 'error')
  } finally {
    isDownloadingScreenshot.value = false
  }
}

const handleScreenshotUploadSuccess = (fileItem: any) => {
  const response = fileItem.response
  if (response?.success && response?.data?.path) {
    form.value.screenshots.push(response.data.path)
    showScreenshotSelector.value = false
    uiStore.addAlert('截图上传成功', 'success')
  } else {
    uiStore.addAlert('上传失败：' + (response?.error || '未知错误'), 'error')
  }
}

const handleScreenshotUploadError = () => {
  uiStore.addAlert('截图上传失败', 'error')
}

const removeCover = () => {
  const coverUrl = form.value.cover_image
  if (!coverUrl) return
  queueAssetDeletion('cover', coverUrl)
  form.value.cover_image = ''
}

const removeBanner = () => {
  const bannerUrl = form.value.banner_image
  if (!bannerUrl) return
  queueAssetDeletion('banner', bannerUrl)
  form.value.banner_image = ''
}

const removeScreenshot = (index: number) => {
  const screenshotUrl = form.value.screenshots[index]
  if (!screenshotUrl) return
  queueAssetDeletion('screenshot', screenshotUrl)
  form.value.screenshots.splice(index, 1)
}

// File path management
const addFilePath = () => {
  form.value.file_paths.push({ path: '', label: '' })
}

const removeFilePath = (index: number) => {
  form.value.file_paths.splice(index, 1)
}

const openFileBrowser = async (index: number) => {
  currentFileIndex.value = index
  try {
    const defaultPath = await directoryService.getDefaultDirectory()
    initialPath.value = form.value.file_paths[index]?.path || defaultPath
    showFileBrowser.value = true
  } catch (error) {
    console.error('Failed to get default directory:', error)
  }
}

// Steam 封面搜索
const handleCoverSearchClear = () => {
  steamCoverSearchQuery.value = ''
  steamCoverSearchResults.value = []
  selectedSteamGame.value = null
  steamCoverImages.value = []
  selectedCoverImage.value = ''
}

const searchSteamForCover = async () => {
  if (!steamCoverSearchQuery.value.trim()) return

  isSearchingSteamCover.value = true
  try {
    const results = await steamService.searchGames(steamCoverSearchQuery.value.trim())
    steamCoverSearchResults.value = results
    selectedSteamGame.value = null
    steamCoverImages.value = []
    selectedCoverImage.value = ''
  } catch (error: any) {
    uiStore.addAlert('搜索失败：' + error.message, 'error')
  } finally {
    isSearchingSteamCover.value = false
  }
}

// Steam 横幅搜索
const handleBannerSearchClear = () => {
  steamBannerSearchQuery.value = ''
  steamBannerSearchResults.value = []
  selectedSteamBannerGame.value = null
  steamBannerImages.value = []
  selectedBannerImage.value = ''
}

const searchSteamForBanner = async () => {
  if (!steamBannerSearchQuery.value.trim()) return

  isSearchingSteamBanner.value = true
  try {
    const results = await steamService.searchGames(steamBannerSearchQuery.value.trim())
    steamBannerSearchResults.value = results
    selectedSteamBannerGame.value = null
    steamBannerImages.value = []
    selectedBannerImage.value = ''
  } catch (error: any) {
    uiStore.addAlert('搜索失败：' + error.message, 'error')
  } finally {
    isSearchingSteamBanner.value = false
  }
}

const selectSteamBannerGame = async (game: any) => {
  selectedSteamBannerGame.value = game
  isSearchingSteamBanner.value = true
  try {
    const details = await steamService.getGameDetails(game.id)
    // Steam 专用的横幅资产
    const libraryHero = details.libraryHero
    const background = details.background
    const headerImage = details.headerImage
    
    steamBannerImages.value = Array.from(
      new Set([libraryHero, background, headerImage].filter(Boolean) as string[])
    )
    
    // 如果没有这些，再回退到截图
    if (steamBannerImages.value.length < 2 && details.screenshots && details.screenshots.length > 0) {
      steamBannerImages.value = [...steamBannerImages.value, ...details.screenshots.slice(0, 5)]
    }
  } catch (error: any) {
    uiStore.addAlert('获取详情失败：' + error.message, 'error')
  } finally {
    isSearchingSteamBanner.value = false
  }
}

const backToBannerGameSearch = () => {
  selectedSteamBannerGame.value = null
  steamBannerImages.value = []
}

const downloadSelectedSteamBanner = async () => {
  if (!selectedBannerImage.value) return

  bannerSearchUrl.value = selectedBannerImage.value
  await loadBannerFromUrl()
}

const loadBannerFromUrl = async () => {
  if (!bannerSearchUrl.value.trim()) return

  isDownloadingBanner.value = true
  try {
    const path = await uploadAssetFromUrl(bannerSearchUrl.value, 'banner')
    if (form.value.banner_image) {
      queueAssetDeletion('banner', form.value.banner_image)
    }
    form.value.banner_image = path
    showBannerSelector.value = false
    bannerSearchUrl.value = ''
    bannerPreviewUrl.value = ''
    uiStore.addAlert('横幅下载成功', 'success')
  } catch (error: any) {
    uiStore.addAlert('下载失败：' + error.message, 'error')
  } finally {
    isDownloadingBanner.value = false
  }
}

const handleBannerUploadSuccess = (fileItem: any) => {
  const response = fileItem.response
  if (response?.success && response?.data?.path) {
    if (form.value.banner_image) {
      queueAssetDeletion('banner', form.value.banner_image)
    }
    form.value.banner_image = response.data.path
    showBannerSelector.value = false
    uiStore.addAlert('横幅上传成功', 'success')
  } else {
    uiStore.addAlert('上传失败：' + (response?.error || '未知错误'), 'error')
  }
}

const handleBannerUploadError = () => {
  uiStore.addAlert('横幅上传失败', 'error')
}

const confirmBannerSelection = async () => {
  if (bannerSearchUrl.value) {
    await loadBannerFromUrl()
  }
}

// 当封面选择器打开时，自动使用英文名搜索
watch(showCoverSelector, (isOpen) => {
  if (isOpen && form.value.title_alt && form.value.title_alt.trim()) {
    steamCoverSearchQuery.value = form.value.title_alt.trim()
    searchSteamForCover()
  }
})

// 当横幅选择器打开时，自动使用英文名搜索
watch(showBannerSelector, (isOpen) => {
  if (isOpen && form.value.title_alt && form.value.title_alt.trim()) {
    steamBannerSearchQuery.value = form.value.title_alt.trim()
    searchSteamForBanner()
  }
})

const selectSteamCoverGame = async (game: any) => {
  selectedSteamGame.value = game
  isSearchingSteamCover.value = true
  try {
    // 获取游戏的封面图 URL
    const coverUrl = `https://steamcdn-a.akamaihd.net/steam/apps/${game.id}/library_600x900_2x.jpg`
    // 暂时只显示一个封面，Steam 通常只有一个版本
    steamCoverImages.value = [coverUrl]
  } catch (error: any) {
    uiStore.addAlert('获取封面失败：' + error.message, 'error')
  } finally {
    isSearchingSteamCover.value = false
  }
}

const backToCoverGameSearch = () => {
  selectedSteamGame.value = null
  steamCoverImages.value = []
  selectedCoverImage.value = ''
}

const downloadSelectedSteamCover = async () => {
  if (!selectedCoverImage.value || !props.game?.id) return

  isSearchingSteamCover.value = true
  try {
    const path = await uploadAssetFromUrl(selectedCoverImage.value, 'cover')
    if (form.value.cover_image) {
      queueAssetDeletion('cover', form.value.cover_image)
    }
    form.value.cover_image = path
    showCoverSelector.value = false
    backToCoverGameSearch()
    steamCoverSearchQuery.value = ''
    steamCoverSearchResults.value = []
    uiStore.addAlert('封面下载成功', 'success')
  } catch (error: any) {
    uiStore.addAlert('下载失败：' + error.message, 'error')
  } finally {
    isSearchingSteamCover.value = false
  }
}

// Steam 截图搜索
const handleScreenshotSearchClear = () => {
  steamScreenshotSearchQuery.value = ''
  steamScreenshotSearchResults.value = []
  selectedSteamScreenshotGame.value = null
  steamScreenshotsData.value = null
  selectedSteamScreenshots.value.clear()
}

const searchSteamForScreenshots = async () => {
  if (!steamScreenshotSearchQuery.value.trim()) return

  isSearchingSteamScreenshots.value = true
  try {
    const results = await steamService.searchGames(steamScreenshotSearchQuery.value.trim())
    steamScreenshotSearchResults.value = results
    selectedSteamScreenshotGame.value = null
    steamScreenshotsData.value = null
    selectedSteamScreenshots.value.clear()
  } catch (error: any) {
    uiStore.addAlert('搜索失败：' + error.message, 'error')
  } finally {
    isSearchingSteamScreenshots.value = false
  }
}

// 当截图选择器打开时，自动使用英文名搜索
watch(showScreenshotSelector, (isOpen) => {
  if (isOpen && form.value.title_alt && form.value.title_alt.trim()) {
    steamScreenshotSearchQuery.value = form.value.title_alt.trim()
    searchSteamForScreenshots()
  }
})

const selectSteamScreenshotGame = async (game: any) => {
  selectedSteamScreenshotGame.value = game
  isSearchingSteamScreenshots.value = true
  try {
    const details = await steamService.getGameDetails(game.id)
    const screenshotCandidates = (details.screenshots || []).filter(Boolean)
    const fallbackAssets = [details.libraryHero, details.background, details.headerImage].filter(
      (value): value is string => !!value
    )
    const finalAssets =
      screenshotCandidates.length > 0
        ? screenshotCandidates
        : Array.from(new Set(fallbackAssets))

    steamScreenshotsData.value = {
      name: game.name,
      cover: game.tinyImage || '',
      screenshots: finalAssets,
      appId: game.id,
      usedFallbackAssets: screenshotCandidates.length === 0 && finalAssets.length > 0,
    }
    selectedSteamScreenshots.value.clear()
  } catch (error: any) {
    uiStore.addAlert('获取截图失败：' + error.message, 'error')
  } finally {
    isSearchingSteamScreenshots.value = false
  }
}

const backToScreenshotGameSearch = () => {
  selectedSteamScreenshotGame.value = null
  steamScreenshotsData.value = null
  selectedSteamScreenshots.value.clear()
}

const toggleSteamScreenshot = (index: number) => {
  if (selectedSteamScreenshots.value.has(index)) {
    selectedSteamScreenshots.value.delete(index)
  } else {
    selectedSteamScreenshots.value.add(index)
  }
}

const downloadSelectedSteamScreenshots = async () => {
  if (!steamScreenshotsData.value || !props.game?.id) return

  const indices = Array.from(selectedSteamScreenshots.value).sort((a, b) => a - b)
  if (indices.length === 0) return

  isDownloadingSteamScreenshots.value = true
  try {
    // 下载每张选中的截图
    for (let i = 0; i < indices.length; i++) {
      const index = indices[i]
      const screenshotUrl = steamScreenshotsData.value!.screenshots[index]
      const currentIndex = form.value.screenshots.length
      const path = await uploadAssetFromUrl(screenshotUrl, 'screenshot', currentIndex)
      form.value.screenshots.push(path)
    }

    showScreenshotSelector.value = false
    backToScreenshotGameSearch()
    steamScreenshotSearchQuery.value = ''
    steamScreenshotSearchResults.value = []
    uiStore.addAlert(`成功添加 ${indices.length} 张截图`, 'success')
  } catch (error: any) {
    uiStore.addAlert('下载失败：' + error.message, 'error')
  } finally {
    isDownloadingSteamScreenshots.value = false
  }
}

const handleFileSelect = (path: string) => {
  if (currentFileIndex.value >= 0) {
    form.value.file_paths[currentFileIndex.value].path = path
  }
}

const normalizeOptionId = (value: unknown): number | null => {
  if (typeof value === 'number' && !Number.isNaN(value)) return value
  return null
}

const slugifyMetadataName = (name: string) => {
  return name
    .trim()
    .toLowerCase()
    .replace(/[^a-z0-9]+/g, '-')
    .replace(/^-+|-+$/g, '')
}

const resolveSingleMetadataSelection = async <T extends { id: number; name: string }>(
  value: string | number | null | undefined,
  options: { value: T[] },
  createItem: (payload: { name: string; slug?: string }) => Promise<T>,
) => {
  if (value === null || value === undefined || value === '') {
    return [] as number[]
  }

  const normalizedId = normalizeOptionId(value)
  if (normalizedId !== null) {
    return [normalizedId]
  }

  if (typeof value !== 'string' || !value.trim()) {
    return [] as number[]
  }

  const name = value.trim()
  const existing = options.value.find((item) => item.name.trim().toLowerCase() === name.toLowerCase())
  if (existing) {
    return [existing.id]
  }

  const created = await createItem({
    name,
    slug: slugifyMetadataName(name),
  })
  options.value = [...options.value, created]
  return [created.id]
}

const resolvePlatformSelections = async (values: Array<string | number>) => {
  const ids: number[] = []

  for (const value of values) {
    const normalizedId = normalizeOptionId(value)
    if (normalizedId !== null) {
      ids.push(normalizedId)
      continue
    }

    if (typeof value !== 'string' || !value.trim()) {
      continue
    }

    const name = value.trim()
    const existing = platformOptions.value.find(
      (item) => item.name.trim().toLowerCase() === name.toLowerCase(),
    )
    if (existing) {
      ids.push(existing.id)
      continue
    }

    const created = await platformService.createPlatform({
      name,
      slug: slugifyMetadataName(name),
    })
    platformOptions.value = [...platformOptions.value, created]
    ids.push(created.id)
  }

  return ids
}

const handleCancel = () => {
  visible.value = false
  pendingDeleteAssets.value = []
}

const handleSubmit = async () => {
  if (!props.game) return

  // Validate form
  try {
    await formRef.value?.validate()
  } catch {
    return
  }

  isSubmitting.value = true

  try {
    const originalFileIds = new Set(
      (props.game.file_paths || [])
        .filter((item): item is { id?: number; path: string; label?: string } => typeof item !== 'string')
        .map((item) => item.id)
        .filter((id): id is number => typeof id === 'number'),
    )

    // Process series - handle both existing ID and new name
    let seriesIds: number[] | undefined = undefined
    if (form.value.series === null || form.value.series === undefined || form.value.series === '') {
      seriesIds = []
    } else {
      const { seriesService } = await import('@/services/series.service')
      const item = form.value.series

      if (typeof item === 'number') {
        // Existing series ID
        seriesIds = [item]
      } else if (typeof item === 'string' && item.trim()) {
        // New series name - backend will check for existence
        try {
          const seriesName = item.trim()
          const newSeries = await seriesService.createSeries({
            name: seriesName,
            slug: seriesName.toLowerCase().replace(/[^a-z0-9]+/g, '-')
          })
          seriesIds = [newSeries.id]
        } catch (error: any) {
          console.error('Failed to process series:', item, error)
          uiStore.addAlert(`系列 "${item}" 处理失败`, 'warning')
        }
      }
    }

    // Process developer - handle both existing ID and new name
    let developerIds: number[] | undefined = undefined
    try {
      const { developersService } = await import('@/services/developers.service')
      developerIds = await resolveSingleMetadataSelection(
        form.value.developer,
        developerOptions,
        (payload) => developersService.createDeveloper(payload),
      )
      form.value.developer = developerIds[0] ?? ''
    } catch (error: any) {
      console.error('Failed to process developer:', form.value.developer, error)
      uiStore.addAlert(`开发商 "${form.value.developer}" 处理失败`, 'warning')
    }

    // Process publisher - handle both existing ID and new name
    let publisherIds: number[] | undefined = undefined
    try {
      const { publishersService } = await import('@/services/publishers.service')
      publisherIds = await resolveSingleMetadataSelection(
        form.value.publisher,
        publisherOptions,
        (payload) => publishersService.createPublisher(payload),
      )
      form.value.publisher = publisherIds[0] ?? ''
    } catch (error: any) {
      console.error('Failed to process publisher:', form.value.publisher, error)
      uiStore.addAlert(`发行商 "${form.value.publisher}" 处理失败`, 'warning')
    }

    let platformIds: number[] = []
    try {
      platformIds = await resolvePlatformSelections(form.value.platform)
      form.value.platform = [...platformIds]
    } catch (error: any) {
      console.error('Failed to process platform:', form.value.platform, error)
      uiStore.addAlert('平台处理失败', 'warning')
    }

    // Submit game update with series, developers, publishers
    await gamesService.updateGame(String(props.game.id), {
      title: form.value.title,
      title_alt: form.value.title_alt,
      release_date: form.value.release_date || undefined,
      engine: form.value.engine,
      platforms: platformIds,
      series: seriesIds,
      developers: developerIds,
      publishers: publisherIds,
      summary: form.value.summary,
      cover_image: form.value.cover_image,
      banner_image: form.value.banner_image,
    })

    const keptFileIds = new Set<number>()
    const validFilePaths = form.value.file_paths.filter(item => item.path.trim())

    for (let index = 0; index < validFilePaths.length; index++) {
      const item = validFilePaths[index]
      const payload = {
        file_path: item.path.trim(),
        label: item.label.trim() || null,
        notes: null,
        sort_order: index,
      }

      if (item.id) {
        keptFileIds.add(item.id)
        await gamesService.updateGameFile(String(props.game.id), String(item.id), payload)
      } else {
        const created = await gamesService.createGameFile(String(props.game.id), payload)
        if (created.id) keptFileIds.add(created.id)
      }
    }

    for (const fileId of originalFileIds) {
      if (!keptFileIds.has(fileId)) {
        await gamesService.deleteGameFile(String(props.game.id), String(fileId))
      }
    }

    // After successful save, delete files that were marked for deletion
    if (pendingDeleteAssets.value.length > 0) {
      for (const item of pendingDeleteAssets.value) {
        try {
          await deleteAsset(props.game.id, item.type, item.path)
        } catch (e) {
          console.error('Failed to delete asset after save:', item.path, e)
        }
      }
      pendingDeleteAssets.value = []
    }

    // Refresh series options after successful save (load popular)
    try {
      const popularSeries = await seriesService.getPopularSeries(50)
      seriesOptions.value = popularSeries
    } catch (error) {
      console.error('Failed to refresh series:', error)
    }

    uiStore.addAlert('保存成功', 'success')
    emit('success')
    visible.value = false
  } catch (error: any) {
    uiStore.addAlert(error?.message || '保存失败', 'error')
  } finally {
    isSubmitting.value = false
  }
}
</script>

<style scoped>
.file-paths-container {
  display: flex;
  flex-direction: column;
  gap: 8px;
  width: 100%;
}

.summary-label {
  display: flex;
  align-items: center;
  justify-content: space-between;
  width: 100%;
  gap: 12px;
}

.file-path-item {
  display: flex;
  flex-direction: column;
  gap: 8px;
}

.file-path-row {
  display: flex;
  gap: 8px;
  align-items: center;
  width: 100%;
}

.file-path-input {
  flex: 5;
  min-width: 0;
}

.file-label-input {
  flex: 4;
  min-width: 0;
}

.file-path-row :deep(.arco-btn) {
  flex-shrink: 0;
}

.file-path-item :deep(.arco-input-prepend) {
  background: var(--color-fill-2);
  border-right: 1px solid var(--color-border-2);
  padding: 0 8px;
}

.path-index {
  font-size: 12px;
  color: var(--color-text-3);
  font-weight: 600;
}

.ml-2 {
  margin-left: 8px;
  white-space: nowrap;
}

.w-full {
  width: 100%;
}

/* Media Section Styles */
.media-section {
  display: flex;
  flex-direction: column;
  gap: 8px;
  width: 100%;
}

.media-frame {
  width: 100%;
  overflow: hidden;
  border-radius: 8px;
  border: 1px solid var(--color-border-2);
  background: var(--color-fill-2);
  display: flex;
  align-items: center;
  justify-content: center;
}

.media-preview {
  position: relative;
  width: 100%;
  height: 100%;
}

.media-empty-action {
  width: 100%;
  height: 100%;
  border: 1px dashed rgba(255, 255, 255, 0.1);
  border-radius: 8px;
  background: linear-gradient(180deg, rgba(255, 255, 255, 0.03) 0%, rgba(255, 255, 255, 0.015) 100%);
  color: var(--color-text-3);
  display: flex;
  flex-direction: column;
  align-items: center;
  justify-content: center;
  gap: 10px;
  cursor: pointer;
  transition: color 0.2s ease, background 0.2s ease, border-color 0.2s ease;
}

.media-empty-action:hover {
  color: rgb(var(--primary-6));
  background: rgba(var(--primary-6), 0.06);
  border-color: rgba(var(--primary-6), 0.45);
}

.media-empty-icon {
  font-size: 30px;
}

.media-empty-title {
  font-size: 14px;
  font-weight: 500;
}

.media-empty-subtitle {
  font-size: 12px;
  opacity: 0.75;
}

.media-overlay {
  position: absolute;
  top: 0;
  right: 0;
  bottom: 0;
  left: 0;
  display: flex;
  align-items: center;
  justify-content: center;
  background: rgba(8, 10, 16, 0.5);
  opacity: 0;
  transition: opacity 0.2s ease;
}

.media-overlay-actions {
  display: inline-flex;
  align-items: center;
  gap: 12px;
}

.media-preview:hover .media-overlay,
.screenshot-thumb:hover .screenshot-overlay {
  opacity: 1;
}

.media-frame--cover {
  aspect-ratio: 2 / 3;
}

.media-frame--banner {
  aspect-ratio: 16 / 9;
}

.media-actions {
  margin-top: 4px;
}

.media-action-button {
  width: 40px;
  height: 40px;
  display: inline-flex;
  align-items: center;
  justify-content: center;
  border: 0;
  border-radius: 999px;
  background: rgba(255, 255, 255, 0.92);
  color: var(--color-text-1);
  font-size: 18px;
  cursor: pointer;
  box-shadow: 0 10px 24px rgba(0, 0, 0, 0.25);
  transition: transform 0.2s ease, background 0.2s ease, color 0.2s ease;
}

.media-action-button:hover {
  background: #fff;
  transform: scale(1.06);
}

.media-action-button--danger {
  background: rgba(208, 58, 74, 0.92);
  color: #fff;
}

.media-action-button--danger:hover {
  background: rgba(224, 73, 89, 0.98);
}


/* Screenshots Grid */
.screenshots-grid {
  display: grid;
  grid-template-columns: repeat(auto-fill, minmax(100px, 1fr));
  gap: 8px;
}

.screenshot-thumb {
  aspect-ratio: 16/9;
  border-radius: 6px;
  overflow: hidden;
  background: var(--color-fill-2);
  cursor: pointer;
  position: relative;
  border: 1px solid var(--color-border-2);
}

.screenshot-thumb img {
  width: 100%;
  height: 100%;
  object-fit: cover;
}

.screenshot-add-tile {
  aspect-ratio: 16 / 9;
  border-radius: 6px;
  border: 1px dashed rgba(255, 255, 255, 0.1);
  background: linear-gradient(180deg, rgba(255, 255, 255, 0.03) 0%, rgba(255, 255, 255, 0.015) 100%);
  color: var(--color-text-3);
  display: flex;
  flex-direction: column;
  align-items: center;
  justify-content: center;
  gap: 10px;
  cursor: pointer;
  transition: border-color 0.2s ease, color 0.2s ease, background 0.2s ease;
}

.screenshot-add-tile:hover {
  border-color: rgb(var(--primary-6));
  color: rgb(var(--primary-6));
  background: rgba(var(--primary-6), 0.06);
}

.screenshot-overlay {
  position: absolute;
  top: 0;
  left: 0;
  right: 0;
  bottom: 0;
  display: flex;
  align-items: center;
  justify-content: center;
  background: rgba(8, 10, 16, 0.5);
  opacity: 0;
  transition: opacity 0.2s ease;
}

.cover-placeholder {
  display: flex;
  flex-direction: column;
  align-items: center;
  gap: 8px;
  color: var(--color-text-3);
  font-size: 12px;
}

.placeholder-icon {
  font-size: 32px;
}

.banner-thumb {
  aspect-ratio: 16 / 9 !important;
}

.banner-thumb img {
  object-fit: cover;
}

/* Cover Selector Modal */
.cover-selector-content {
  display: flex;
  flex-direction: column;
  gap: 16px;
}

.cover-preview-large {
  width: 100%;
  max-height: 400px;
  border-radius: 8px;
  overflow: hidden;
  background: var(--color-fill-2);
  display: flex;
  align-items: center;
  justify-content: center;
}

.cover-preview-large img {
  max-width: 100%;
  max-height: 400px;
  object-fit: contain;
}

/* Cover Preview Section */
.cover-preview-section {
  width: 100%;
  max-height: 300px;
  border-radius: 8px;
  overflow: hidden;
  background: var(--color-fill-2);
  display: flex;
  align-items: center;
  justify-content: center;
  margin-top: 12px;
}

.cover-preview-img {
  max-width: 100%;
  max-height: 300px;
  object-fit: contain;
}

.cover-selector-actions {
  display: flex;
  justify-content: flex-end;
  gap: 8px;
}

/* Steam 搜索区域 */
.steam-search-section {
  display: flex;
  flex-direction: column;
  gap: 12px;
}

.steam-search-title {
  font-size: 14px;
  font-weight: 500;
  color: var(--color-text-1);
  display: flex;
  align-items: center;
  justify-content: space-between;
}

.steam-search-results {
  display: flex;
  flex-direction: column;
  gap: 8px;
  max-height: 300px;
  overflow-y: auto;
  padding: 8px;
  background: var(--color-fill-1);
  border-radius: 6px;
}

.steam-search-result-item {
  display: flex;
  align-items: center;
  gap: 12px;
  padding: 8px;
  border-radius: 6px;
  cursor: pointer;
  transition: all 0.2s;
  border: 1px solid var(--color-border-2);
}

.steam-search-result-item:hover {
  background: var(--color-fill-2);
  border-color: rgb(var(--primary-6));
}

.steam-search-result-item img {
  width: 60px;
  height: 32px;
  object-fit: cover;
  border-radius: 4px;
}

.steam-result-info {
  flex: 1;
  min-width: 0;
}

.steam-result-name {
  font-size: 14px;
  font-weight: 500;
  color: var(--color-text-1);
  overflow: hidden;
  text-overflow: ellipsis;
  white-space: nowrap;
}

.steam-result-meta {
  font-size: 12px;
  color: var(--color-text-3);
}

/* Steam 图片选择 */
.steam-images-section {
  display: flex;
  flex-direction: column;
  gap: 12px;
}

.steam-summary-section {
  display: flex;
  flex-direction: column;
  gap: 12px;
}

.steam-summary-preview {
  max-height: 320px;
  overflow-y: auto;
  padding: 14px 16px;
  border-radius: 8px;
  background: var(--color-fill-1);
  border: 1px solid var(--color-border-2);
  color: var(--color-text-2);
  line-height: 1.75;
  white-space: pre-wrap;
}

.steam-images-grid {
  display: grid;
  grid-template-columns: repeat(auto-fill, minmax(150px, 1fr));
  gap: 12px;
  max-height: 300px;
  overflow-y: auto;
}

.steam-image-item {
  aspect-ratio: 2/3;
  border-radius: 6px;
  overflow: hidden;
  cursor: pointer;
  border: 2px solid var(--color-border-2);
  transition: all 0.2s;
}

.steam-image-item img {
  width: 100%;
  height: 100%;
  object-fit: cover;
}

.steam-image-item:hover {
  border-color: rgb(var(--primary-6));
}

.steam-image-selected {
  border-color: rgb(var(--primary-6));
  box-shadow: 0 0 0 2px rgba(var(--primary-6), 0.2);
}

/* Steam 截图选择 */
.steam-screenshots-section {
  display: flex;
  flex-direction: column;
  gap: 12px;
}

.steam-screenshot-hint {
  padding: 10px 12px;
  border-radius: 6px;
  background: rgba(var(--primary-6), 0.08);
  color: var(--color-text-2);
  font-size: 12px;
}

.steam-game-info {
  display: flex;
  align-items: center;
  gap: 8px;
  padding: 8px;
  background: var(--color-fill-2);
  border-radius: 6px;
}

.steam-game-info img {
  width: 40px;
  height: 21px;
  object-fit: cover;
  border-radius: 3px;
}

.steam-game-info span {
  font-size: 14px;
  font-weight: 500;
  color: var(--color-text-1);
}

.steam-screenshots-grid {
  display: grid;
  grid-template-columns: repeat(auto-fill, minmax(120px, 1fr));
  gap: 8px;
}

.steam-screenshots-empty {
  padding: 20px 0;
  border-radius: 8px;
  background: var(--color-fill-1);
}

.steam-screenshot-item {
  aspect-ratio: 16/9;
  border-radius: 6px;
  overflow: hidden;
  cursor: pointer;
  position: relative;
  border: 2px solid var(--color-border-2);
  transition: all 0.2s;
}

.steam-screenshot-item img {
  width: 100%;
  height: 100%;
  object-fit: cover;
}

.steam-screenshot-item:hover {
  border-color: rgb(var(--primary-6));
}

.steam-screenshot-selected {
  border-color: rgb(var(--primary-6));
  background: rgba(var(--primary-6), 0.1);
}

.steam-screenshot-check {
  position: absolute;
  top: 4px;
  right: 4px;
  width: 20px;
  height: 20px;
  background: rgb(var(--primary-6));
  border-radius: 50%;
  display: flex;
  align-items: center;
  justify-content: center;
  color: white;
}

/* 截图选择器内容 */
.screenshot-selector-content {
  display: flex;
  flex-direction: column;
  gap: 16px;
}
.banner-preview-modal-content img {
  max-width: 100%;
  max-height: 80vh;
  object-fit: contain;
}
</style>
