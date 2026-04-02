<template>
  <a-sub-menu v-if="hasChildren" :key="item.name">
    <template v-if="item.icon" #icon>
      <component :is="item.icon" />
    </template>
    <template #title>
      {{ item.title }}
    </template>
    <app-navigation-menu-node
      v-for="child in item.children"
      :key="child.name"
      :item="child"
    />
  </a-sub-menu>

  <a-menu-item v-else :key="item.name">
    <template v-if="item.icon" #icon>
      <component :is="item.icon" />
    </template>
    {{ item.title }}
  </a-menu-item>
</template>

<script setup lang="ts">
import { computed } from 'vue'
import type { MenuItem } from '@/hooks/useMenu'

const props = defineProps<{
  item: MenuItem
}>()

const hasChildren = computed(() => (props.item.children?.length ?? 0) > 0)

defineOptions({
  name: 'AppNavigationMenuNode',
})
</script>
