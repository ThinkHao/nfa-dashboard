<script setup lang="ts">
import { ref, reactive, onMounted } from 'vue'
import { useRouter } from 'vue-router'
import api from '../api'
import PageHeader from '@/components/ui/PageHeader.vue'
import FilterPanel from '@/components/ui/FilterPanel.vue'
import SectionCard from '@/components/ui/SectionCard.vue'
import QueryActionButton from '@/components/ui/QueryActionButton.vue'
import { useCancelableQuery, isAbortError } from '@/composables/useCancelableQuery'
import { usePageRefresh } from '@/composables/usePageRefresh'
import { sanitizeScopeOptionValues } from '@/utils/scope-options'
import { 
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
const queryCtl = useCancelableQuery()

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
    await queryCtl.run((signal) => loadSchools(signal), { showCancelMessage: false })
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
async function loadSchools(signal?: AbortSignal) {
  try {
    loading.value = true
    
    const params = {
      ...queryForm,
      limit: pageSize.value,
      offset: (currentPage.value - 1) * pageSize.value
    }
    
    const res = await (api as any).v2.getSchools(params, { signal }) as any
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
    if (isAbortError(error)) return
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
  queryCtl.run((signal) => loadSchools(signal), { toggleIfRunning: true })
}

// 当选择省份变化时重置学校名称
function handleRegionChange() {
  queryForm.school_name = ''
  // 基于地区/运营商重新加载学校并刷新选项
  queryCtl.run((signal) => loadSchools(signal), { showCancelMessage: false }).then(() => computeRegionCpOptions())
}

// 当选择运营商变化时重置学校名称
function handleCPChange() {
  queryForm.school_name = ''
  // 基于地区/运营商重新加载学校并刷新选项
  queryCtl.run((signal) => loadSchools(signal), { showCancelMessage: false }).then(() => computeRegionCpOptions())
}

// 重置按钮点击事件
function handleReset() {
  queryForm.school_name = ''
  queryForm.region = ''
  queryForm.cp = ''
  currentPage.value = 1
  queryCtl.run((signal) => loadSchools(signal), { showCancelMessage: false })
}

// 分页变化事件
function handlePageChange(page) {
  currentPage.value = page
  queryCtl.run((signal) => loadSchools(signal), { showCancelMessage: false })
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
    regions.value = sanitizeScopeOptionValues(Array.isArray(r) ? r : []).sort()
  } catch {
    regions.value = []
  }
  try {
    const c = await (api as any).v2.getCPs()
    cps.value = sanitizeScopeOptionValues(Array.isArray(c) ? c : []).sort()
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

usePageRefresh(() => {
  queryCtl.run((signal) => loadSchools(signal), { showCancelMessage: false })
})
</script>

<template>
  <div class="page-container">
    <PageHeader title="学校管理" description="按地区、CP、学校名称筛选并查看流量入口。" />
    
    <!-- 查询表单 -->
    <FilterPanel>
      <ElForm :model="queryForm" label-width="80px" inline class="filter-form">
        <ElFormItem label="地区">
          <ElSelect v-model="queryForm.region" placeholder="选择地区" clearable @change="handleRegionChange" class="field-sm">
            <ElOption 
              v-for="region in regions" 
              :key="region" 
              :label="region" 
              :value="region" 
            />
          </ElSelect>
        </ElFormItem>
        
        <ElFormItem label="CP">
          <ElSelect v-model="queryForm.cp" placeholder="选择 CP" clearable @change="handleCPChange" class="field-sm">
            <ElOption 
              v-for="cp in cps" 
              :key="cp" 
              :label="cp" 
              :value="cp" 
            />
          </ElSelect>
        </ElFormItem>
        
        <ElFormItem label="学校名称">
          <ElInput v-model="queryForm.school_name" placeholder="输入学校名称" clearable class="field-sm" />
        </ElFormItem>
        
        <ElFormItem>
          <QueryActionButton :running="queryCtl.running.value" @trigger="handleQuery" />
          <ElButton @click="handleReset">重置</ElButton>
        </ElFormItem>
      </ElForm>
    </FilterPanel>
    
    <!-- 学校数据表格 -->
    <SectionCard title="学校列表">
      <ElTable :data="schools" border stripe v-loading="loading">
        <ElTableColumn prop="school_id" label="学校ID" width="100" />
        <ElTableColumn prop="school_name" label="学校名称" />
        <ElTableColumn prop="region" label="地区" />
        <ElTableColumn prop="cp" label="CP" />
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
    </SectionCard>
  </div>
</template>

<style scoped>
.pagination-container {
  margin-top: 1rem;
  display: flex;
  justify-content: flex-end;
}

.filter-form { row-gap: var(--form-item-gap); }

.field-sm:deep(.el-select__wrapper),
.field-sm:deep(.el-input__wrapper) {
  width: 180px !important;
}
</style>
