<template>
  <div class="settings-view">
    <div class="settings-view__hero page-hero">
      <div class="page-hero__content">
        <h1 class="page-hero__title text-gradient">设置</h1>
        <p class="page-hero__subtitle">管理自定义背景图片与环境配置。</p>
      </div>
    </div>

    <div class="settings-view__sections">
      <div class="settings-top-row">
        <a-card class="settings-card app-glass-surface" :bordered="false">
          <template #title>自定义背景图片</template>
          <p class="settings-card__desc">上传一张背景图片用作网站全局背景</p>

          <div class="bg-preview-row">
            <div class="bg-preview">
              <img v-if="bgPreviewUrl" :src="bgPreviewUrl" class="bg-preview-img" />
              <div v-else class="bg-preview-empty">
                <icon-image :size="36" />
                <span>未设置</span>
              </div>
            </div>
            <div class="bg-actions">
              <a-upload :auto-upload="false" accept="image/*" :show-file-list="false" @change="handleBgUpload">
                <a-button type="primary" :loading="bgUploading">
                  <template #icon><icon-upload /></template>
                  上传背景图片
                </a-button>
              </a-upload>
              <a-button
                v-if="bgExists"
                class="app-text-action-btn"
                type="text"
                status="danger"
                :loading="bgRemoving"
                @click="handleBgRemove"
              >
                <template #icon><icon-delete /></template>
                删除背景
              </a-button>
            </div>
          </div>
        </a-card>
        <a-card class="settings-card settings-card--center app-glass-surface" :bordered="false">
          <template #title>数据维护</template>
          <p class="settings-card__desc">扫描游戏目录并更新数据库中的文件大小信息</p>
          <div class="settings-card__spacer"></div>
          <div class="settings-card__action">
            <a-button type="primary" :loading="isRefreshing" @click="handleRefreshSizes">
              <template #icon><icon-refresh /></template>
              刷新文件大小
            </a-button>
          </div>
        </a-card>
        <a-card class="settings-card settings-card--center app-glass-surface" :bordered="false">
          <template #title>服务管理</template>
          <p class="settings-card__desc">重启后端服务进程</p>
          <div class="settings-card__spacer"></div>
          <div class="settings-card__action">
            <a-button type="primary" :loading="isRestarting" @click="handleRestart">
              <template #icon><icon-refresh /></template>
              重启服务端
            </a-button>
          </div>
        </a-card>
      </div>

      <a-card class="settings-card app-glass-surface" :bordered="false">
        <template #title>常规设置</template>
        <p class="settings-card__desc">修改后需重启服务才能生效</p>

        <a-form :model="configForm" layout="vertical">
          <a-row :gutter="16">
            <a-col v-for="entry in generalEntries" :key="entry.key" :xs="24" :md="12">
              <a-form-item :label="entry.label">
                <a-input-password
                  v-if="entry.key === 'ADMIN_PASSWORD'"
                  v-model="configForm[entry.key]"
                  placeholder="管理员登录密码，必填"
                />
                <a-input-password
                  v-else-if="entry.key === 'STEAMGRIDDB_API_KEY'"
                  v-model="configForm[entry.key]"
                  placeholder="SteamGridDB API Key"
                />
                <a-input
                  v-else-if="entry.key === 'PRIMARY_ROM_ROOT'"
                  v-model="configForm[entry.key]"
                  placeholder="游戏文件根目录，如 /mnt/Game"
                />
                <a-input
                  v-else-if="entry.key === 'VHD_DIFF_ROOT'"
                  v-model="configForm[entry.key]"
                  placeholder="客户端差分盘盘符，如 C:"
                />
                <a-input
                  v-else-if="entry.key === 'PROXY'"
                  v-model="configForm[entry.key]"
                  placeholder="http / https / socks5，留空直连"
                />
                <a-input
                  v-else-if="entry.key === 'AUTH_MAX_FAILS'"
                  v-model="configForm[entry.key]"
                  placeholder="登录失败次数限制，如 5"
                />
                <a-input
                  v-else-if="entry.key === 'AUTH_COOLDOWN'"
                  v-model="configForm[entry.key]"
                  placeholder="限制冷却时间，如 10m"
                />
                <a-input
                  v-else-if="entry.key === 'AUTH_FAIL_WINDOW'"
                  v-model="configForm[entry.key]"
                  placeholder="失败计数时间窗口，如 30m"
                />
                <a-input
                  v-else-if="entry.key === 'AUTH_STATE_TTL'"
                  v-model="configForm[entry.key]"
                  placeholder="登录会话有效期，如 24h"
                />
                <a-select
                  v-else-if="entry.key === 'AUTH_TRACK_BY'"
                  v-model="configForm[entry.key]"
                  placeholder="失败追踪方式"
                >
                  <a-option value="ip">按 IP</a-option>
                  <a-option value="ip_ua">按 IP + User-Agent</a-option>
                </a-select>
                <a-select
                  v-else-if="entry.key === 'APP_ENV'"
                  v-model="configForm[entry.key]"
                  placeholder="运行环境"
                >
                  <a-option value="production">生产</a-option>
                  <a-option value="development">开发</a-option>
                </a-select>
                <a-input v-else v-model="configForm[entry.key]" :placeholder="`请输入${entry.label}`" />
              </a-form-item>
            </a-col>
          </a-row>
          <a-button type="primary" :loading="configSaving" @click="handleSaveConfig">
            保存配置
          </a-button>
        </a-form>
      </a-card>

      <a-card class="settings-card app-glass-surface" :bordered="false">
        <template #title>数据保护</template>
        <p class="settings-card__desc">默认每天备份数据库；未登记素材会移入 data/orphaned-assets 并保留 7 天</p>

        <a-form :model="configForm" layout="vertical">
          <a-row :gutter="16">
            <a-col v-for="entry in backupEntries" :key="entry.key" :xs="24" :md="12">
              <a-form-item :label="entry.label">
                <a-switch
                  v-if="entry.key === 'DB_BACKUP_ENABLED'"
                  v-model="databaseBackupEnabled"
                />
                <a-input
                  v-else-if="entry.key === 'DB_BACKUP_DIR'"
                  v-model="configForm[entry.key]"
                  placeholder="目录路径"
                />
                <a-input
                  v-else-if="entry.key === 'DB_BACKUP_RETENTION_COUNT'"
                  v-model="configForm[entry.key]"
                  inputmode="numeric"
                  placeholder="保留份数，如 5；0 表示不自动清理"
                />
                <a-input
                  v-else
                  v-model="configForm[entry.key]"
                  placeholder="备份间隔，如 24h、72h"
                />
              </a-form-item>
            </a-col>
          </a-row>
          <a-button type="primary" :loading="configSaving" @click="handleSaveConfig">
            保存配置
          </a-button>
        </a-form>
      </a-card>

      <a-card class="settings-card app-glass-surface" :bordered="false">
        <template #title>SMB 相关设置</template>
        <p class="settings-card__desc">配置 SMB 网络共享路径及凭据</p>

        <a-form :model="configForm" layout="vertical">
          <a-row :gutter="16">
            <a-col v-for="entry in smbEntries" :key="entry.key" :xs="24" :md="12">
              <a-form-item :label="entry.label">
                <a-input-password
                  v-if="entry.key === 'SMB_PASSWORD'"
                  v-model="configForm[entry.key]"
                  placeholder="SMB 访问密码"
                />
                <a-textarea
                  v-else-if="entry.key === 'SMB_PATH_MAPPINGS'"
                  v-model="smbPathMappingsDisplay"
                  placeholder="每行一条，如：&#10;/mnt/Game=\\192.168.1.4\Game&#10;/mnt/Gal=\\192.168.1.4\Gal"
                  :auto-size="{ minRows: 3, maxRows: 6 }"
                />
                <a-input
                  v-else-if="entry.key === 'SMB_USERNAME'"
                  v-model="configForm[entry.key]"
                  placeholder="SMB 访问用户名"
                />
                <a-input v-else v-model="configForm[entry.key]" :placeholder="`请输入${entry.label}`" />
              </a-form-item>
            </a-col>
          </a-row>
          <a-button type="primary" :loading="configSaving" @click="handleSaveConfig">
            保存配置
          </a-button>
        </a-form>
      </a-card>
    </div>
  </div>
</template>

<script setup lang="ts">
import { ref, computed, onMounted } from 'vue'
import { settingsService, type EnvEntry } from '@/services/settings.service'
import gamesService from '@/services/games.service'
import { useUiStore } from '@/stores/ui'
import type { FileItem } from '@arco-design/web-vue/es/upload/interfaces'
import {
  IconImage,
  IconUpload,
  IconDelete,
  IconRefresh,
} from '@arco-design/web-vue/es/icon'

const uiStore = useUiStore()

const configEntries = ref<EnvEntry[]>([])
const generalEntries = computed(() =>
  configEntries.value.filter((e) =>
    ['general', 'auth', 'network', 'runtime', 'paths'].includes(e.group)
  )
)
const backupEntries = computed(() => configEntries.value.filter((e) => e.group === 'backup'))
const smbEntries = computed(() => configEntries.value.filter((e) => e.group === 'smb'))
const configForm = ref<Record<string, string>>({})
const configSaving = ref(false)

const databaseBackupEnabled = computed({
  get: () => configForm.value['DB_BACKUP_ENABLED'] === 'true',
  set: (enabled: boolean) => {
    configForm.value['DB_BACKUP_ENABLED'] = String(enabled)
  },
})

const smbPathMappingsDisplay = computed({
  get: () => (configForm.value['SMB_PATH_MAPPINGS'] || '').split(';').filter(Boolean).join('\n'),
  set: (val: string) => {
    configForm.value['SMB_PATH_MAPPINGS'] = val.split('\n').filter(Boolean).join(';')
  },
})

const bgPreviewUrl = ref<string | null>(null)
const bgExists = ref(false)
const bgUploading = ref(false)
const bgRemoving = ref(false)

const CUSTOM_BG_PATH = '/data/bg.jpg'

const resetBgPreview = () => {
  bgPreviewUrl.value = `${CUSTOM_BG_PATH}?t=${Date.now()}`
}

const loadConfig = async () => {
  try {
    const entries = await settingsService.getConfig()
    configEntries.value = entries
    const form: Record<string, string> = {}
    for (const e of entries) {
      form[e.key] = e.value
    }
    configForm.value = form
  } catch {
    uiStore.addAlert('加载配置失败', 'error')
  }
}

const checkBgExists = async () => {
  try {
    const resp = await fetch(CUSTOM_BG_PATH, { method: 'HEAD' })
    bgExists.value = resp.ok
    if (resp.ok) {
      resetBgPreview()
    }
  } catch {
    bgExists.value = false
  }
}

const handleBgUpload = async (_list: FileItem[], fileItem: FileItem) => {
  const file = fileItem?.file
  if (!file) return

  bgUploading.value = true
  try {
    await settingsService.uploadBackground(file)
    uiStore.addAlert('背景图片已上传', 'success')
    bgExists.value = true
    resetBgPreview()
    await uiStore.refreshSharedBackgroundAvailability()
  } catch {
    uiStore.addAlert('上传背景图片失败', 'error')
  } finally {
    bgUploading.value = false
  }
}

const handleBgRemove = async () => {
  bgRemoving.value = true
  try {
    await settingsService.removeBackground()
    uiStore.addAlert('背景图片已删除', 'success')
    bgExists.value = false
    bgPreviewUrl.value = null
    await uiStore.refreshSharedBackgroundAvailability()
  } catch {
    uiStore.addAlert('删除背景图片失败', 'error')
  } finally {
    bgRemoving.value = false
  }
}

const handleSaveConfig = async () => {
  configSaving.value = true
  try {
    const result = await settingsService.updateConfig(configForm.value)
    uiStore.addAlert(result.message, 'success')
  } catch {
    uiStore.addAlert('保存配置失败', 'error')
  } finally {
    configSaving.value = false
  }
}

const isRefreshing = ref(false)

const handleRefreshSizes = async () => {
  if (isRefreshing.value) return
  isRefreshing.value = true
  try {
    const result = await gamesService.refreshFileSizes()
    uiStore.addAlert(
      `刷新完成：更新 ${result.updated} 个文件，失败 ${result.errors} 个`,
      'success'
    )
  } catch {
    uiStore.addAlert('刷新文件大小失败', 'error')
  } finally {
    isRefreshing.value = false
  }
}

const isRestarting = ref(false)

const handleRestart = async () => {
  if (isRestarting.value) return
  isRestarting.value = true
  try {
    const result = await settingsService.restart()
    uiStore.addAlert(result.message, 'success')
  } catch {
    uiStore.addAlert('重启服务失败', 'error')
  } finally {
    isRestarting.value = false
  }
}

onMounted(() => {
  loadConfig()
  checkBgExists()
})
</script>

<style scoped>
.settings-view {
  padding-bottom: 40px;
}

.settings-view__hero {
  margin-bottom: 20px;
}

.settings-card {
  margin-bottom: 20px;
  height: 100%;
}

.settings-card :deep(.arco-card-body) {
  padding: 16px 12px;
}

.settings-top-row {
  display: grid;
  grid-template-columns: repeat(3, 1fr);
  gap: 16px;
  margin-bottom: 20px;
}

.settings-top-row .settings-card {
  margin-bottom: 0;
}

@media (max-width: 767px) {
  .settings-top-row {
    grid-template-columns: 1fr;
  }
}

.settings-card__desc {
  font-size: 13px;
  color: var(--color-text-3);
  margin: 0 0 16px;
}

.bg-preview-row {
  display: flex;
  gap: 20px;
  align-items: flex-start;
}

.bg-preview {
  width: 200px;
  height: 120px;
  border-radius: 8px;
  overflow: hidden;
  border: 1px solid var(--color-border-2);
  background: var(--color-fill-2);
  flex-shrink: 0;
}

.bg-preview-img {
  width: 100%;
  height: 100%;
  object-fit: cover;
}

.bg-preview-empty {
  width: 100%;
  height: 100%;
  display: flex;
  flex-direction: column;
  align-items: center;
  justify-content: center;
  gap: 6px;
  color: var(--color-text-4);
  font-size: 13px;
}

.bg-actions {
  display: flex;
  flex-direction: column;
  gap: 8px;
  margin-top: 25px;
}

.settings-card--center :deep(.arco-card-body) {
  display: flex;
  flex-direction: column;
  flex: 1;
}

.settings-card--center .settings-card__spacer {
  flex: 1 0 40%;
}

.settings-card--center .settings-card__action {
  display: flex;
  justify-content: center;
  margin-top: 50px;
}

@media (max-width: 768px) {
  .bg-preview-row {
    flex-direction: column;
  }

  .bg-preview {
    width: 100%;
    height: 160px;
  }
}
</style>
