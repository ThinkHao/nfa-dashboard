function isVisibleElement(el: HTMLElement): boolean {
  let current: HTMLElement | null = el
  while (current) {
    const style = getComputedStyle(current)
    if (style.display === 'none' || style.visibility === 'hidden') return false
    current = current.parentElement
  }
  return true
}

function hasActiveModalContent(root: ParentNode): boolean {
  // Element Plus puts aria-modal on the overlay's dialog wrapper
  // (.el-overlay-dialog), not on the inner .el-dialog element.
  const selectors = [
    '[role="dialog"][aria-modal="true"]',
    // Image viewer markup has varied between Element Plus versions.
    '.el-image-viewer__wrapper',
  ]
  return selectors.some((selector) =>
    Array.from(root.querySelectorAll(selector)).some((node) =>
      isVisibleElement(node as HTMLElement),
    ),
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
