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
          <button class="color-swatch swatch-primary" @click="themeStore.setPrimary('#2563eb')"></button>
          <button class="color-swatch swatch-teal" @click="themeStore.setPrimary('#0f766e')"></button>
          <button class="color-swatch swatch-sky" @click="themeStore.setPrimary('#0369a1')"></button>
          <button class="color-swatch swatch-amber" @click="themeStore.setPrimary('#b45309')"></button>
          <button class="color-swatch swatch-slate" @click="themeStore.setPrimary('#334155')"></button>
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


