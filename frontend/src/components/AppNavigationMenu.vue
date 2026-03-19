<template>
  <a-menu
    class="app-navigation-menu"
    :selected-keys="[activeKey]"
    :open-keys="openKeys"
    :auto-open-selected="autoOpenSelected"
    :collapsed="collapsed"
    @menu-item-click="handleMenuItemClick"
    @update:open-keys="handleOpenKeysChange"
  >
    <app-navigation-menu-node
      v-for="item in items"
      :key="item.name"
      :item="item"
    />
  </a-menu>
</template>

<script setup lang="ts">
import type { MenuItem } from '@/hooks/useMenu'
import AppNavigationMenuNode from './AppNavigationMenuNode.vue'

withDefaults(defineProps<{
  items: MenuItem[]
  activeKey: string
  openKeys?: string[]
  collapsed?: boolean
  autoOpenSelected?: boolean
}>(), {
  openKeys: () => [],
  collapsed: false,
  autoOpenSelected: true,
})

const emit = defineEmits<{
  navigate: [key: string]
  'update:openKeys': [keys: string[]]
}>()

const handleMenuItemClick = (key: string) => {
  emit('navigate', key)
}

const handleOpenKeysChange = (keys: string[]) => {
  emit('update:openKeys', keys)
}
</script>
