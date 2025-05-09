<script setup lang="ts">
import { ref, onMounted } from 'vue'
import { useRouter } from 'vue-router'
import api from '../api'
import { ElCard, ElRow, ElCol, ElStatistic } from 'element-plus'

const router = useRouter()
const summary = ref({
  total_recv: 0,
  total_send: 0,
  total: 0
})
const schoolCount = ref(0)
const loading = ref(true)

onMounted(async () => {
  try {
    loading.value = true
    // 获取流量汇总数据
    const summaryRes = await api.getTrafficSummary()
    if (summaryRes.code === 200) {
      summary.value = summaryRes.data
    }
    
    // 获取学校数量
    const schoolsRes = await api.getSchools({ limit: 1 })
    if (schoolsRes.code === 200) {
      schoolCount.value = schoolsRes.data.total
    }
  } catch (error) {
    console.error('加载首页数据失败:', error)
  } finally {
    loading.value = false
  }
})

// 格式化流量数据，将字节转换为更易读的格式
const formatTraffic = (bytes) => {
  if (bytes === 0) return '0 B'
  const k = 1024
  const sizes = ['B', 'KB', 'MB', 'GB', 'TB', 'PB']
  const i = Math.floor(Math.log(bytes) / Math.log(k))
  return parseFloat((bytes / Math.pow(k, i)).toFixed(2)) + ' ' + sizes[i]
}

const navigateTo = (path) => {
  router.push(path)
}
</script>

<template>
  <div class="home-container">
    <h1 class="page-title">学校流量监控系统</h1>
    <p class="page-description">实时监控学校网络流量，提供数据分析和可视化</p>
    
    <ElRow :gutter="20" class="dashboard-cards">
      <ElCol :span="8">
        <ElCard shadow="hover" @click="navigateTo('/traffic')" class="dashboard-card">
          <ElStatistic :value="formatTraffic(summary.total)" title="总流量" :loading="loading">
            <template #suffix>
              <div class="dashboard-card-icon">
                <i class="el-icon-data-analysis"></i>
              </div>
            </template>
          </ElStatistic>
        </ElCard>
      </ElCol>
      
      <ElCol :span="8">
        <ElCard shadow="hover" @click="navigateTo('/traffic')" class="dashboard-card">
          <ElStatistic :value="formatTraffic(summary.total_recv)" title="总下载流量" :loading="loading">
            <template #suffix>
              <div class="dashboard-card-icon download-icon">
                <i class="el-icon-download"></i>
              </div>
            </template>
          </ElStatistic>
        </ElCard>
      </ElCol>
      
      <ElCol :span="8">
        <ElCard shadow="hover" @click="navigateTo('/traffic')" class="dashboard-card">
          <ElStatistic :value="formatTraffic(summary.total_send)" title="总上传流量" :loading="loading">
            <template #suffix>
              <div class="dashboard-card-icon upload-icon">
                <i class="el-icon-upload"></i>
              </div>
            </template>
          </ElStatistic>
        </ElCard>
      </ElCol>
    </ElRow>
    
    <ElRow :gutter="20" class="feature-cards">
      <ElCol :span="12">
        <ElCard shadow="hover" @click="navigateTo('/traffic')" class="feature-card">
          <div class="feature-content">
            <h3>流量监控</h3>
            <p>实时监控学校网络流量数据，支持按时间、学校、地区和运营商筛选</p>
            <div class="feature-icon">
              <i class="el-icon-monitor"></i>
            </div>
          </div>
        </ElCard>
      </ElCol>
      
      <ElCol :span="12">
        <ElCard shadow="hover" @click="navigateTo('/schools')" class="feature-card">
          <div class="feature-content">
            <h3>学校管理</h3>
            <p>管理监控的学校列表，查看学校详细信息，当前监控 {{ schoolCount }} 所学校</p>
            <div class="feature-icon">
              <i class="el-icon-school"></i>
            </div>
          </div>
        </ElCard>
      </ElCol>
    </ElRow>
  </div>
</template>

<style scoped>
.home-container {
  padding: 2rem 0;
}

.page-title {
  font-size: 2.5rem;
  text-align: center;
  margin-bottom: 1rem;
  color: var(--dark-color);
}

.page-description {
  text-align: center;
  font-size: 1.2rem;
  color: #666;
  margin-bottom: 3rem;
}

.dashboard-cards {
  margin-bottom: 2rem;
}

.dashboard-card {
  cursor: pointer;
  transition: transform 0.3s, box-shadow 0.3s;
  height: 100%;
}

.dashboard-card:hover {
  transform: translateY(-5px);
  box-shadow: 0 10px 20px rgba(0, 0, 0, 0.1);
}

.dashboard-card-icon {
  font-size: 1.5rem;
  margin-left: 0.5rem;
  color: var(--primary-color);
}

.download-icon {
  color: var(--primary-color);
}

.upload-icon {
  color: var(--secondary-color);
}

.feature-cards {
  margin-top: 3rem;
}

.feature-card {
  cursor: pointer;
  transition: transform 0.3s, box-shadow 0.3s;
  height: 100%;
}

.feature-card:hover {
  transform: translateY(-5px);
  box-shadow: 0 10px 20px rgba(0, 0, 0, 0.1);
}

.feature-content {
  display: flex;
  flex-direction: column;
  padding: 1rem;
}

.feature-content h3 {
  font-size: 1.5rem;
  margin-bottom: 1rem;
  color: var(--dark-color);
}

.feature-content p {
  color: #666;
  margin-bottom: 1.5rem;
}

.feature-icon {
  font-size: 2.5rem;
  color: var(--primary-color);
  align-self: flex-end;
}
</style>
