import { onBeforeUnmount, ref } from 'vue'
import { ElMessage } from 'element-plus'

export function isAbortError(error: unknown): boolean {
  if (!error || typeof error !== 'object') return false
  const e = error as { name?: string; code?: string; message?: string }
  return e.name === 'AbortError' || e.name === 'CanceledError' || e.code === 'ERR_CANCELED' || e.message === 'canceled'
}

type Executor = (signal: AbortSignal) => Promise<void>

type RunOptions = {
  toggleIfRunning?: boolean
  showCancelMessage?: boolean
}

export function useCancelableQuery() {
  const running = ref(false)
  const lastCanceledByUser = ref(false)
  let controller: AbortController | null = null

  function cancel() {
    if (!controller) return false
    lastCanceledByUser.value = true
    controller.abort()
    return true
  }

  async function run(executor: Executor, options: RunOptions = {}): Promise<'completed' | 'canceled'> {
    const { toggleIfRunning = false, showCancelMessage = true } = options
    if (running.value) {
      if (toggleIfRunning) {
        cancel()
        return 'canceled'
      }
      controller?.abort()
    }

    const current = new AbortController()
    controller = current
    running.value = true
    lastCanceledByUser.value = false

    try {
      await executor(current.signal)
      return 'completed'
    } catch (error) {
      if (isAbortError(error)) {
        if (lastCanceledByUser.value && showCancelMessage) {
          ElMessage.info('已取消查询')
        }
        return 'canceled'
      }
      throw error
    } finally {
      if (controller === current) {
        controller = null
        running.value = false
        lastCanceledByUser.value = false
      }
    }
  }

  onBeforeUnmount(() => {
    controller?.abort()
    controller = null
    running.value = false
  })

  return {
    running,
    run,
    cancel,
  }
}
