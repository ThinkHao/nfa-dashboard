function hasActiveModalContent(root: ParentNode): boolean {
  const selectors = [
    '.el-dialog[aria-modal="true"]',
    '.el-drawer[aria-modal="true"]',
    '.el-message-box',
    '.el-image-viewer__wrapper',
  ]
  return selectors.some((selector) =>
    Array.from(root.querySelectorAll(selector)).some((node) => {
      const el = node as HTMLElement
      return getComputedStyle(el).display !== 'none'
    }),
  )
}

export function cleanupStaleElementOverlays(doc: Document = document): boolean {
  const body = doc.body
  if (!body) return false

  if (hasActiveModalContent(doc)) {
    return false
  }

  const overlays = Array.from(doc.querySelectorAll('.el-overlay')) as HTMLElement[]
  const staleOverlays = overlays.filter((overlay) => {
    const style = getComputedStyle(overlay)
    return style.display !== 'none'
  })

  if (staleOverlays.length === 0 && !body.classList.contains('el-popup-parent--hidden')) {
    return false
  }

  staleOverlays.forEach((overlay) => {
    overlay.style.display = 'none'
    overlay.setAttribute('aria-hidden', 'true')
  })
  body.classList.remove('el-popup-parent--hidden')
  body.style.removeProperty('overflow')
  body.style.removeProperty('width')
  return true
}
