<script setup lang="ts">
import { useRoute, useRouter } from 'vue-router'
import { computed } from 'vue'

const route = useRoute()
const router = useRouter()

// 依据 matched 构建面包屑，仅展示有 meta.title 的项
const crumbs = computed(() => {
  const list = route.matched
    .filter(r => !!r.meta?.title)
    .map(r => ({ title: r.meta?.title as string, path: r.path.startsWith('/') ? r.path : `/${r.path}` }))
  // 若首个不是首页，补一个首页
  if (!list.length || list[0].path !== '/') {
    return [{ title: '首页', path: '/' }, ...list]
  }
  return list
})

function onClick(path: string, idx: number) {
  // 最后一个为当前页，不跳转
  if (idx === crumbs.value.length - 1) return
  router.push(path)
}
</script>

<template>
  <el-breadcrumb separator="/" class="app-breadcrumb">
    <el-breadcrumb-item v-for="(c, i) in crumbs" :key="c.path">
      <a class="crumb-link" @click.prevent="onClick(c.path, i)">{{ c.title }}</a>
    </el-breadcrumb-item>
  </el-breadcrumb>
</template>

<style scoped>
.app-breadcrumb { color: var(--text-muted); }
.crumb-link { color: var(--text-muted); cursor: pointer; }
.crumb-link:hover { color: var(--color-primary); }
</style>
