<script setup lang="ts">
import { ref, reactive, onMounted } from 'vue'
import { useRouter } from 'vue-router'
import api from '../api'
import { 
  ElCard, 
  ElTable, 
  ElTableColumn, 
  ElPagination, 
  ElForm, 
  ElFormItem, 
  ElInput, 
  ElSelect, 
  ElOption, 
  ElButton,
  ElMessage
} from 'element-plus'

// 数据状态
const loading = ref(false)
const schools = ref([])
const regions = ref([])
const cps = ref([])
const total = ref(0)
const currentPage = ref(1)
const pageSize = ref(10)

// 查询表单
const queryForm = reactive({
  school_name: '',
  region: '',
  cp: ''
})

const router = useRouter()

// 初始化数据
onMounted(async () => {
  try {
    await loadRegionCpOptions()
    // 先加载学校数据（基于 v2，按用户过滤）
    await loadSchools()
    // 基于学校数据动态派生地区与运营商选项
    if ((!regions.value || regions.value.length === 0) || (!cps.value || cps.value.length === 0)) {
      computeRegionCpOptions()
    }
  } catch (error) {
    console.error('初始化数据失败:', error)
    ElMessage.error('加载数据失败，请刷新页面重试')
  }
})

// 基于当前 schools 列表动态派生地区/运营商选项（仅限可见院校）
function computeRegionCpOptions() {
  try {
    const rset = new Set<string>()
    const cset = new Set<string>()
    ;(schools.value || []).forEach((s: any) => {
      if (s && typeof s.region === 'string' && s.region && s.region !== 'NULL') rset.add(s.region)
      if (s && typeof s.cp === 'string' && s.cp && s.cp !== 'NULL') cset.add(s.cp)
    })
    regions.value = Array.from(rset).sort()
    cps.value = Array.from(cset).sort()
  } catch (e) {
    console.warn('派生地区/运营商选项失败:', e)
    regions.value = []
    cps.value = []
  }
}

// 加载学校数据
async function loadSchools() {
  try {
    loading.value = true
    
    const params = {
      ...queryForm,
      limit: pageSize.value,
      offset: (currentPage.value - 1) * pageSize.value
    }
    
    const res = await (api as any).v2.getSchools(params) as any
    console.log('学校数据原始响应:', res)
    
    // 已解包：只支持数组或 { items, total }
    if (Array.isArray(res)) {
      schools.value = res
      total.value = res.length
    } else if (res && Array.isArray(res.items)) {
      schools.value = res.items
      total.value = typeof res.total === 'number' ? res.total : res.items.length
    } else {
      console.warn('未找到有效的学校数据结构')
      schools.value = []
      total.value = 0
    }
    
    // 如果没有数据，显示错误提示
    if (schools.value.length === 0) {
      console.warn('未获取到学校数据')
      ElMessage.warning('未能加载学校数据，请检查网络连接')
    }
  } catch (error) {
    console.error('加载学校数据失败:', error)
    ElMessage.error('加载学校数据失败')
    schools.value = []
    total.value = 0
  } finally {
    loading.value = false
  }
}

// 查询按钮点击事件
function handleQuery() {
  currentPage.value = 1
  loadSchools()
}

// 当选择省份变化时重置学校名称
function handleRegionChange() {
  queryForm.school_name = ''
  // 基于地区/运营商重新加载学校并刷新选项
  loadSchools().then(() => computeRegionCpOptions())
}

// 当选择运营商变化时重置学校名称
function handleCPChange() {
  queryForm.school_name = ''
  // 基于地区/运营商重新加载学校并刷新选项
  loadSchools().then(() => computeRegionCpOptions())
}

// 重置按钮点击事件
function handleReset() {
  queryForm.school_name = ''
  queryForm.region = ''
  queryForm.cp = ''
  currentPage.value = 1
  loadSchools()
}

// 分页变化事件
function handlePageChange(page) {
  currentPage.value = page
  loadSchools()
}

// 格式化日期
function formatDate(dateStr) {
  if (!dateStr) return '-'
  const date = new Date(dateStr)
  return date.toLocaleString()
}

async function loadRegionCpOptions() {
  try {
    const r = await (api as any).v2.getRegions()
    regions.value = Array.isArray(r) ? r.filter((v: any) => v && v !== 'NULL').sort() : []
  } catch {
    regions.value = []
  }
  try {
    const c = await (api as any).v2.getCPs()
    cps.value = Array.isArray(c) ? c.filter((v: any) => v && v !== 'NULL').sort() : []
  } catch {
    cps.value = []
  }
}

// 跳转到流量监控并带上过滤参数
function goTraffic(row: any) {
  const query: Record<string, string> = {}
  if (row?.school_name) query.school_name = String(row.school_name)
  if (row?.region && row.region !== 'NULL') query.region = String(row.region)
  if (row?.cp && row.cp !== 'NULL') query.cp = String(row.cp)
  router.push({ path: '/traffic', query })
}
</script>

<template>
  <div class="schools-container">
    <h1 class="page-title">学校管理</h1>
    
    <!-- 查询表单 -->
    <ElCard class="query-card">
      <ElForm :model="queryForm" label-width="80px" inline class="filter-form">
        <ElFormItem label="地区">
          <ElSelect v-model="queryForm.region" placeholder="选择地区" clearable @change="handleRegionChange">
            <ElOption 
              v-for="region in regions" 
              :key="region" 
              :label="region" 
              :value="region" 
            />
          </ElSelect>
        </ElFormItem>
        
        <ElFormItem label="运营商">
          <ElSelect v-model="queryForm.cp" placeholder="选择运营商" clearable @change="handleCPChange">
            <ElOption 
              v-for="cp in cps" 
              :key="cp" 
              :label="cp" 
              :value="cp" 
            />
          </ElSelect>
        </ElFormItem>
        
        <ElFormItem label="学校名称">
          <ElInput v-model="queryForm.school_name" placeholder="输入学校名称" clearable />
        </ElFormItem>
        
        <ElFormItem>
          <ElButton type="primary" @click="handleQuery" :loading="loading">查询</ElButton>
          <ElButton @click="handleReset">重置</ElButton>
        </ElFormItem>
      </ElForm>
    </ElCard>
    
    <!-- 学校数据表格 -->
    <ElCard class="data-card">
      <ElTable :data="schools" border stripe v-loading="loading">
        <ElTableColumn prop="school_id" label="学校ID" width="100" />
        <ElTableColumn prop="school_name" label="学校名称" />
        <ElTableColumn prop="region" label="地区" />
        <ElTableColumn prop="cp" label="运营商" />
        <ElTableColumn prop="hash_count" label="Hash数量" width="100" />
        <ElTableColumn prop="update_time" label="更新时间" width="180">
          <template #default="scope">
            {{ formatDate(scope.row.update_time) }}
          </template>
        </ElTableColumn>
        <ElTableColumn label="操作" width="150">
          <template #default="scope">
            <ElButton 
              type="primary" 
              size="small" 
              @click="goTraffic(scope.row)"
            >
              查看流量
            </ElButton>
          </template>
        </ElTableColumn>
      </ElTable>
      
      <div class="pagination-container">
        <ElPagination
          v-model:current-page="currentPage"
          :page-size="pageSize"
          layout="total, prev, pager, next, jumper"
          :total="total"
          @current-change="handlePageChange"
        />
      </div>
    </ElCard>
  </div>
</template>

<style scoped>
.schools-container {
  padding: 1rem 0;
}

.query-card {
  margin-bottom: 1.5rem;
}

.data-card {
  margin-bottom: 1.5rem;
}

.pagination-container {
  margin-top: 1rem;
  display: flex;
  justify-content: flex-end;
}

.filter-form { row-gap: var(--form-item-gap); }

:deep(.el-select) {
  width: 180px !important;
}

:deep(.el-input) {
  width: 180px !important;
}
</style>
