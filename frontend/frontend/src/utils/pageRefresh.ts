const PAGE_REFRESH_EVENT = 'nfa:page-refresh'

type PageRefreshDetail = {
  path?: string
}

export function emitPageRefresh(path?: string) {
  if (typeof window === 'undefined') return
  window.dispatchEvent(new CustomEvent<PageRefreshDetail>(PAGE_REFRESH_EVENT, { detail: { path } }))
}

export function listenPageRefresh(handler: (detail: PageRefreshDetail) => void) {
  if (typeof window === 'undefined') return () => {}
  const listener = (evt: Event) => {
    const detail = (evt as CustomEvent<PageRefreshDetail>).detail || {}
    handler(detail)
  }
  window.addEventListener(PAGE_REFRESH_EVENT, listener)
  return () => window.removeEventListener(PAGE_REFRESH_EVENT, listener)
}
