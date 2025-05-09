<script setup lang="ts">
import { ref, reactive, onMounted } from 'vue'
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

// 初始化数据
onMounted(async () => {
  try {
    // 加载下拉选项数据
    await Promise.all([
      loadRegions(),
      loadCPs()
    ])
    
    // 加载学校数据
    await loadSchools()
  } catch (error) {
    console.error('初始化数据失败:', error)
    ElMessage.error('加载数据失败，请刷新页面重试')
  }
})

// 加载地区数据
async function loadRegions() {
  try {
    const res = await api.getRegions()
    if (res.code === 200) {
      regions.value = res.data
    }
  } catch (error) {
    console.error('加载地区数据失败:', error)
  }
}

// 加载运营商数据
async function loadCPs() {
  try {
    const res = await api.getCPs()
    if (res.code === 200) {
      cps.value = res.data
    }
  } catch (error) {
    console.error('加载运营商数据失败:', error)
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
    
    const res = await api.getSchools(params)
    console.log('学校数据原始响应:', res)
    
    if (res.code === 200 && res.data) {
      // 正确的数据结构：data.items
      if (Array.isArray(res.data.items)) {
        schools.value = res.data.items
        total.value = res.data.total || 0
        console.log('加载学校数据成功:', schools.value.length, '所学校')
      } 
      // 兼容旧的数据结构：data.schools
      else if (Array.isArray(res.data.schools)) {
        schools.value = res.data.schools
        total.value = res.data.total || 0
        console.log('加载学校数据成功(旧结构):', schools.value.length, '所学校')
      }
      // 如果数据本身就是数组
      else if (Array.isArray(res.data)) {
        schools.value = res.data
        total.value = res.data.length
        console.log('加载学校数据成功(直接数组):', schools.value.length, '所学校')
      }
      // 如果没有有效数据
      else {
        console.warn('未找到有效的学校数据结构')
        schools.value = []
        total.value = 0
      }
    } else {
      console.warn('学校数据请求失败:', res)
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
}

// 当选择运营商变化时重置学校名称
function handleCPChange() {
  queryForm.school_name = ''
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
</script>

<template>
  <div class="schools-container">
    <h1 class="page-title">学校管理</h1>
    
    <!-- 查询表单 -->
    <ElCard class="query-card">
      <ElForm :model="queryForm" label-width="80px" inline>
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
              @click="$router.push(`/traffic?school_name=${scope.row.school_name}`)"
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

.page-title {
  font-size: 1.8rem;
  margin-bottom: 1.5rem;
  color: var(--dark-color);
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

:deep(.el-form-item) {
  margin-bottom: 18px;
  margin-right: 18px;
}

:deep(.el-select) {
  width: 180px !important;
}

:deep(.el-input) {
  width: 180px !important;
}
</style>
