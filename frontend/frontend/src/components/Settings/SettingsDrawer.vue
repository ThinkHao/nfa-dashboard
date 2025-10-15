<script setup lang="ts">
import { useThemeStore } from '@/stores/theme'

const themeStore = useThemeStore()
</script>

<template>
  <el-popover placement="bottom-end" width="320" trigger="click">
    <template #reference>
      <slot name="reference">
        <el-button link type="primary">外观</el-button>
      </slot>
    </template>
    <div class="settings-panel">
      <div class="row">
        <span>暗色模式</span>
        <el-switch :model-value="themeStore.isDark" @change="(v:boolean) => themeStore.setDark(v)" />
      </div>
      <div class="row">
        <span>组件尺寸</span>
        <el-radio-group :model-value="themeStore.size" @change="(s:any) => themeStore.setSize(s)">
          <el-radio-button label="small">小</el-radio-button>
          <el-radio-button label="default">中</el-radio-button>
          <el-radio-button label="large">大</el-radio-button>
        </el-radio-group>
      </div>
      <div class="row">
        <span>主色</span>
        <div class="colors">
          <button class="color-swatch" style="background:#1890ff" @click="themeStore.setPrimary('#1890ff')"></button>
          <button class="color-swatch" style="background:#409EFF" @click="themeStore.setPrimary('#409EFF')"></button>
          <button class="color-swatch" style="background:#13c2c2" @click="themeStore.setPrimary('#13c2c2')"></button>
          <button class="color-swatch" style="background:#722ed1" @click="themeStore.setPrimary('#722ed1')"></button>
          <button class="color-swatch" style="background:#52c41a" @click="themeStore.setPrimary('#52c41a')"></button>
        </div>
      </div>
    </div>
  </el-popover>
</template>

<style scoped>
.settings-panel { display: flex; flex-direction: column; gap: 10px; }
.settings-panel .row { display: flex; align-items: center; justify-content: space-between; gap: 12px; }
.settings-panel .colors { display: flex; gap: 8px; }
.settings-panel .color-swatch { width: 20px; height: 20px; border-radius: 50%; border: 1px solid rgba(0,0,0,0.15); cursor: pointer; padding: 0; }
</style>
