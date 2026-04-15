import { onBeforeUnmount, onMounted } from 'vue'
import { useRoute } from 'vue-router'
import { listenPageRefresh } from '@/utils/pageRefresh'

export function usePageRefresh(onRefresh: () => void | Promise<void>) {
  const route = useRoute()
  let off: (() => void) | null = null

  onMounted(() => {
    off = listenPageRefresh((detail) => {
      if (!detail.path || detail.path === route.path) {
        onRefresh()
      }
    })
  })

  onBeforeUnmount(() => {
    off?.()
    off = null
  })
}
