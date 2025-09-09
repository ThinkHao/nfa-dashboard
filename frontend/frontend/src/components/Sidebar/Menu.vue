<script setup lang="ts">
import { useRouter, useRoute } from 'vue-router'
import { computed } from 'vue'
import { useAuthStore } from '@/stores/auth'

const router = useRouter()
const route = useRoute()
const auth = useAuthStore()

function hasMenuAccess(r: any): boolean {
  const required = (r.meta?.permissions as string[] | undefined) || []
  if (!required.length) return true
  return auth.hasAnyPermission(required)
}

// 根据 path 前缀对页面进行分组，支持 meta.hideInMenu 与 meta.order
const menus = computed(() => {
  const raws = router.getRoutes()
    .filter(r => !!r.meta?.title && !r.meta?.public && r.path !== '/403' && r.path !== '/login')
    .filter(hasMenuAccess)

  type Item = { title: string; path: string; order: number; hideInMenu: boolean; icon?: string }
  const items: Item[] = raws.map(r => ({
    title: r.meta?.title as string,
    path: r.path,
    order: (r.meta as any)?.order ?? 0,
    hideInMenu: !!(r.meta as any)?.hideInMenu,
    icon: (r.meta as any)?.icon as string | undefined,
  })).filter(i => !i.hideInMenu)

  // 排序
  items.sort((a, b) => a.order - b.order)

  const groups: Record<string, { title: string; children: Item[] }> = {}
  const ensure = (key: string, title: string) => (groups[key] ||= { title, children: [] })

  const settlement = ensure('settlement', '结算系统')
  const system = ensure('system', '系统管理')
  const topLevel: Item[] = []

  for (const r of items) {
    // 首页在模板中单独渲染，这里跳过以避免重复
    if (r.path === '/') { continue }
    const seg = r.path.split('/')[1]
    if (seg === 'settlement') {
      settlement.children.push(r)
    } else if (seg === 'system') {
      system.children.push(r)
    } else {
      topLevel.push(r)
    }
  }

  // 各组内部再按 order 排序
  settlement.children.sort((a, b) => a.order - b.order)
  system.children.sort((a, b) => a.order - b.order)
  topLevel.sort((a, b) => a.order - b.order)

  // 过滤空的分组
  const groupList = [settlement, system].filter(g => g.children.length)
  return { topLevel, groupList }
})
</script>

<template>
  <el-menu
    :default-active="$route.path"
    router
    background-color="transparent"
    text-color="#e5e7eb"
    active-text-color="#fff"
    class="menu"
  >
    <el-menu-item index="/"><span class="menu-icon">🏠</span><span>首页</span></el-menu-item>
    <template v-for="item in menus.topLevel" :key="item.path">
      <el-menu-item :index="item.path">
        <span v-if="(item as any).icon" class="menu-icon">{{ (item as any).icon }}</span>
        <span>{{ item.title }}</span>
      </el-menu-item>
    </template>

    <template v-for="g in menus.groupList" :key="g.title">
      <el-sub-menu :index="g.title">
        <template #title>{{ g.title }}</template>
        <el-menu-item v-for="c in g.children" :key="c.path" :index="c.path">
          <span v-if="(c as any).icon" class="menu-icon">{{ (c as any).icon }}</span>
          <span>{{ c.title }}</span>
        </el-menu-item>
      </el-sub-menu>
    </template>
  </el-menu>
</template>

<style scoped>
.menu { border-right: none; }
.menu-icon { display: inline-flex; width: 18px; margin-right: 6px; align-items: center; justify-content: center; }
</style>
