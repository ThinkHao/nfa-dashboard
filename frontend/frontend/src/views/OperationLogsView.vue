<template>
  <div class="oplog-view">
    <el-card class="box-card" shadow="never">
      <template #header>
        <div class="card-header">
          <span>操作日志筛选</span>
          <div>
            <el-button type="primary" :loading="loading" @click="onSearch">查询</el-button>
            <el-button @click="onReset">重置</el-button>
            <el-button type="success" :loading="exporting" @click="onExport">导出 CSV</el-button>
          </div>
        </div>
      </template>

      <el-form :inline="true" :model="query" label-width="90px" class="filter-form">
        <el-form-item label="用户ID">
          <el-input v-model.number="query.user_id" placeholder="如 1" clearable style="width: 160px" />
        </el-form-item>
        <el-form-item label="方法">
          <el-select v-model="query.method" placeholder="全部" clearable style="width: 140px">
            <el-option v-for="m in methods" :key="m" :label="m" :value="m" />
          </el-select>
        </el-form-item>
        <el-form-item label="路径">
          <el-input v-model="query.path" placeholder="包含 ..." clearable style="width: 260px" />
        </el-form-item>
        <el-form-item label="状态码">
          <el-input v-model.number="query.status_code" placeholder="如 200" clearable style="width: 140px" />
        </el-form-item>
        <el-form-item label="成功">
          <el-select v-model="query.success" placeholder="全部" clearable style="width: 140px">
            <el-option label="成功" :value="1" />
            <el-option label="失败" :value="0" />
          </el-select>
        </el-form-item>
        <el-form-item label="时间范围">
          <el-date-picker
            v-model="queryRange"
            type="datetimerange"
            range-separator="至"
            start-placeholder="开始时间"
            end-placeholder="结束时间"
            value-format="x"
          />
        </el-form-item>
        <el-form-item label="关键词">
          <el-input v-model="query.keyword" placeholder="匹配 path / error" clearable style="width: 260px" />
        </el-form-item>
      </el-form>
    </el-card>

    <el-card class="box-card" shadow="never" style="margin-top: 16px">
      <template #header>
        <div class="card-header">
          <span>日志列表</span>
        </div>
      </template>

      <el-table :data="items" border stripe height="600px" v-loading="loading">
        <el-table-column prop="created_at" label="时间" width="180">
          <template #default="{ row }">
            <span>{{ formatTime(row.created_at) }}</span>
          </template>
        </el-table-column>
        <el-table-column prop="user_id" label="用户ID" width="100" />
        <el-table-column prop="method" label="方法" width="90" />
        <el-table-column prop="path" label="路径" min-width="260" show-overflow-tooltip />
        <el-table-column prop="status_code" label="状态码" width="100" />
        <el-table-column prop="success" label="成功" width="90">
          <template #default="{ row }">
            <el-tag :type="row.success === 1 ? 'success' : 'danger'">{{ row.success === 1 ? '是' : '否' }}</el-tag>
          </template>
        </el-table-column>
        <el-table-column prop="latency_ms" label="耗时(ms)" width="110" />
        <el-table-column prop="ip" label="IP" width="140" />
        <el-table-column prop="error_message" label="错误信息" min-width="240" show-overflow-tooltip />
      </el-table>

      <div class="pagination">
        <el-pagination
          background
          layout="prev, pager, next, sizes, total"
          :total="total"
          :page-size="pageSize"
          :current-page="page"
          :page-sizes="[10,20,50,100]"
          @size-change="onPageSizeChange"
          @current-change="onPageChange"
        />
      </div>
    </el-card>
  </div>
</template>

<script setup lang="ts">
import { onMounted, reactive, ref, watch } from 'vue'
import { ElMessage } from 'element-plus'
import api from '@/api'
import type { OperationLog, PaginatedData } from '@/types/api'

const methods = ['GET','POST','PUT','DELETE','PATCH','OPTIONS','HEAD']

const loading = ref(false)
const exporting = ref(false)
const items = ref<OperationLog[]>([])
const total = ref(0)
const page = ref(1)
const pageSize = ref(10)

// 查询参数
const query = reactive<{ 
  user_id?: number
  method?: string
  path?: string
  status_code?: number
  success?: number
  keyword?: string
}>({})

// 时间范围（使用时间戳毫秒）
const queryRange = ref<[string, string] | null>(null)

function formatTime(ts: string | Date) {
  try {
    const d = new Date(ts)
    return isNaN(d.getTime()) ? String(ts) : d.toLocaleString()
  } catch {
    return String(ts)
  }
}

function buildParams() {
  const params: any = {
    page: page.value,
    page_size: pageSize.value
  }
  if (query.user_id) params.user_id = query.user_id
  if (query.method) params.method = query.method
  if (query.path) params.path = query.path
  if (query.status_code) params.status_code = query.status_code
  if (query.success !== undefined) params.success = query.success
  if (query.keyword) params.keyword = query.keyword
  if (queryRange.value && queryRange.value.length === 2) {
    const [startMs, endMs] = queryRange.value
    const startAt = new Date(Number(startMs)).toISOString()
    const endAt = new Date(Number(endMs)).toISOString()
    params.start_at = startAt
    params.end_at = endAt
  }
  return params
}

async function fetchData() {
  loading.value = true
  try {
    const res = await api.operationLogs.list(buildParams())
    items.value = res.items || []
    total.value = res.total || 0
  } catch (e: any) {
    ElMessage.error(e?.response?.data?.message || e?.message || '加载失败')
  } finally {
    loading.value = false
  }
}

function onSearch() {
  page.value = 1
  fetchData()
}

function onReset() {
  Object.assign(query, { user_id: undefined, method: undefined, path: undefined, status_code: undefined, success: undefined, keyword: undefined })
  queryRange.value = null
  page.value = 1
  pageSize.value = 10
  fetchData()
}

function onPageChange(p: number) { page.value = p; fetchData() }
function onPageSizeChange(ps: number) { pageSize.value = ps; page.value = 1; fetchData() }

watch([page, pageSize], () => { /* 留空，交由回调触发 */ })

onMounted(fetchData)

function csvEscape(val: any): string {
  if (val === null || val === undefined) return ''
  let s = String(val)
  if (s.includes('"')) s = s.replace(/"/g, '""')
  if (s.search(/[",\n]/) >= 0) s = `"${s}"`
  return s
}

function downloadBlob(blob: Blob, filename: string) {
  const url = URL.createObjectURL(blob)
  const a = document.createElement('a')
  a.href = url
  a.download = filename
  document.body.appendChild(a)
  a.click()
  document.body.removeChild(a)
  URL.revokeObjectURL(url)
}

async function onExport() {
  exporting.value = true
  const ts = new Date().toISOString().replace(/[:T]/g, '-').split('.')[0]
  try {
    // 优先后端导出（服务端分页流式生成）
    const exportParams = (() => {
      const p = buildParams()
      // 后端内部分页，无需显式 page/page_size
      delete p.page
      delete p.page_size
      return p
    })()
    const blob = await api.operationLogs.export(exportParams)
    // 若后端未包含 BOM，这里不强制添加，尊重服务端输出
    downloadBlob(blob, `operation-logs-${ts}.csv`)
    ElMessage.success('导出成功')
  } catch (e: any) {
    // 回退至前端分页导出（保留原实现，带导出上限）
    try {
      const header = ['时间','用户ID','方法','路径','状态码','成功','耗时(ms)','IP','错误信息']
      const rows: string[] = []
      const paramsBase = buildParams()
      let p = 1
      const maxExport = 2000 // 保护上限
      let exported = 0
      const pageSizeExport = 500
      while (exported < maxExport) {
        const res: PaginatedData<OperationLog> = await api.operationLogs.list({
          ...paramsBase,
          page: p,
          page_size: pageSizeExport,
        })
        for (const r of res.items) {
          rows.push([
            csvEscape(formatTime(r.created_at)),
            csvEscape(r.user_id ?? ''),
            csvEscape(r.method),
            csvEscape(r.path),
            csvEscape(r.status_code),
            csvEscape(r.success === 1 ? '是' : '否'),
            csvEscape(r.latency_ms ?? ''),
            csvEscape(r.ip ?? ''),
            csvEscape(r.error_message ?? ''),
          ].join(','))
          exported++
          if (exported >= maxExport) break
        }
        const totalRows = res.total || 0
        if (p * pageSizeExport >= totalRows) break
        if (exported >= maxExport) break
        p++
      }
      const content = ['\uFEFF' + header.join(','), ...rows].join('\n')
      const blob = new Blob([content], { type: 'text/csv;charset=utf-8;' })
      downloadBlob(blob, `operation-logs-${ts}.csv`)
      ElMessage.success(`导出成功：${exported} 条（最多导出 ${maxExport} 条）`)
    } catch (e2: any) {
      ElMessage.error(e2?.response?.data?.message || e2?.message || '导出失败')
    }
  } finally {
    exporting.value = false
  }
}
</script>

<style scoped>
.oplog-view { padding: 20px; }
.box-card { margin-bottom: 12px; }
.card-header { display: flex; justify-content: space-between; align-items: center; }
.filter-form { row-gap: 8px; }
.pagination { display: flex; justify-content: flex-end; margin-top: 12px; }
</style>
