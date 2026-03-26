<script setup lang="ts">
import { useRouter, useRoute } from 'vue-router'
import { computed } from 'vue'
import { useAuthStore } from '@/stores/auth'
import * as Icons from '@element-plus/icons-vue'

const router = useRouter()
const route = useRoute()
const auth = useAuthStore()

const fallbackIcon = Icons.Menu

function resolveIcon(iconName?: string) {
  if (!iconName) return fallbackIcon
  const pack = Icons as Record<string, any>
  return pack[iconName] || fallbackIcon
}

function hasMenuAccess(r: any): boolean {
  const required = (r.meta?.permissions as string[] | undefined) || []
  if (!required.length) return true
  return auth.hasAnyPermission(required)
}

const menus = computed(() => {
  const raws = router.getRoutes()
    .filter(r => !!r.meta?.title && !r.meta?.public && r.path !== '/403' && r.path !== '/login')
    .filter(hasMenuAccess)

  type Item = { title: string; path: string; order: number; hideInMenu: boolean; icon?: string; group?: string }
  const items: Item[] = raws.map(r => ({
    title: r.meta?.title as string,
    path: r.path,
    order: (r.meta as any)?.order ?? 0,
    hideInMenu: !!(r.meta as any)?.hideInMenu,
    icon: (r.meta as any)?.icon as string | undefined,
    group: (r.meta as any)?.group as string | undefined,
  })).filter(i => !i.hideInMenu)

  // 排序
  items.sort((a, b) => a.order - b.order)

  const groupDefinitions: Record<string, { title: string; order: number }> = {
    'settlement-dashboard': { title: '结算系统', order: 28 },
    'settlement-config': { title: '结算系统配置', order: 29 },
    system: { title: '系统管理', order: 80 },
  }

  const groups: Record<string, { title: string; order: number; children: Item[] }> = {}
  const topLevel: Item[] = []

  for (const item of items) {
    if (item.path === '/') continue

    const fallbackGroup = (() => {
      const seg = item.path.split('/')[1]
      if (seg === 'system') return 'system'
      return undefined
    })()

    const groupKey = item.group || fallbackGroup
    if (!groupKey) {
      topLevel.push(item)
      continue
    }

    const def = groupDefinitions[groupKey]
    const group = groups[groupKey] || {
      title: def?.title ?? item.title,
      order: def?.order ?? item.order,
      children: [],
    }
    group.children.push(item)
    groups[groupKey] = group
  }

  for (const group of Object.values(groups)) {
    group.children.sort((a, b) => a.order - b.order)
  }

  topLevel.sort((a, b) => a.order - b.order)

  const groupList = Object.values(groups)
    .filter(g => g.children.length)
    .sort((a, b) => a.order - b.order)

  return { topLevel, groupList }
})
</script>

<template>
  <el-menu
    :default-active="route.path"
    router
    class="menu"
  >
    <el-menu-item index="/">
      <el-icon class="menu-icon"><component :is="resolveIcon('House')" /></el-icon>
      <span>首页</span>
    </el-menu-item>
    <template v-for="item in menus.topLevel" :key="item.path">
      <el-menu-item :index="item.path">
        <el-icon class="menu-icon"><component :is="resolveIcon((item as any).icon)" /></el-icon>
        <span>{{ item.title }}</span>
      </el-menu-item>
    </template>

    <template v-for="g in menus.groupList" :key="g.title">
      <el-sub-menu :index="g.title">
        <template #title>{{ g.title }}</template>
        <el-menu-item v-for="c in g.children" :key="c.path" :index="c.path">
          <el-icon class="menu-icon"><component :is="resolveIcon((c as any).icon)" /></el-icon>
          <span>{{ c.title }}</span>
        </el-menu-item>
      </el-sub-menu>
    </template>
  </el-menu>
</template>

<style scoped>
.menu { border-right: none; }
.menu-icon { display: inline-flex; width: 18px; margin-right: 8px; align-items: center; justify-content: center; }
</style>


