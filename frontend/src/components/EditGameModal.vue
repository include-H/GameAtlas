<template>
  <a-modal
    v-model:visible="visible"
    title="编辑游戏信息"
    :width="800"
    :footer="false"
    :align-center="false"
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
              v-model="form.developers"
              placeholder="选择开发商（可多选）"
              multiple
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
              v-model="form.publishers"
              placeholder="选择发行商（可多选）"
              multiple
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
          <a-form-item label="可见性">
            <a-radio-group v-model="form.visibility" type="button">
              <a-radio value="public">公开</a-radio>
              <a-radio value="private">私有</a-radio>
            </a-radio-group>
          </a-form-item>
        </a-col>
      </a-row>

      <a-row :gutter="16">
        <a-col :span="12">
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
        <a-col :span="12">
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
        </a-col>
      </a-row>

      <a-form-item>
        <template #label>
          <div class="summary-label">
            <span>标签</span>
            <a-button
              type="text"
              size="mini"
              html-type="button"
              :disabled="!props.game?.wiki_content"
              :loading="isPreparingWikiTagCandidates"
              @click="handleParseWikiTags"
            >
              从 Wiki 提取字段
            </a-button>
          </div>
        </template>
        <div v-if="tagGroups.length > 0" class="tag-group-grid">
          <div
            v-for="group in tagGroups"
            :key="group.id"
            class="tag-group-field"
          >
            <div class="tag-group-field__label">
              <span>{{ group.name }}</span>
            </div>
            <a-select
              class="tag-group-select"
              :model-value="tagSelectionsByGroup[group.id]"
              :multiple="group.allow_multiple"
              allow-clear
              allow-search
              allow-create
              :max-tag-count="group.allow_multiple ? 2 : 1"
              :placeholder="`选择${group.name}`"
              @change="handleTagSelectionChange(group.id, $event)"
            >
              <a-option
                v-for="pendingTag in pendingTagOptionsByGroup[group.id] || []"
                :key="pendingTag.value"
                :value="pendingTag.value"
                :label="pendingTag.label"
              >
                {{ pendingTag.label }}
              </a-option>
              <a-option
                v-for="tag in tagOptionsByGroup[group.id] || []"
                :key="tag.id"
                :value="tag.id"
                :label="tag.name"
              >
                {{ tag.name }}
              </a-option>
            </a-select>
          </div>
        </div>
        <div v-else class="tag-group-empty">
          暂无可用标签。重启后端完成 migration 后，这里会显示可选标签组。
        </div>
      </a-form-item>

      <a-form-item>
        <template #label>
          <div class="summary-label">
            <span>简介</span>
            <a-button
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
              <a-button type="secondary" html-type="button" @click="openFileBrowser(index)">
                <template #icon>
                  <icon-folder />
                </template>
                浏览
              </a-button>
                <a-button
                  type="text"
                  status="danger"
                  html-type="button"
                  @click="removeFilePath(index)"
                >
                <icon-minus />
              </a-button>
            </div>
          </div>
          
          <a-button
            type="secondary"
            long
            html-type="button"
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
            <div class="media-section media-section--cover">
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
                      <a-button
                        class="media-action-button"
                        type="secondary"
                        shape="circle"
                        size="small"
                        html-type="button"
                        @click.stop="showCoverSelector = true"
                      >
                        <icon-settings />
                      </a-button>
                      <a-button
                        class="media-action-button media-action-button--danger"
                        type="secondary"
                        status="danger"
                        shape="circle"
                        size="small"
                        html-type="button"
                        @click.stop="removeCover"
                      >
                        <icon-delete />
                      </a-button>
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
                      <a-button
                        class="media-action-button"
                        type="secondary"
                        shape="circle"
                        size="small"
                        html-type="button"
                        @click.stop="showBannerSelector = true"
                      >
                        <icon-settings />
                      </a-button>
                      <a-button
                        class="media-action-button media-action-button--danger"
                        type="secondary"
                        status="danger"
                        shape="circle"
                        size="small"
                        html-type="button"
                        @click.stop="removeBanner"
                      >
                        <icon-delete />
                      </a-button>
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

	          <a-form-item label="预告片" class="media-subitem">
	            <div class="media-section">
	              <div class="media-frame media-frame--video">
	                <div v-if="primaryPreviewVideo" class="media-preview">
	                  <video
	                    class="media-video"
	                    controls
	                    playsinline
	                    preload="metadata"
	                  >
	                    <source
	                      v-for="src in previewVideoSources"
	                      :key="src"
	                      :src="src"
	                    />
	                  </video>
	                  <div class="media-overlay media-overlay--top-right">
	                    <div class="media-overlay-actions">
	                      <a-button
                          class="media-action-button"
                          type="secondary"
                          shape="circle"
                          size="small"
                          html-type="button"
                          @click.stop="openVideoSelector"
                        >
	                        <icon-settings />
	                      </a-button>
	                    </div>
	                  </div>
	                </div>
	                <div
	                  v-else
	                  class="media-empty-action"
	                  role="button"
	                  tabindex="0"
	                  @click="openVideoSelector"
	                >
	                  <icon-upload class="media-empty-icon" />
	                  <span class="media-empty-title">未设置预告片</span>
	                  <span class="media-empty-subtitle">点击上传本地视频</span>
	                </div>
	              </div>
	            </div>
	          </a-form-item>
	        </a-col>

        <!-- 截图 -->
        <a-col :span="8">
          <a-form-item label="截图">
            <div class="media-section media-section--cover">
              <div class="media-frame media-frame--cover screenshots-frame">
                <div v-if="form.screenshots.length === 0" class="media-empty-action">
                  <div
                    class="media-empty-action media-empty-action--inner"
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
                  <div class="screenshots-grid-wrapper">
                    <div class="screenshots-grid">
                      <div
                        v-for="screenshot in form.screenshots"
                        :key="screenshot.asset_uid || screenshot.client_key"
                        class="screenshot-thumb"
                        :class="{ 'is-dragging': draggedScreenshotKey === screenshot.client_key, 'is-drop-target': dragOverScreenshotKey === screenshot.client_key }"
                        draggable="true"
                        @dragstart="handleScreenshotDragStart(screenshot.client_key)"
                        @dragenter.prevent="handleScreenshotDragEnter(screenshot.client_key)"
                        @dragover.prevent
                        @drop.prevent="handleScreenshotDrop(screenshot.client_key)"
                        @dragend="handleScreenshotDragEnd"
                      >
                        <a-image
                          :src="screenshot.path"
                          width="100%"
                          height="100%"
                          fit="cover"
                          hide-footer
                        />
                        <div class="screenshot-overlay">
	                          <a-button
	                            class="media-action-button media-action-button--danger"
	                            type="secondary"
	                            status="danger"
	                            shape="circle"
	                            size="small"
	                            html-type="button"
	                            @click.stop="removeScreenshot(screenshot.client_key)"
	                          >
                            <icon-delete />
                          </a-button>
                        </div>
                      </div>
                      <div
                        class="screenshot-add-tile"
                        role="button"
                        tabindex="0"
                        @click="showScreenshotSelector = true"
                      >
                        <span class="screenshot-add-tile__label">添加截图</span>
                      </div>
                    </div>
                  </div>
                </a-image-preview-group>
              </div>
            </div>
          </a-form-item>
        </a-col>
	      </a-row>

		      <a-form-item>
	        <a-space style="justify-content: flex-end; width: 100%">
          <a-button type="secondary" html-type="button" @click="handleCancel">取消</a-button>
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
        <steam-search-panel
          v-model:query="steamSummarySearchQuery"
          placeholder="搜索 Steam 游戏..."
          :loading="isSearchingSteamSummary"
          :results="steamSummarySearchResults"
          :selected-game="selectedSteamSummaryGame"
          @search="searchSteamForSummary"
          @clear="handleSummarySearchClear"
          @select="selectSteamSummaryGame"
        >
          <div v-if="selectedSteamSummaryGame" class="steam-summary-section">
            <div class="steam-search-title">
              {{ selectedSteamSummaryGame.name }} 的简介
              <a-button type="text" size="mini" html-type="button" @click="backToSummarySearch">返回</a-button>
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
              html-type="button"
              @click="confirmSummaryImport"
            >
              导入这段简介
            </a-button>
          </div>
        </steam-search-panel>
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
	        <a-divider>从 Steam 获取</a-divider>
	        <!-- Steam 搜索 -->
	        <steam-search-panel
          v-model:query="steamCoverSearchQuery"
          placeholder="搜索 Steam 游戏..."
          :loading="isSearchingSteamCover"
          :results="steamCoverSearchResults"
          :selected-game="selectedSteamGame"
          @search="searchSteamForCover"
          @clear="handleCoverSearchClear"
          @select="selectSteamCoverGame"
        >
          <!-- Steam 封面图片选择 -->
          <div v-if="selectedSteamGame && steamCoverImages.length > 0" class="steam-images-section">
            <div class="steam-search-title">
              {{ selectedSteamGame.name }} 的封面
              <a-button type="text" size="mini" html-type="button" @click="backToCoverGameSearch">返回</a-button>
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
              html-type="button"
              @click="downloadSelectedSteamCover"
            >
              下载选中的封面
            </a-button>
          </div>
        </steam-search-panel>

	        <a-divider>本地上传</a-divider>

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
          <a-button type="secondary" long html-type="button">
            <template #icon>
              <icon-upload />
            </template>
            点击上传本地图片
          </a-button>
        </a-upload>

        <a-divider>或从 URL 加载</a-divider>

        <!-- URL 加载 -->
        <div class="url-input-row">
          <a-input
            v-model="coverSearchUrl"
            class="url-input-row__field"
            placeholder="输入图片 URL..."
            @press-enter="loadCoverFromUrl"
          />
          <a-button class="url-input-row__action" type="secondary" html-type="button" @click="loadCoverFromUrl">
            加载
          </a-button>
        </div>
        <div v-if="coverPreviewUrl" class="cover-preview-large">
          <img :src="coverPreviewUrl" @error="handleCoverError" />
        </div>
        <div class="cover-selector-actions">
          <a-button type="secondary" html-type="button" @click="showCoverSelector = false">取消</a-button>
          <a-button type="primary" html-type="button" :disabled="!coverPreviewUrl" :loading="isDownloadingCover" @click="confirmCoverSelection">
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
	        <a-divider>从 Steam 获取</a-divider>
	        <!-- Steam 搜索 -->
	        <steam-search-panel
          v-model:query="steamBannerSearchQuery"
          placeholder="搜索 Steam 游戏..."
          :loading="isSearchingSteamBanner"
          :results="steamBannerSearchResults"
          :selected-game="selectedSteamBannerGame"
          @search="searchSteamForBanner"
          @clear="handleBannerSearchClear"
          @select="selectSteamBannerGame"
        >
          <!-- Steam 横幅图片选择 -->
          <div v-if="selectedSteamBannerGame && steamBannerImages.length > 0" class="steam-images-section">
            <div class="steam-search-title">
              {{ selectedSteamBannerGame.name }} 的横幅
              <a-button type="text" size="mini" html-type="button" @click="backToBannerGameSearch">返回</a-button>
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
              html-type="button"
              @click="downloadSelectedSteamBanner"
            >
              下载选中的横幅
            </a-button>
          </div>
        </steam-search-panel>

	        <a-divider>本地上传</a-divider>

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
          <a-button type="secondary" long html-type="button">
            <template #icon>
              <icon-upload />
            </template>
            点击上传本地图片
          </a-button>
        </a-upload>

        <a-divider>或从 URL 加载</a-divider>

        <!-- URL 加载 -->
        <div class="url-input-row">
          <a-input
            v-model="bannerSearchUrl"
            class="url-input-row__field"
            placeholder="输入图片 URL..."
            @press-enter="loadBannerFromUrl"
          />
          <a-button class="url-input-row__action" type="secondary" html-type="button" @click="loadBannerFromUrl">
            加载
          </a-button>
        </div>
        <div v-if="bannerPreviewUrl" class="cover-preview-large">
          <img :src="bannerPreviewUrl" @error="handleCoverError" />
        </div>
        <div class="cover-selector-actions">
          <a-button type="secondary" html-type="button" @click="showBannerSelector = false">取消</a-button>
          <a-button type="primary" html-type="button" :disabled="!bannerPreviewUrl" :loading="isDownloadingBanner" @click="confirmBannerSelection">
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
	        <a-divider>从 Steam 获取</a-divider>
	        <!-- Steam 搜索 -->
	        <steam-search-panel
          v-model:query="steamScreenshotSearchQuery"
          placeholder="搜索 Steam 游戏..."
          :loading="isSearchingSteamScreenshots"
          :results="steamScreenshotSearchResults"
          :selected-game="selectedSteamScreenshotGame"
          @search="searchSteamForScreenshots"
          @clear="handleScreenshotSearchClear"
          @select="selectSteamScreenshotGame"
        >
          <!-- Steam 截图选择 -->
          <div v-if="steamScreenshotsData" class="steam-screenshots-section">
            <div class="steam-game-info">
              <img :src="steamScreenshotsData.cover" :alt="steamScreenshotsData.name" />
              <span>{{ steamScreenshotsData.name }}</span>
              <a-button type="text" size="mini" html-type="button" @click="backToScreenshotGameSearch">返回</a-button>
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
              html-type="button"
              @click="downloadSelectedSteamScreenshots"
            >
              下载选中的 {{ selectedSteamScreenshots.size }} 张截图
            </a-button>
          </div>
        </steam-search-panel>

	        <a-divider>本地上传</a-divider>

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
          <a-button type="secondary" long html-type="button">
            <template #icon>
              <icon-upload />
            </template>
            本地上传
          </a-button>
        </a-upload>

	        <a-divider>或从 URL 加载</a-divider>

        <!-- URL 下载 -->
        <div class="url-input-section">
          <div class="url-input-row">
            <a-input
              v-model="screenshotSearchUrl"
              class="url-input-row__field"
              placeholder="输入图片 URL..."
              @press-enter="loadScreenshotPreview"
            />
            <a-button class="url-input-row__action" type="secondary" html-type="button" @click="loadScreenshotPreview">
              加载
            </a-button>
          </div>

          <!-- 预览区域 -->
          <div v-if="screenshotPreviewUrl" class="cover-preview-section">
            <img :src="screenshotPreviewUrl" class="cover-preview-img" />
          </div>
        </div>

        <!-- 操作按钮 -->
        <div class="cover-selector-actions">
          <a-button type="secondary" html-type="button" @click="showScreenshotSelector = false">取消</a-button>
          <a-button type="primary" html-type="button" :disabled="!screenshotPreviewUrl" :loading="isDownloadingScreenshot" @click="confirmScreenshotSelection">
            确定
          </a-button>
	        </div>
	      </div>
	    </a-modal>

	    <a-modal
	      v-model:visible="showVideoSelector"
	      title="设置预告片"
	      :width="720"
	      :footer="false"
	    >
	      <div class="cover-selector-content">
	        <input
	          ref="videoFileInput"
	          type="file"
	          accept="video/mp4,video/webm"
	          class="hidden-file-input"
	          @change="handleVideoFileChange"
	        />
	        <a-button type="secondary" long html-type="button" :loading="isUploadingVideo" @click="openVideoFilePicker">
	          <template #icon>
	            <icon-upload />
	          </template>
	          {{ isUploadingVideo ? '上传中...' : '上传 MP4 / WebM' }}
	        </a-button>
	        <div v-if="isUploadingVideo || videoUploadProgress > 0" class="video-upload-progress">
	          <div class="video-upload-progress__meta">
	            <span>{{ videoUploadFileName || '预告片上传中' }}</span>
	            <span>{{ videoUploadProgress }}%</span>
	          </div>
	          <a-progress :percent="videoUploadProgress" :show-text="false" size="small" />
	        </div>
	        <div v-if="form.preview_videos.length > 0" class="video-library-card">
	          <div class="video-library-card__header">
	            <span>当前预告片</span>
	            <span>{{ form.preview_videos.length }} 个</span>
	          </div>
	          <div class="video-library-list">
	            <div
	              v-for="(video, index) in form.preview_videos"
	              :key="video.asset_uid || video.path"
	              class="video-library-item"
	              :class="{ 'is-primary': form.primary_preview_video_uid === video.asset_uid }"
	            >
	              <div
                  class="video-library-item__preview"
                  role="button"
                  tabindex="0"
                  @click="setPrimaryPreviewVideo(video.asset_uid)"
                  @keydown.enter.prevent="setPrimaryPreviewVideo(video.asset_uid)"
                  @keydown.space.prevent="setPrimaryPreviewVideo(video.asset_uid)"
                >
	                <div class="video-library-item__thumb">
	                  <img
	                    v-if="form.banner_image || form.cover_image"
	                    :src="form.banner_image || form.cover_image || ''"
	                    :alt="`预告片 ${index + 1}`"
	                  />
	                  <div v-else class="video-library-item__thumb-placeholder">
	                    <icon-video-camera />
	                  </div>
	                </div>
	                <div class="video-library-item__info">
	                  <div class="video-library-item__meta-row">
	                    <span class="video-library-item__title">预告片 {{ index + 1 }}</span>
	                    <a-tag v-if="form.primary_preview_video_uid === video.asset_uid" size="small" color="arcoblue">主预告</a-tag>
	                    <span class="video-library-item__path">{{ video.asset_uid || video.path }}</span>
	                  </div>
	                </div>
	              </div>
	              <div class="video-library-item__actions">
	                <a-button
	                  size="mini"
	                  type="text"
	                  html-type="button"
	                  :disabled="index === 0"
	                  @click="reorderEditableVideos(video.asset_uid || video.path, -1)"
	                >
	                  上移
	                </a-button>
	                <a-button
	                  size="mini"
	                  type="text"
	                  html-type="button"
	                  :disabled="index === form.preview_videos.length - 1"
	                  @click="reorderEditableVideos(video.asset_uid || video.path, 1)"
	                >
	                  下移
	                </a-button>
	                <a-button
	                  v-if="form.primary_preview_video_uid !== video.asset_uid"
	                  size="mini"
	                  type="text"
	                  html-type="button"
	                  @click="setPrimaryPreviewVideo(video.asset_uid)"
	                >
	                  设为主预告
	                </a-button>
	                <a-button size="mini" type="text" status="danger" html-type="button" @click="removePreviewVideo(video.asset_uid)">
	                  删除
	                </a-button>
	              </div>
	            </div>
	          </div>
	        </div>
	        <a-empty
	          v-else
	          description="还没有添加预告片，可上传本地视频"
	          class="video-library-empty"
	        />
	        <div class="cover-selector-actions">
	          <a-button type="secondary" html-type="button" @click="showVideoSelector = false">完成</a-button>
	        </div>
	      </div>
	    </a-modal>

      <a-modal
        v-model:visible="wikiTagPickerVisible"
        title="从 Wiki 提取字段"
        :width="760"
        :footer="false"
      >
        <div v-if="wikiTagCandidates.length > 0" class="wiki-tag-picker">
          <div class="wiki-tag-picker__hint">
            从当前 Wiki 中提取到了这些词条。你可以给每一项选择归类，也可以忽略。
          </div>

          <div class="wiki-tag-picker__list">
            <div
              v-for="item in wikiTagCandidates"
              :key="item.key"
              class="wiki-tag-picker__item"
            >
              <div class="wiki-tag-picker__meta">
                <div class="wiki-tag-picker__value">{{ item.value }}</div>
                <div class="wiki-tag-picker__source">来源：{{ item.sourceLabel }}</div>
              </div>
              <a-select
                class="wiki-tag-picker__select"
                :model-value="item.groupKey"
                @change="handleWikiTagCandidateGroupChange(item.key, $event)"
              >
                <a-option value="ignore">忽略</a-option>
                <a-option value="genre">题材</a-option>
                <a-option value="subgenre">子类型</a-option>
                <a-option value="perspective">视角</a-option>
                <a-option value="theme">内容属性</a-option>
              </a-select>
            </div>
          </div>

          <div class="cover-selector-actions">
            <a-button type="secondary" html-type="button" @click="wikiTagPickerVisible = false">取消</a-button>
            <a-button
              type="primary"
              html-type="button"
              :loading="isApplyingWikiTags"
              @click="applySelectedWikiTags"
            >
              应用到标签
            </a-button>
          </div>
        </div>
        <a-empty v-else description="没有识别到可提取的字段" />
      </a-modal>
	  </a-modal>
</template>

<script setup lang="ts">
import { ref, watch, computed } from 'vue'
import { useUiStore } from '@/stores/ui'
import { deleteAsset, reorderScreenshots, reorderVideos, uploadAsset, type UploadedAssetResult } from '@/services/assets'
import { directoryService } from '@/services/directory.service'
import type { Game } from '@/services/types'
import gamesService from '@/services/games.service'
import FileBrowserModal from '@/components/FileBrowserModal.vue'
import SteamSearchPanel from '@/components/SteamSearchPanel.vue'
import {
  IconFolder,
  IconPlus,
  IconMinus,
  IconImage,
  IconDelete,
  IconUpload,
  IconVideoCamera,
  IconSettings,
} from '@arco-design/web-vue/es/icon'
import steamService, { proxySteamAssetUrl } from '@/services/steam.service'
import { seriesService } from '@/services/series.service'
import platformService from '@/services/platforms.service'
import tagsService from '@/services/tags.service'
import { resolveAssetCandidates } from '@/utils/asset-url'
import { useSteamPicker } from '@/composables/useSteamPicker'
import {
  normalizeOptionId,
  resolveCreatableSelections,
  searchCreatableOptions,
  sortCreatableOptionsByName,
} from '@/utils/creatable-select'
import { extractWikiTagCandidates, type WikiTagGroupKey } from '@/utils/wiki-tag-parser'
import type { Developer, Platform, Publisher, ScreenshotItem, Series, SteamGameDetails, Tag, TagGroup, VideoAssetItem } from '@/services/types'

interface Props {
  visible: boolean
  game: Game | null
}

interface FilePathItem {
  id?: number
  path: string
  label: string
}

interface EditableScreenshot {
  id?: number
  asset_uid?: string
  path: string
  sort_order?: number
  client_key: string
}

interface EditableVideo {
  id?: number
  asset_uid?: string
  path: string
  sort_order?: number
}

interface WikiTagCandidateSelection {
  key: string
  value: string
  sourceLabel: string
  groupKey: WikiTagGroupKey | 'ignore'
}

interface GameForm {
  title: string
  title_alt: string
  visibility: 'public' | 'private'
  developers: Array<string | number>
  publishers: Array<string | number>
  release_date: string | undefined
  engine: string
  platform: (string | number)[]
  series: string | number | null
  tag_ids: Array<string | number>
  summary: string
  cover_image: string
  banner_image: string
  preview_videos: EditableVideo[]
  primary_preview_video_uid: string
  screenshots: EditableScreenshot[]
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
const steamCoverImages = ref<string[]>([])
const selectedCoverImage = ref('')

const steamScreenshotsData = ref<{
  name: string
  cover: string
  screenshots: string[]
  appId: string
  usedFallbackAssets: boolean
} | null>(null)
const selectedSteamScreenshots = ref<Set<number>>(new Set())
const isDownloadingSteamScreenshots = ref(false)
const seriesOptions = ref<Series[]>([])
const platformOptions = ref<Platform[]>([])
const tagGroups = ref<TagGroup[]>([])
const tagOptions = ref<Tag[]>([])
const isPreparingWikiTagCandidates = ref(false)
const isApplyingWikiTags = ref(false)
const wikiTagPickerVisible = ref(false)
const wikiTagCandidates = ref<WikiTagCandidateSelection[]>([])
const developerOptions = ref<Developer[]>([])
const publisherOptions = ref<Publisher[]>([])
const isSearchingSeries = ref(false)
const isSearchingDevelopers = ref(false)
const isSearchingPublishers = ref(false)

const showSummarySelector = ref(false)
const steamSummaryPreview = ref('')
const steamSummaryDetails = ref<SteamGameDetails | null>(null)

const steamBannerImages = ref<string[]>([])
const selectedBannerImage = ref('')
const pendingTagDraftsByGroup = ref<Record<number, string[]>>({})

const summarySteamPicker = useSteamPicker<SteamGameDetails>({
  onSelect: async (game) => {
    const details = await steamService.getGameDetails(game.id)
    steamSummaryDetails.value = details
    steamSummaryPreview.value = stripHtmlToText(details.description || '')
    return details
  },
  onError: (message) => {
    uiStore.addAlert('Steam 简介处理失败：' + message, 'error')
  },
})

const coverSteamPicker = useSteamPicker<string[]>({
  onSelect: async (game) => {
    const coverUrl = proxySteamAssetUrl(`https://steamcdn-a.akamaihd.net/steam/apps/${game.id}/library_600x900_2x.jpg`)
    steamCoverImages.value = [coverUrl]
    selectedCoverImage.value = ''
    return [coverUrl]
  },
  onError: (message) => {
    uiStore.addAlert('Steam 封面处理失败：' + message, 'error')
  },
})

const bannerSteamPicker = useSteamPicker<string[]>({
  onSelect: async (game) => {
    const details = await steamService.getGameDetails(game.id)
    const libraryHero = details.libraryHero
    const background = details.background
    const headerImage = details.headerImage
    const images = Array.from(new Set([libraryHero, background, headerImage].filter(Boolean) as string[]))
    const finalImages = images.length < 2 && details.screenshots && details.screenshots.length > 0
      ? [...images, ...details.screenshots.slice(0, 5)]
      : images
    steamBannerImages.value = finalImages
    selectedBannerImage.value = ''
    return finalImages
  },
  onError: (message) => {
    uiStore.addAlert('Steam 横幅处理失败：' + message, 'error')
  },
})

const screenshotSteamPicker = useSteamPicker<NonNullable<typeof steamScreenshotsData.value>>({
  onSelect: async (game) => {
    const details = await steamService.getGameDetails(game.id)
    const screenshotCandidates = (details.screenshots || []).filter(Boolean)
    const fallbackAssets = [details.libraryHero, details.background, details.headerImage].filter(
      (value): value is string => !!value,
    )
    const finalAssets =
      screenshotCandidates.length > 0
        ? screenshotCandidates
        : Array.from(new Set(fallbackAssets))

    const data = {
      name: game.name,
      cover: game.tinyImage || '',
      screenshots: finalAssets,
      appId: game.id,
      usedFallbackAssets: screenshotCandidates.length === 0 && finalAssets.length > 0,
    }
    steamScreenshotsData.value = data
    selectedSteamScreenshots.value.clear()
    return data
  },
  onError: (message) => {
    uiStore.addAlert('Steam 截图处理失败：' + message, 'error')
  },
})

const steamSummarySearchQuery = summarySteamPicker.query
const steamSummarySearchResults = summarySteamPicker.results
const selectedSteamSummaryGame = summarySteamPicker.selectedGame
const isSearchingSteamSummary = summarySteamPicker.isSearching

const steamCoverSearchQuery = coverSteamPicker.query
const steamCoverSearchResults = coverSteamPicker.results
const selectedSteamGame = coverSteamPicker.selectedGame
const isSearchingSteamCover = coverSteamPicker.isSearching

const steamBannerSearchQuery = bannerSteamPicker.query
const steamBannerSearchResults = bannerSteamPicker.results
const selectedSteamBannerGame = bannerSteamPicker.selectedGame
const isSearchingSteamBanner = bannerSteamPicker.isSearching

const steamScreenshotSearchQuery = screenshotSteamPicker.query
const steamScreenshotSearchResults = screenshotSteamPicker.results
const selectedSteamScreenshotGame = screenshotSteamPicker.selectedGame
const isSearchingSteamScreenshots = screenshotSteamPicker.isSearching

const tagGroupIdByTagId = computed(() => {
  return new Map(tagOptions.value.map((tag) => [tag.id, tag.group_id]))
})

const tagOptionsByGroup = computed<Record<number, Tag[]>>(() => {
  const grouped: Record<number, Tag[]> = {}

  for (const tag of tagOptions.value) {
    if (!tag.is_active) continue
    if (!grouped[tag.group_id]) {
      grouped[tag.group_id] = []
    }
    grouped[tag.group_id].push(tag)
  }

  for (const groupId of Object.keys(grouped)) {
    grouped[Number(groupId)].sort((a, b) => a.sort_order - b.sort_order || a.id - b.id)
  }

  return grouped
})

const tagSelectionsByGroup = computed<Record<number, string | number | Array<string | number> | undefined>>(() => {
  const grouped: Record<number, Array<string | number>> = {}

  for (const tagId of form.value.tag_ids) {
    if (typeof tagId !== 'number') continue
    const groupId = tagGroupIdByTagId.value.get(tagId)

    if (!groupId) continue

    if (!grouped[groupId]) {
      grouped[groupId] = []
    }

    grouped[groupId].push(tagId)
  }

  for (const [groupId, drafts] of Object.entries(pendingTagDraftsByGroup.value)) {
    const normalizedGroupId = Number(groupId)
    if (!grouped[normalizedGroupId]) {
      grouped[normalizedGroupId] = []
    }
    grouped[normalizedGroupId].push(...drafts)
  }

  const selections: Record<number, string | number | Array<string | number> | undefined> = {}

  for (const group of tagGroups.value) {
    const values = grouped[group.id] || []
    selections[group.id] = group.allow_multiple ? values : values[0]
  }

  return selections
})

// Files to delete only after successful submit
const pendingDeleteAssets = ref<Array<{ type: 'cover' | 'banner' | 'screenshot' | 'video'; path: string; assetId?: number; assetUid?: string }>>([])
const primaryPreviewVideo = computed(() => {
  if (form.value.preview_videos.length === 0) return null
  const selected = form.value.preview_videos.find((item) => item.asset_uid === form.value.primary_preview_video_uid)
  return selected || form.value.preview_videos[0]
})
const previewVideoSources = computed(() => resolveAssetCandidates(primaryPreviewVideo.value?.path || ''))

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
  return sortCreatableOptionsByName(developerOptions.value)
})

const handleDeveloperSearch = async (query: string) => {
  if (!query) return
  isSearchingDevelopers.value = true
  try {
    const { developersService } = await import('@/services/developers.service')
    developerOptions.value = await searchCreatableOptions({
      query,
      selectedValues: form.value.developers,
      currentOptions: developerOptions.value,
      search: (keyword) => developersService.searchDevelopers(keyword),
    })
  } finally {
    isSearchingDevelopers.value = false
  }
}

const filteredPublisherOptions = computed(() => {
  return sortCreatableOptionsByName(publisherOptions.value)
})

const handlePublisherSearch = async (query: string) => {
  if (!query) return
  isSearchingPublishers.value = true
  try {
    const { publishersService } = await import('@/services/publishers.service')
    publisherOptions.value = await searchCreatableOptions({
      query,
      selectedValues: form.value.publishers,
      currentOptions: publisherOptions.value,
      search: (keyword) => publishersService.searchPublishers(keyword),
    })
  } finally {
    isSearchingPublishers.value = false
  }
}

const form = ref<GameForm>({
  title: '',
  title_alt: '',
  visibility: 'public',
  developers: [],
  publishers: [],
  release_date: undefined,
  engine: '',
  platform: [],
  series: null,
  tag_ids: [],
  summary: '',
  cover_image: '',
  banner_image: '',
  preview_videos: [],
  primary_preview_video_uid: '',
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
const draggedScreenshotKey = ref<string | null>(null)
const dragOverScreenshotKey = ref<string | null>(null)
const showVideoSelector = ref(false)
const videoFileInput = ref<HTMLInputElement | null>(null)
const isUploadingVideo = ref(false)
const videoUploadProgress = ref(0)
const videoUploadFileName = ref('')
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

const createScreenshotKey = (asset: Pick<EditableScreenshot, 'id' | 'asset_uid' | 'path'>, index = 0) => {
  if (asset.asset_uid) return `uid:${asset.asset_uid}`
  if (typeof asset.id === 'number') return `db:${asset.id}`
  return `path:${asset.path}:${index}:${Date.now()}`
}

const createEditableScreenshot = (
  asset: ScreenshotItem | UploadedAssetResult | string,
  index: number,
): EditableScreenshot => {
  if (typeof asset === 'string') {
    return {
      path: asset,
      sort_order: index,
      client_key: createScreenshotKey({ path: asset }, index),
    }
  }

  const screenshotId = 'id' in asset ? asset.id : ('asset_id' in asset ? asset.asset_id : undefined)
  const screenshotSortOrder = 'sort_order' in asset ? asset.sort_order : index

  return {
    id: screenshotId,
    asset_uid: asset.asset_uid,
    path: asset.path,
    sort_order: screenshotSortOrder ?? index,
    client_key: createScreenshotKey({
      id: screenshotId,
      asset_uid: asset.asset_uid,
      path: asset.path,
    }, index),
  }
}

const createEditableVideo = (asset: VideoAssetItem | UploadedAssetResult | string): EditableVideo => {
  if (typeof asset === 'string') {
    return { path: asset }
  }
  return {
    id: 'id' in asset ? asset.id : ('asset_id' in asset ? asset.asset_id : undefined),
    asset_uid: asset.asset_uid,
    path: asset.path,
    sort_order: 'sort_order' in asset ? asset.sort_order : undefined,
  }
}

const getEditableVideoKey = (video: EditableVideo) => {
  return video.asset_uid || video.path
}

const reorderEditableVideos = (targetKey: string, direction: -1 | 1) => {
  const videos = [...form.value.preview_videos]
  const index = videos.findIndex((item) => getEditableVideoKey(item) === targetKey)
  if (index === -1) return
  const nextIndex = index + direction
  if (nextIndex < 0 || nextIndex >= videos.length) return

  const [moved] = videos.splice(index, 1)
  videos.splice(nextIndex, 0, moved)
  form.value.preview_videos = videos.map((item, order) => ({
    ...item,
    sort_order: order,
  }))
}

const reorderEditableScreenshots = (fromKey: string, toKey: string) => {
  const screenshots = [...form.value.screenshots]
  const fromIndex = screenshots.findIndex((item) => item.client_key === fromKey)
  const toIndex = screenshots.findIndex((item) => item.client_key === toKey)
  if (fromIndex === -1 || toIndex === -1 || fromIndex === toIndex) return

  const [moved] = screenshots.splice(fromIndex, 1)
  screenshots.splice(toIndex, 0, moved)
  form.value.screenshots = screenshots.map((item, index) => ({
    ...item,
    sort_order: index,
  }))
}

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
  visibility: 'public',
  developers: [],
  publishers: [],
  release_date: undefined,
  engine: '',
  platform: [],
  series: null,
  tag_ids: [],
  summary: '',
  cover_image: '',
  banner_image: '',
  preview_videos: [],
  primary_preview_video_uid: '',
  screenshots: [],
  file_paths: [{ path: '', label: '' }],
})

const resetTransientState = () => {
  pendingTagDraftsByGroup.value = {}
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
  steamSummaryDetails.value = null

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
  showVideoSelector.value = false
  videoUploadProgress.value = 0
  videoUploadFileName.value = ''
  isUploadingVideo.value = false
  if (videoFileInput.value) {
    videoFileInput.value.value = ''
  }
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
  const developerIds: Array<string | number> = game.developers
    ? game.developers.map((item) => item.id)
    : []
  const publisherIds: Array<string | number> = game.publishers
    ? game.publishers.map((item) => item.id)
    : []

  form.value = {
    title: game.title || '',
    title_alt: game.title_alt || '',
    visibility: game.visibility || 'public',
    developers: developerIds,
    publishers: publisherIds,
    release_date: game.release_date || undefined,
    engine: game.engine || '',
    platform: platformList,
    series: seriesId,
    tag_ids: (game.tags || []).map((item) => item.id),
    summary: game.summary || '',
    cover_image: game.cover_image || '',
    banner_image: game.banner_image || '',
    preview_videos: (game.preview_videos || (game.preview_video ? [game.preview_video] : [])).map((asset) => createEditableVideo(asset)),
    primary_preview_video_uid: game.preview_video?.asset_uid || game.preview_videos?.[0]?.asset_uid || '',
    screenshots: (game.screenshot_items || game.screenshots || []).map((asset, index) =>
      createEditableScreenshot(asset, index),
    ),
    file_paths: filePaths,
  }
  pendingTagDraftsByGroup.value = {}

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
    if (currentGame?.developers && currentGame.developers.length > 0) {
      for (const developer of currentGame.developers) {
        const existing = developerOptions.value.find((item) => item.id === developer.id)
        if (!existing) {
          developerOptions.value.push(developer as any)
        }
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
    if (currentGame?.publishers && currentGame.publishers.length > 0) {
      for (const publisher of currentGame.publishers) {
        const existing = publisherOptions.value.find((item) => item.id === publisher.id)
        if (!existing) {
          publisherOptions.value.push(publisher as any)
        }
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

  try {
    const [loadedGroups, loadedTags] = await Promise.all([
      tagsService.getTagGroups(),
      tagsService.getTags({ active: true }),
    ])
    tagGroups.value = loadedGroups.sort((a, b) => a.sort_order - b.sort_order || a.id - b.id)
    tagOptions.value = loadedTags
  } catch (error) {
    console.error('Failed to load tags:', error)
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

const queueAssetDeletion = (
  type: 'cover' | 'banner' | 'screenshot' | 'video',
  path: string,
  assetId?: number,
  assetUid?: string,
) => {
  if (!path) return
  pendingDeleteAssets.value.push({ type, path, assetId, assetUid })
}

const handleScreenshotDragStart = (clientKey: string) => {
  draggedScreenshotKey.value = clientKey
  dragOverScreenshotKey.value = clientKey
}

const handleScreenshotDragEnter = (clientKey: string) => {
  if (!draggedScreenshotKey.value || draggedScreenshotKey.value === clientKey) return
  dragOverScreenshotKey.value = clientKey
}

const handleScreenshotDrop = (clientKey: string) => {
  if (!draggedScreenshotKey.value) return
  reorderEditableScreenshots(draggedScreenshotKey.value, clientKey)
  draggedScreenshotKey.value = null
  dragOverScreenshotKey.value = null
}

const handleScreenshotDragEnd = () => {
  draggedScreenshotKey.value = null
  dragOverScreenshotKey.value = null
}

const uploadAssetFromUrl = async (
  url: string,
  assetType: 'cover' | 'banner' | 'screenshot' | 'video',
  sortOrder = 0,
) => {
  if (!props.game?.id) {
    throw new Error('缺少游戏 ID')
  }

  const response = await fetch(proxySteamAssetUrl(url))
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
    coverPreviewUrl.value = proxySteamAssetUrl(coverSearchUrl.value.trim())
  }
}

const confirmCoverSelection = async () => {
  if (!coverPreviewUrl.value) return
  isDownloadingCover.value = true
  try {
    const uploaded = await uploadAssetFromUrl(coverPreviewUrl.value, 'cover')
    if (form.value.cover_image) {
      queueAssetDeletion('cover', form.value.cover_image)
    }
    form.value.cover_image = uploaded.path
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
    screenshotPreviewUrl.value = proxySteamAssetUrl(screenshotSearchUrl.value.trim())
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

const applySteamMetadataToForm = (details: { releaseDate?: string; developers?: string[]; publishers?: string[] }) => {
  if (details.releaseDate) {
    form.value.release_date = details.releaseDate
    releaseDate.value = new Date(`${details.releaseDate}T00:00:00`)
  }
  if (details.developers && details.developers.length > 0) {
    const merged = new Set<string | number>(form.value.developers)
    for (const name of details.developers) {
      if (name.trim()) merged.add(name.trim())
    }
    form.value.developers = Array.from(merged)
  }
  if (details.publishers && details.publishers.length > 0) {
    const merged = new Set<string | number>(form.value.publishers)
    for (const name of details.publishers) {
      if (name.trim()) merged.add(name.trim())
    }
    form.value.publishers = Array.from(merged)
  }
}

const handleSummarySearchClear = () => {
  summarySteamPicker.clear()
  steamSummaryPreview.value = ''
  steamSummaryDetails.value = null
}

const searchSteamForSummary = async () => {
  steamSummaryPreview.value = ''
  steamSummaryDetails.value = null
  await summarySteamPicker.search()
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
  steamSummaryPreview.value = ''
  steamSummaryDetails.value = null
  await summarySteamPicker.select(game)
}

const backToSummarySearch = () => {
  summarySteamPicker.back()
  steamSummaryPreview.value = ''
  steamSummaryDetails.value = null
}

const confirmSummaryImport = () => {
  const details = steamSummaryDetails.value
  const hasImportableMetadata = !!details?.releaseDate || !!details?.developers?.[0] || !!details?.publishers?.[0]
  if (!steamSummaryPreview.value && !hasImportableMetadata) {
    uiStore.addAlert('当前没有可导入的 Steam 信息', 'warning')
    return
  }

  if (steamSummaryPreview.value) {
    form.value.summary = steamSummaryPreview.value
  }
  if (details) {
    applySteamMetadataToForm(details)
  }
  showSummarySelector.value = false
  uiStore.addAlert(
    `已导入 Steam 信息：${selectedSteamSummaryGame.value?.name || 'Steam 游戏'}`,
    'success',
  )
}

const confirmScreenshotSelection = async () => {
  if (!screenshotPreviewUrl.value) return
  isDownloadingScreenshot.value = true
  try {
    const uploaded = await uploadAssetFromUrl(screenshotPreviewUrl.value, 'screenshot', form.value.screenshots.length)
    form.value.screenshots.push(createEditableScreenshot(uploaded, form.value.screenshots.length))
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
    form.value.screenshots.push(
      createEditableScreenshot(response.data, form.value.screenshots.length),
    )
    showScreenshotSelector.value = false
    uiStore.addAlert('截图上传成功', 'success')
  } else {
    uiStore.addAlert('上传失败：' + (response?.error || '未知错误'), 'error')
  }
}

const handleScreenshotUploadError = () => {
  uiStore.addAlert('截图上传失败', 'error')
}

const appendPreviewVideo = (video: EditableVideo) => {
  form.value.preview_videos.push(video)
  if (!form.value.primary_preview_video_uid && video.asset_uid) {
    form.value.primary_preview_video_uid = video.asset_uid
  }
}

const openVideoSelector = () => {
  showVideoSelector.value = true
}

const setPrimaryPreviewVideo = (assetUid?: string) => {
  if (!assetUid) return
  form.value.primary_preview_video_uid = assetUid
}

const openVideoFilePicker = () => {
  if (isUploadingVideo.value) return
  videoFileInput.value?.click()
}

const handleVideoFileChange = async (event: Event) => {
  const input = event.target as HTMLInputElement
  const file = input.files?.[0]
  if (!file || !props.game?.id) return

  isUploadingVideo.value = true
  videoUploadProgress.value = 0
  videoUploadFileName.value = file.name

  try {
    const uploaded = await uploadAsset('video', props.game.id, file, form.value.preview_videos.length, (percent) => {
      videoUploadProgress.value = percent
    })
    appendPreviewVideo(createEditableVideo(uploaded))
    videoUploadProgress.value = 100
    uiStore.addAlert('预告片上传成功', 'success')
  } catch (error: any) {
    videoUploadProgress.value = 0
    uiStore.addAlert('预告片上传失败：' + (error?.message || '未知错误'), 'error')
  } finally {
    isUploadingVideo.value = false
    if (videoFileInput.value) {
      videoFileInput.value.value = ''
    }
  }
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

const removeScreenshot = (clientKey: string) => {
  const screenshot = form.value.screenshots.find((item) => item.client_key === clientKey)
  if (!screenshot) return
  queueAssetDeletion('screenshot', screenshot.path, screenshot.id, screenshot.asset_uid)
  form.value.screenshots = form.value.screenshots.filter((item) => item.client_key !== clientKey)
}

const removePreviewVideo = (assetUid?: string) => {
  const target = form.value.preview_videos.find((item) => item.asset_uid === assetUid)
  if (!target) return
  queueAssetDeletion('video', target.path, target.id, target.asset_uid)
  form.value.preview_videos = form.value.preview_videos.filter((item) => item.asset_uid !== assetUid)
  if (form.value.primary_preview_video_uid === assetUid) {
    form.value.primary_preview_video_uid = form.value.preview_videos[0]?.asset_uid || ''
  }
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
    const existingPath = (form.value.file_paths[index]?.path || '').trim()
    if (!existingPath) {
      initialPath.value = defaultPath
    } else if (!existingPath.includes('/') && !existingPath.includes('\\')) {
      initialPath.value = defaultPath
    } else {
      initialPath.value = existingPath.replace(/[\\/][^\\/]*$/, '') || defaultPath
    }
    showFileBrowser.value = true
  } catch (error) {
    console.error('Failed to get default directory:', error)
  }
}

// Steam 封面搜索
const handleCoverSearchClear = () => {
  coverSteamPicker.clear()
  steamCoverImages.value = []
  selectedCoverImage.value = ''
}

const searchSteamForCover = async () => {
  steamCoverImages.value = []
  selectedCoverImage.value = ''
  await coverSteamPicker.search()
}

// Steam 横幅搜索
const handleBannerSearchClear = () => {
  bannerSteamPicker.clear()
  steamBannerImages.value = []
  selectedBannerImage.value = ''
}

const searchSteamForBanner = async () => {
  steamBannerImages.value = []
  selectedBannerImage.value = ''
  await bannerSteamPicker.search()
}

const selectSteamBannerGame = async (game: any) => {
  await bannerSteamPicker.select(game)
}

const backToBannerGameSearch = () => {
  bannerSteamPicker.back()
  steamBannerImages.value = []
}

const downloadSelectedSteamBanner = async () => {
  if (!selectedBannerImage.value || !props.game?.id) return

  isDownloadingBanner.value = true
  try {
    const uploaded = await uploadAssetFromUrl(selectedBannerImage.value, 'banner')
    if (form.value.banner_image) {
      queueAssetDeletion('banner', form.value.banner_image)
    }
    form.value.banner_image = uploaded.path
    showBannerSelector.value = false
    backToBannerGameSearch()
    steamBannerSearchQuery.value = ''
    steamBannerSearchResults.value = []
    bannerSearchUrl.value = ''
    bannerPreviewUrl.value = ''
    uiStore.addAlert('横幅下载成功', 'success')
  } catch (error: any) {
    uiStore.addAlert('下载失败：' + error.message, 'error')
  } finally {
    isDownloadingBanner.value = false
  }
}

const loadBannerFromUrl = async () => {
  if (!bannerSearchUrl.value.trim()) return

  isDownloadingBanner.value = true
  try {
    const uploaded = await uploadAssetFromUrl(bannerSearchUrl.value, 'banner')
    if (form.value.banner_image) {
      queueAssetDeletion('banner', form.value.banner_image)
    }
    form.value.banner_image = uploaded.path
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
  if (!isOpen) return
  const query = pickSteamSearchQuery()
  if (!query) return
  steamCoverSearchQuery.value = query
  searchSteamForCover()
})

// 当横幅选择器打开时，自动使用英文名搜索
watch(showBannerSelector, (isOpen) => {
  if (!isOpen) return
  const query = pickSteamSearchQuery()
  if (!query) return
  steamBannerSearchQuery.value = query
  searchSteamForBanner()
})

const selectSteamCoverGame = async (game: any) => {
  await coverSteamPicker.select(game)
}

const backToCoverGameSearch = () => {
  coverSteamPicker.back()
  steamCoverImages.value = []
  selectedCoverImage.value = ''
}

const downloadSelectedSteamCover = async () => {
  if (!selectedCoverImage.value || !props.game?.id) return

  isSearchingSteamCover.value = true
  try {
    const uploaded = await uploadAssetFromUrl(selectedCoverImage.value, 'cover')
    if (form.value.cover_image) {
      queueAssetDeletion('cover', form.value.cover_image)
    }
    form.value.cover_image = uploaded.path
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
  screenshotSteamPicker.clear()
  steamScreenshotsData.value = null
  selectedSteamScreenshots.value.clear()
}

const searchSteamForScreenshots = async () => {
  steamScreenshotsData.value = null
  selectedSteamScreenshots.value.clear()
  await screenshotSteamPicker.search()
}

// 当截图选择器打开时，自动使用英文名搜索
watch(showScreenshotSelector, (isOpen) => {
  if (!isOpen) return
  const query = pickSteamSearchQuery()
  if (!query) return
  steamScreenshotSearchQuery.value = query
  searchSteamForScreenshots()
})

const selectSteamScreenshotGame = async (game: any) => {
  await screenshotSteamPicker.select(game)
}

const backToScreenshotGameSearch = () => {
  screenshotSteamPicker.back()
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
      const uploaded = await uploadAssetFromUrl(screenshotUrl, 'screenshot', currentIndex)
      form.value.screenshots.push(createEditableScreenshot(uploaded, currentIndex))
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

const slugifyMetadataName = (name: string) => {
  return name
    .trim()
    .toLowerCase()
    .replace(/[^a-z0-9]+/g, '-')
    .replace(/^-+|-+$/g, '')
}

const pendingTagOptionsByGroup = computed<Record<number, Array<{ value: string; label: string }>>>(() => {
  const grouped: Record<number, Array<{ value: string; label: string }>> = {}

  for (const [groupId, names] of Object.entries(pendingTagDraftsByGroup.value)) {
    const normalizedGroupId = Number(groupId)
    grouped[normalizedGroupId] = names.map((name) => ({
      value: name,
      label: name,
    }))
  }

  return grouped
})

const resolveTagSelections = async () => {
  const idsByGroup = new Map<number, number[]>()

  for (const tagId of form.value.tag_ids) {
    const normalizedId = normalizeOptionId(tagId)
    if (normalizedId === null) continue
    const groupId = tagGroupIdByTagId.value.get(normalizedId)
    if (!groupId) continue
    const current = idsByGroup.get(groupId) || []
    current.push(normalizedId)
    idsByGroup.set(groupId, current)
  }

  for (const group of tagGroups.value) {
    const values: Array<string | number> = [
      ...(idsByGroup.get(group.id) || []),
      ...(pendingTagDraftsByGroup.value[group.id] || []),
    ]
    if (values.length === 0) continue

    const result = await resolveCreatableSelections({
      values,
      options: tagOptions.value,
      findExisting: (name, options) =>
        options.find((item) => item.group_id === group.id && item.name.trim().toLowerCase() === name.toLowerCase()),
      createItem: (name) =>
        tagsService.createTag({
          group_id: group.id,
          name,
          slug: slugifyMetadataName(name),
        }),
    })

    tagOptions.value = result.options
    idsByGroup.set(group.id, result.ids)
  }

  pendingTagDraftsByGroup.value = {}
  return Array.from(idsByGroup.values()).flat()
}

const handleFormTagChange = (groupId: number, value: number | number[] | string | string[] | undefined) => {
  const rawValues = Array.isArray(value) ? value : value === undefined || value === null || value === '' ? [] : [value]
  const nextIds: number[] = []
  const nextDrafts: string[] = []

  for (const item of rawValues) {
    const normalizedId = normalizeOptionId(item)
    if (normalizedId !== null) {
      nextIds.push(normalizedId)
      continue
    }

    if (typeof item !== 'string') continue

    const name = item.trim()
    if (!name) continue

    const existing = tagOptions.value.find(
      (tag) => tag.group_id === groupId && tag.name.trim().toLowerCase() === name.toLowerCase(),
    )
    if (existing) {
      nextIds.push(existing.id)
      continue
    }

    if (!nextDrafts.some((draft) => draft.toLowerCase() === name.toLowerCase())) {
      nextDrafts.push(name)
    }
  }

  const preserved = form.value.tag_ids.filter((tagId) => {
    const normalizedId = normalizeOptionId(tagId)
    if (normalizedId === null) return false
    return tagGroupIdByTagId.value.get(normalizedId) !== groupId
  })

  form.value.tag_ids = [...preserved, ...nextIds]
  pendingTagDraftsByGroup.value = {
    ...pendingTagDraftsByGroup.value,
    [groupId]: nextDrafts,
  }
}

const handleTagSelectionChange = (groupId: number, value: number | number[] | string | string[] | undefined) => {
  handleFormTagChange(groupId, value)
}

const handleParseWikiTags = async () => {
  const content = props.game?.wiki_content || ''
  if (!content.trim()) {
    uiStore.addAlert('当前游戏没有可解析的 Wiki 内容', 'warning')
    return
  }

  if (tagGroups.value.length === 0) {
    uiStore.addAlert('当前没有可用标签组', 'warning')
    return
  }

  isPreparingWikiTagCandidates.value = true

  try {
    const extracted = extractWikiTagCandidates(content)
    if (extracted.length === 0) {
      uiStore.addAlert('没有识别到可提取的“类型：...”字段', 'warning')
      return
    }

    wikiTagCandidates.value = extracted.map((item) => ({
      key: `${item.sourceLabel}:${item.value.toLowerCase()}`,
      value: item.value,
      sourceLabel: item.sourceLabel,
      groupKey: 'ignore',
    }))
    wikiTagPickerVisible.value = true
  } catch (error) {
    console.error('Failed to extract wiki tags:', error)
    uiStore.addAlert('从 Wiki 提取字段失败', 'warning')
  } finally {
    isPreparingWikiTagCandidates.value = false
  }
}

const handleWikiTagCandidateGroupChange = (
  key: string,
  value: WikiTagGroupKey | 'ignore' | number | string | undefined,
) => {
  const nextValue: WikiTagGroupKey | 'ignore' =
    value === 'genre' || value === 'subgenre' || value === 'perspective' || value === 'theme'
      ? value
      : 'ignore'

  wikiTagCandidates.value = wikiTagCandidates.value.map((item) =>
    item.key === key
      ? {
          ...item,
          groupKey: nextValue,
        }
      : item,
  )
}

const applySelectedWikiTags = async () => {
  const selected = wikiTagCandidates.value.filter((item) => item.groupKey !== 'ignore')
  if (selected.length === 0) {
    uiStore.addAlert('还没有选择要应用的字段', 'warning')
    return
  }

  isApplyingWikiTags.value = true

  try {
    const mergedIds = new Set<number>(
      form.value.tag_ids
        .map((item) => normalizeOptionId(item))
        .filter((item): item is number => item !== null),
    )

    const grouped = new Map<WikiTagGroupKey, string[]>()
    for (const item of selected) {
      const values = grouped.get(item.groupKey) || []
      if (!values.some((value) => value.toLowerCase() === item.value.toLowerCase())) {
        values.push(item.value)
      }
      grouped.set(item.groupKey, values)
    }

    const appliedLabels: string[] = []

    for (const [groupKey, names] of grouped.entries()) {
      const group = tagGroups.value.find((item) => item.key === groupKey)
      if (!group) continue

      const result = await resolveCreatableSelections({
        values: names,
        options: tagOptions.value,
        findExisting: (name, options) =>
          options.find((item) => item.group_id === group.id && item.name.trim().toLowerCase() === name.toLowerCase()),
        createItem: (name) =>
          tagsService.createTag({
            group_id: group.id,
            name,
          }),
      })

      tagOptions.value = result.options
      for (const id of result.ids) {
        mergedIds.add(id)
      }

      if (result.ids.length > 0) {
        appliedLabels.push(`${group.name}：${names.join('、')}`)
      }
    }

    form.value.tag_ids = Array.from(mergedIds)
    pendingTagDraftsByGroup.value = {}
    wikiTagPickerVisible.value = false

    if (appliedLabels.length === 0) {
      uiStore.addAlert('已选择字段，但没有成功应用到标签组', 'warning')
      return
    }

    uiStore.addAlert(`已应用 Wiki 字段：${appliedLabels.join('；')}`, 'success')
  } catch (error) {
    console.error('Failed to apply wiki tags:', error)
    uiStore.addAlert('应用 Wiki 字段失败', 'warning')
  } finally {
    isApplyingWikiTags.value = false
  }
}

const handleCancel = () => {
  visible.value = false
  pendingDeleteAssets.value = []
}

const handleSubmit = async () => {
  if (!props.game) return
  if (isSubmitting.value) return

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
      const result = await resolveCreatableSelections({
        values: form.value.developers,
        options: developerOptions.value,
        createItem: (name) => developersService.createDeveloper({
          name,
          slug: slugifyMetadataName(name),
        }),
      })
      developerOptions.value = result.options
      developerIds = result.ids
      form.value.developers = [...developerIds]
    } catch (error: any) {
      console.error('Failed to process developers:', form.value.developers, error)
      uiStore.addAlert('开发商处理失败', 'warning')
    }

    // Process publisher - handle both existing ID and new name
    let publisherIds: number[] | undefined = undefined
    try {
      const { publishersService } = await import('@/services/publishers.service')
      const result = await resolveCreatableSelections({
        values: form.value.publishers,
        options: publisherOptions.value,
        createItem: (name) => publishersService.createPublisher({
          name,
          slug: slugifyMetadataName(name),
        }),
      })
      publisherOptions.value = result.options
      publisherIds = result.ids
      form.value.publishers = [...publisherIds]
    } catch (error: any) {
      console.error('Failed to process publishers:', form.value.publishers, error)
      uiStore.addAlert('发行商处理失败', 'warning')
    }

    let platformIds: number[] = []
    try {
      const result = await resolveCreatableSelections({
        values: form.value.platform,
        options: platformOptions.value,
        createItem: (name) => platformService.createPlatform({
          name,
          slug: slugifyMetadataName(name),
        }),
      })
      platformOptions.value = result.options
      platformIds = result.ids
      form.value.platform = [...platformIds]
    } catch (error: any) {
      console.error('Failed to process platform:', form.value.platform, error)
      uiStore.addAlert('平台处理失败', 'warning')
    }

    let tagIds: number[] = []
    try {
      tagIds = await resolveTagSelections()
      form.value.tag_ids = [...tagIds]
    } catch (error: any) {
      console.error('Failed to process tags:', form.value.tag_ids, error)
      uiStore.addAlert('标签处理失败', 'warning')
    }

    // Submit game update with series, developers, publishers
    await gamesService.updateGame(String(props.game.id), {
      title: form.value.title,
      title_alt: form.value.title_alt,
      visibility: form.value.visibility,
      release_date: form.value.release_date || undefined,
      engine: form.value.engine,
      platforms: platformIds,
      series: seriesIds,
      developers: developerIds,
      publishers: publisherIds,
      tag_ids: tagIds,
      summary: form.value.summary,
      cover_image: form.value.cover_image,
      banner_image: form.value.banner_image,
      preview_video_asset_uid: form.value.primary_preview_video_uid || null,
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
          await deleteAsset(props.game.id, item.type, item.path, item.assetId, item.assetUid)
        } catch (e) {
          console.error('Failed to delete asset after save:', item.path, e)
        }
      }
      pendingDeleteAssets.value = []
    }

    const orderedScreenshotUids = form.value.screenshots
      .map((item, index) => {
        item.sort_order = index
        return item.asset_uid
      })
      .filter((assetUid): assetUid is string => !!assetUid)
    if (orderedScreenshotUids.length > 0) {
      await reorderScreenshots(props.game.id, orderedScreenshotUids)
    }

    const orderedVideoUids = form.value.preview_videos
      .map((item, index) => {
        item.sort_order = index
        return item.asset_uid
      })
      .filter((assetUid): assetUid is string => !!assetUid)
    if (orderedVideoUids.length > 0) {
      await reorderVideos(props.game.id, orderedVideoUids)
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
:deep(.arco-modal-header .arco-modal-title) {
  font-weight: 700;
}

:deep(.arco-form-item-label-col > label) {
  font-weight: 700;
}

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
  font-weight: 700;
}

.tag-group-grid {
  display: grid;
  grid-template-columns: repeat(4, minmax(0, 1fr));
  gap: 12px;
  width: 100%;
}

.tag-group-field {
  display: flex;
  flex-direction: column;
  gap: 6px;
}

.tag-group-select {
  width: 100%;
}

.tag-group-select :deep(.arco-select-view) {
  min-height: 36px;
  align-items: flex-start;
}

.tag-group-select :deep(.arco-select-view-value) {
  flex-wrap: wrap;
  gap: 4px;
}

.tag-group-select :deep(.arco-select-view-tag) {
  max-width: 100%;
}

.tag-group-field__label {
  display: flex;
  font-size: 13px;
  font-weight: 700;
  color: var(--color-text-2);
}

.tag-group-empty {
  font-size: 12px;
  color: var(--color-text-3);
}

.wiki-tag-picker {
  display: flex;
  flex-direction: column;
  gap: 14px;
}

.wiki-tag-picker__hint {
  font-size: 13px;
  color: var(--color-text-2);
}

.wiki-tag-picker__list {
  display: flex;
  flex-direction: column;
  gap: 10px;
  max-height: 420px;
  overflow-y: auto;
}

.wiki-tag-picker__item {
  display: grid;
  grid-template-columns: minmax(0, 1fr) 180px;
  gap: 12px;
  align-items: center;
  padding: 12px 14px;
  border: 1px solid var(--color-border-2);
  border-radius: 10px;
  background: var(--color-fill-1);
}

.wiki-tag-picker__meta {
  min-width: 0;
}

.wiki-tag-picker__value {
  font-size: 14px;
  font-weight: 700;
  color: var(--color-text-1);
  word-break: break-word;
}

.wiki-tag-picker__source {
  margin-top: 4px;
  font-size: 12px;
  color: var(--color-text-3);
}

.wiki-tag-picker__select {
  width: 100%;
}

@media (max-width: 1200px) {
  .tag-group-grid {
    grid-template-columns: repeat(2, minmax(0, 1fr));
  }
}

@media (max-width: 768px) {
  .tag-group-grid {
    grid-template-columns: 1fr;
  }

  .wiki-tag-picker__item {
    grid-template-columns: 1fr;
  }
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

.media-section--cover {
  max-width: 88%;
  margin: 0 auto;
}

.media-subitem {
  margin-top: 8px;
}

.media-frame {
  width: 100%;
  overflow: hidden;
  border-radius: 8px;
  border: 1px solid var(--app-card-border);
  background: color-mix(in srgb, var(--app-card-surface) 90%, transparent);
  backdrop-filter: blur(var(--app-card-backdrop-blur));
  -webkit-backdrop-filter: blur(var(--app-card-backdrop-blur));
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
  background: color-mix(in srgb, var(--app-card-surface) 88%, transparent);
  backdrop-filter: blur(var(--app-card-backdrop-blur));
  -webkit-backdrop-filter: blur(var(--app-card-backdrop-blur));
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

.media-empty-action--inner {
  border: none;
  border-radius: 0;
}

.media-empty-icon {
  font-size: 30px;
}

.media-empty-title {
  font-size: 14px;
  font-weight: 700;
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
  pointer-events: none;
  transition: opacity 0.2s ease;
}

.media-overlay--top-right {
  align-items: flex-start;
  justify-content: flex-end;
  padding: 14px;
}

.media-overlay-actions {
  display: inline-flex;
  align-items: center;
  gap: 12px;
  pointer-events: auto;
}

.media-action-button {
  width: 40px;
  height: 40px;
  min-width: 40px;
  padding: 0;
  border-radius: 999px;
  backdrop-filter: blur(8px);
  box-shadow: 0 10px 24px rgba(0, 0, 0, 0.24);
  transition: transform 0.2s ease;
}

.media-action-button:hover {
  transform: scale(1.06);
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

.media-frame--video {
  aspect-ratio: 16 / 9;
}

.media-video {
  width: 100%;
  height: 100%;
  display: block;
  background: #000;
}

.hidden-file-input {
  display: none;
}

.video-upload-progress {
  margin-top: 14px;
  padding: 12px 14px;
  border: 1px solid var(--app-card-border);
  border-radius: 12px;
  background: color-mix(in srgb, var(--app-card-surface) 84%, transparent);
  backdrop-filter: blur(var(--app-card-backdrop-blur));
  -webkit-backdrop-filter: blur(var(--app-card-backdrop-blur));
}

.video-upload-progress__meta {
  display: flex;
  align-items: center;
  justify-content: space-between;
  margin-bottom: 8px;
  color: rgba(255, 255, 255, 0.75);
  font-size: 13px;
}

.media-actions {
  margin-top: 4px;
}

/* Screenshots Grid */
.screenshots-grid {
  display: grid;
  grid-template-columns: repeat(2, minmax(0, 1fr));
  gap: 8px;
}

.screenshots-frame {
  align-items: stretch;
  justify-content: stretch;
}

.screenshots-grid-wrapper {
  width: 100%;
  height: 100%;
  padding: 10px;
  overflow-y: auto;
}

.screenshot-thumb {
  aspect-ratio: 16/9;
  border-radius: 6px;
  overflow: hidden;
  background: color-mix(in srgb, var(--app-card-surface) 86%, transparent);
  cursor: grab;
  position: relative;
  border: 1px solid var(--app-card-border);
  transition: transform 0.18s ease, border-color 0.18s ease, box-shadow 0.18s ease, opacity 0.18s ease;
}

.screenshot-thumb.is-dragging {
  opacity: 0.45;
  transform: scale(0.98);
  cursor: grabbing;
}

.screenshot-thumb.is-drop-target {
  border-color: rgb(var(--primary-6));
  box-shadow: 0 0 0 1px rgba(var(--primary-6), 0.35);
}

.screenshot-thumb img {
  width: 100%;
  height: 100%;
  object-fit: cover;
}

.screenshot-add-tile {
  width: 100%;
  aspect-ratio: 16 / 9;
  border-radius: 6px;
  overflow: hidden;
  position: relative;
  border: 1px dashed rgba(255, 255, 255, 0.1);
  background: color-mix(in srgb, var(--app-card-surface) 88%, transparent);
  backdrop-filter: blur(var(--app-card-backdrop-blur));
  -webkit-backdrop-filter: blur(var(--app-card-backdrop-blur));
  color: var(--color-text-3);
  display: flex;
  align-items: center;
  justify-content: center;
  padding: 10px;
  box-sizing: border-box;
  cursor: pointer;
  transition: border-color 0.2s ease, color 0.2s ease, background 0.2s ease;
}

.screenshot-add-tile__label {
  font-size: 12px;
  font-weight: 700;
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
  background: color-mix(in srgb, var(--app-card-surface) 88%, transparent);
  border: 1px solid var(--app-card-border);
  backdrop-filter: blur(var(--app-card-backdrop-blur));
  -webkit-backdrop-filter: blur(var(--app-card-backdrop-blur));
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
  background: color-mix(in srgb, var(--app-card-surface) 88%, transparent);
  border: 1px solid var(--app-card-border);
  backdrop-filter: blur(var(--app-card-backdrop-blur));
  -webkit-backdrop-filter: blur(var(--app-card-backdrop-blur));
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

.url-input-row {
  display: flex;
  align-items: stretch;
  gap: 8px;
}

.url-input-row__field {
  flex: 1;
  min-width: 0;
}

.url-input-row__action {
  flex-shrink: 0;
  min-width: 72px;
}

.cover-selector-actions {
  display: flex;
  justify-content: flex-end;
  gap: 8px;
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
  background: color-mix(in srgb, var(--app-card-surface) 88%, transparent);
  border: 1px solid var(--app-card-border);
  backdrop-filter: blur(var(--app-card-backdrop-blur));
  -webkit-backdrop-filter: blur(var(--app-card-backdrop-blur));
  color: var(--color-text-2);
  line-height: 1.75;
  white-space: pre-wrap;
}

.steam-video-source-card {
  padding: 12px;
  border-radius: 8px;
  background: color-mix(in srgb, var(--app-card-surface) 88%, transparent);
  border: 1px solid var(--app-card-border);
  backdrop-filter: blur(var(--app-card-backdrop-blur));
  -webkit-backdrop-filter: blur(var(--app-card-backdrop-blur));
}

.steam-video-source-card__label {
  margin-bottom: 8px;
  color: var(--color-text-2);
  font-size: 12px;
  font-weight: 700;
}

.steam-video-source-list {
  display: flex;
  flex-direction: column;
  gap: 8px;
}

.steam-video-source-item {
  display: flex;
  align-items: center;
  gap: 8px;
}

.steam-video-debug {
  padding: 10px 12px;
  border-radius: 8px;
  border: 1px solid var(--app-card-border);
  background: color-mix(in srgb, var(--app-card-surface) 84%, transparent);
  backdrop-filter: blur(var(--app-card-backdrop-blur));
  -webkit-backdrop-filter: blur(var(--app-card-backdrop-blur));
  color: var(--color-text-2);
  font-size: 12px;
}

.steam-video-debug__title {
  font-size: 12px;
  font-weight: 600;
  color: var(--color-text-1);
  margin-bottom: 6px;
}

.steam-video-debug__line {
  line-height: 1.6;
  color: var(--color-text-3);
  font-family: ui-monospace, SFMono-Regular, Menlo, Monaco, Consolas, "Liberation Mono", "Courier New", monospace;
}

.video-library-card {
  margin-top: 14px;
  padding: 14px;
  border-radius: 12px;
  background: var(--app-card-surface);
  border: 1px solid var(--app-card-border);
  backdrop-filter: blur(var(--app-card-backdrop-blur));
  -webkit-backdrop-filter: blur(var(--app-card-backdrop-blur));
}

.video-library-card__header {
  display: flex;
  align-items: center;
  justify-content: space-between;
  margin-bottom: 12px;
  color: var(--color-text-2);
  font-size: 13px;
  font-weight: 600;
}

.video-library-list {
  display: flex;
  flex-direction: column;
  gap: 10px;
}

.video-library-item {
  display: grid;
  grid-template-columns: minmax(0, 1fr) auto;
  align-items: center;
  gap: 12px;
  padding: 10px 12px;
  border-radius: 10px;
  border: 1px solid var(--app-card-border);
  background: color-mix(in srgb, var(--app-card-surface) 84%, transparent);
  transition: border-color 0.2s ease, background 0.2s ease;
  overflow: hidden;
}

.video-library-item.is-primary {
  border-color: rgba(var(--primary-6), 0.45);
  background: rgba(var(--primary-6), 0.08);
}

.video-library-item__preview {
  display: flex;
  align-items: center;
  gap: 12px;
  width: 100%;
  flex: 1 1 auto;
  min-width: 0;
  padding: 0;
  overflow: hidden;
  cursor: pointer;
}

.video-library-item__preview:hover,
.video-library-item__preview:focus,
.video-library-item__preview:active {
  outline: none;
}

.video-library-item__thumb {
  width: 112px;
  height: 63px;
  flex-shrink: 0;
  border-radius: 8px;
  overflow: hidden;
  background: color-mix(in srgb, var(--app-card-surface) 82%, black 18%);
  display: flex;
  align-items: center;
  justify-content: center;
}

.video-library-item__thumb img {
  width: 100%;
  height: 100%;
  object-fit: cover;
}

.video-library-item__thumb-placeholder {
  color: rgba(255, 255, 255, 0.75);
  font-size: 22px;
}

.video-library-item__info {
  flex: 1;
  min-width: 0;
  display: flex;
  align-items: center;
  overflow: hidden;
}

.video-library-item__meta-row {
  display: flex;
  align-items: center;
  gap: 8px;
  min-width: 0;
  width: 100%;
  flex-wrap: nowrap;
  overflow: hidden;
}

.video-library-item__title {
  flex-shrink: 0;
  color: var(--color-text-1);
  font-size: 14px;
  font-weight: 600;
}

.video-library-item__path {
  min-width: 0;
  color: var(--color-text-3);
  font-size: 12px;
  line-height: 1.5;
  white-space: nowrap;
  overflow: hidden;
  text-overflow: ellipsis;
}

.video-library-item__actions {
  display: flex;
  align-items: center;
  justify-content: flex-end;
  flex-wrap: wrap;
  gap: 4px;
  flex-shrink: 0;
  min-width: fit-content;
}

@media (max-width: 360px) {
  .video-library-item {
    grid-template-columns: minmax(0, 1fr);
    align-items: stretch;
  }

  .video-library-item__actions {
    width: 100%;
    justify-content: flex-start;
  }
}

.video-library-empty {
  margin-top: 14px;
  padding: 20px 0;
  border-radius: 12px;
  background: color-mix(in srgb, var(--app-card-surface) 88%, transparent);
  border: 1px solid var(--app-card-border);
  backdrop-filter: blur(var(--app-card-backdrop-blur));
  -webkit-backdrop-filter: blur(var(--app-card-backdrop-blur));
}

.steam-images-grid {
  display: grid;
  grid-template-columns: repeat(auto-fill, minmax(150px, 1fr));
  gap: 12px;
  max-height: 300px;
  overflow-y: auto;
  padding: 10px;
  border-radius: 12px;
  background: color-mix(in srgb, var(--app-card-surface) 90%, transparent);
  border: 1px solid var(--app-card-border);
  backdrop-filter: blur(var(--app-card-backdrop-blur));
  -webkit-backdrop-filter: blur(var(--app-card-backdrop-blur));
}

.steam-image-item {
  aspect-ratio: 2/3;
  border-radius: 6px;
  overflow: hidden;
  cursor: pointer;
  border: 2px solid var(--app-card-border);
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
  background: color-mix(in srgb, rgba(var(--primary-6), 0.08) 60%, var(--app-card-surface));
  border: 1px solid var(--app-card-border);
  backdrop-filter: blur(var(--app-card-backdrop-blur));
  -webkit-backdrop-filter: blur(var(--app-card-backdrop-blur));
  color: var(--color-text-2);
  font-size: 12px;
}

.steam-game-info {
  display: flex;
  align-items: center;
  gap: 8px;
  padding: 8px;
  background: color-mix(in srgb, var(--app-card-surface) 88%, transparent);
  border: 1px solid var(--app-card-border);
  border-radius: 8px;
  backdrop-filter: blur(var(--app-card-backdrop-blur));
  -webkit-backdrop-filter: blur(var(--app-card-backdrop-blur));
}

.steam-game-info img {
  width: 40px;
  height: 21px;
  object-fit: cover;
  border-radius: 3px;
}

.steam-game-info span {
  font-size: 14px;
  font-weight: 700;
  color: var(--color-text-1);
}

.steam-screenshots-grid {
  display: grid;
  grid-template-columns: repeat(auto-fill, minmax(120px, 1fr));
  gap: 8px;
  padding: 10px;
  border-radius: 12px;
  background: color-mix(in srgb, var(--app-card-surface) 90%, transparent);
  border: 1px solid var(--app-card-border);
  backdrop-filter: blur(var(--app-card-backdrop-blur));
  -webkit-backdrop-filter: blur(var(--app-card-backdrop-blur));
}

.steam-screenshots-empty {
  padding: 20px 0;
  border-radius: 8px;
  background: color-mix(in srgb, var(--app-card-surface) 88%, transparent);
  border: 1px solid var(--app-card-border);
  backdrop-filter: blur(var(--app-card-backdrop-blur));
  -webkit-backdrop-filter: blur(var(--app-card-backdrop-blur));
}

.steam-screenshot-item {
  aspect-ratio: 16/9;
  border-radius: 6px;
  overflow: hidden;
  cursor: pointer;
  position: relative;
  border: 2px solid var(--app-card-border);
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
