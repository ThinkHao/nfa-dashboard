<script setup lang="ts">
import { useRoute, useRouter } from 'vue-router'
import { computed, ref, onMounted, onBeforeUnmount } from 'vue'
import { useTagsViewStore } from '@/stores/tagsView'

const router = useRouter()
const route = useRoute()
const store = useTagsViewStore()

const tags = computed(() => store.visited)
const activeTab = computed({
  get: () => store.activePath,
  set: (v: string) => store.setActive(v),
})

function onTabClick(name: any) {
  const p = typeof name === 'string' ? name : store.activePath
  if (p && p !== store.activePath) router.push(p)
}
function onTabRemove(name: any) {
  store.remove(name as string)
  if (store.activePath && store.activePath !== name) {
    router.push(store.activePath)
  }
}

// ===== Context menu =====
const menuVisible = ref(false)
const menuX = ref(0)
const menuY = ref(0)
const menuPath = ref<string>('')
function openMenu(e: MouseEvent, path: string) {
  e.preventDefault()
  menuVisible.value = true
  menuX.value = e.clientX
  menuY.value = e.clientY
  menuPath.value = path
}
function hideMenu() { menuVisible.value = false }
function onCloseCurrent() {
  const p = menuPath.value
  hideMenu()
  if (!p) return
  const goPath = (store.activePath === p) ? (store.visited.find(t => t.path !== p)?.path || '/') : store.activePath
  store.remove(p)
  router.push(goPath || '/')
}
function onCloseOthers() {
  const p = menuPath.value || store.activePath
  hideMenu()
  if (!p) return
  store.removeOthers(p)
  router.push(p)
}
function onCloseAll() {
  hideMenu()
  store.removeAll()
  router.push('/')
}
function onRefresh() {
  const p = menuPath.value || store.activePath
  hideMenu()
  if (!p) return
  // 简易刷新：追加时间戳查询参数以触发组件更新
  try {
    const hasQuery = p.includes('?')
    const url = `${p}${hasQuery ? '&' : '?'}_r=${Date.now()}`
    router.replace(url)
  } catch {
    router.replace({ path: route.path, query: { ...route.query, _r: Date.now() } })
  }
}

function onGlobalClick() { if (menuVisible.value) hideMenu() }
onMounted(() => { document.addEventListener('click', onGlobalClick) })
onBeforeUnmount(() => { document.removeEventListener('click', onGlobalClick) })
</script>

<template>
  <div class="tags-view">
    <div class="tags-toolbar">
      <div class="spacer"></div>
      <div class="actions">
        <el-button size="small" text @click="onRefresh">刷新</el-button>
        <el-button size="small" text @click="onCloseOthers">关闭其他</el-button>
        <el-button size="small" text type="danger" @click="onCloseAll">关闭全部</el-button>
      </div>
    </div>
    <el-tabs v-model="activeTab" type="card" closable @tab-remove="onTabRemove" @tab-click="(pane: any) => onTabClick(pane.paneName)">
      <el-tab-pane v-for="t in tags" :key="t.path" :name="t.path">
        <template #label>
          <span @contextmenu="(e: MouseEvent) => openMenu(e, t.path)">{{ t.title }}</span>
        </template>
      </el-tab-pane>
    </el-tabs>
    <div v-if="menuVisible" class="ctx-menu" :style="{ left: menuX + 'px', top: menuY + 'px' }">
      <div class="ctx-item" @click="onRefresh">刷新</div>
      <div class="ctx-item" @click="onCloseCurrent">关闭</div>
      <div class="ctx-item" @click="onCloseOthers">关闭其他</div>
      <div class="ctx-item danger" @click="onCloseAll">关闭全部</div>
    </div>
  </div>
</template>

<style scoped>
.tags-view { padding: 6px 12px 0; background: transparent; }
.tags-toolbar { display: flex; align-items: center; gap: 8px; padding: 0 0 6px; }
.tags-toolbar .spacer { flex: 1; }
.ctx-menu {
  position: fixed;
  z-index: 3000;
  min-width: 120px;
  background: var(--bg-card, #fff);
  border: 1px solid var(--border-color, #ebeef5);
  border-radius: 6px;
  box-shadow: var(--shadow-1, 0 6px 16px rgba(0,0,0,.08));
  overflow: hidden;
}
.ctx-item {
  padding: 8px 12px;
  font-size: 13px;
  cursor: pointer;
}
.ctx-item:hover { background: rgba(64, 158, 255, 0.12); }
.ctx-item.danger:hover { background: rgba(255, 77, 79, 0.12); }
</style>
