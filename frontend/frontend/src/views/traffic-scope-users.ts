import { computed, ref } from 'vue'
import type { PaginatedData, TrafficScopeUserLite } from '@/types/api'

type TrafficScopeUserPageParams = {
  username?: string
  page: number
  page_size: number
}

type TrafficScopeUserPage = Pick<PaginatedData<TrafficScopeUserLite>, 'items' | 'total'>
type TrafficScopeUserListFn = (params: TrafficScopeUserPageParams) => Promise<TrafficScopeUserPage>

export const TRAFFIC_SCOPE_USER_PAGE_SIZE = 50

export function isTrafficScopeUserListNearBottom(
  metrics: { scrollTop: number; clientHeight: number; scrollHeight: number },
  threshold = 24,
) {
  return metrics.scrollHeight - metrics.scrollTop - metrics.clientHeight <= threshold
}

export function useTrafficScopeUsers(
  listUsers: TrafficScopeUserListFn,
  pageSize = TRAFFIC_SCOPE_USER_PAGE_SIZE,
) {
  const users = ref<TrafficScopeUserLite[]>([])
  const loading = ref(false)
  const total = ref(0)
  const query = ref('')
  const hasMore = computed(() => users.value.length < total.value)
  let currentPage = 0
  let requestId = 0

  async function search(keyword = '') {
    const normalizedKeyword = keyword.trim()
    if (normalizedKeyword === query.value && currentPage > 0) return

    query.value = normalizedKeyword
    currentPage = 0
    users.value = []
    total.value = 0
    const currentRequestId = ++requestId
    loading.value = true
    try {
      const result = await listUsers({
        username: normalizedKeyword || undefined,
        page: 1,
        page_size: pageSize,
      })
      if (currentRequestId !== requestId) return
      users.value = result.items || []
      total.value = Math.max(users.value.length, Number(result.total) || 0)
      currentPage = 1
    } catch (error) {
      if (currentRequestId === requestId) throw error
    } finally {
      if (currentRequestId === requestId) loading.value = false
    }
  }

  async function loadMore() {
    if (loading.value || currentPage === 0 || !hasMore.value) return

    const currentRequestId = requestId
    const nextPage = currentPage + 1
    loading.value = true
    try {
      const result = await listUsers({
        username: query.value || undefined,
        page: nextPage,
        page_size: pageSize,
      })
      if (currentRequestId !== requestId) return
      const nextUsers = result.items || []
      if (nextUsers.length === 0) {
        total.value = users.value.length
        return
      }
      users.value = [...users.value, ...nextUsers]
      total.value = Math.max(users.value.length, Number(result.total) || total.value)
      currentPage = nextPage
    } catch (error) {
      if (currentRequestId === requestId) throw error
    } finally {
      if (currentRequestId === requestId) loading.value = false
    }
  }

  return { users, loading, total, query, hasMore, search, loadMore }
}
