import { describe, expect, it } from 'vitest'
import { cleanupStaleElementOverlays } from '../overlayCleanup'

describe('cleanupStaleElementOverlays', () => {
  it('removes stale overlay and body lock when no modal content is visible', () => {
    document.body.innerHTML = `
      <div class="el-overlay" style="display: block; position: fixed; inset: 0;"></div>
    `
    document.body.classList.add('el-popup-parent--hidden')
    document.body.style.overflow = 'hidden'

    const cleaned = cleanupStaleElementOverlays(document)

    const overlay = document.querySelector('.el-overlay') as HTMLElement
    expect(cleaned).toBe(true)
    expect(overlay.style.display).toBe('none')
    expect(document.body.classList.contains('el-popup-parent--hidden')).toBe(false)
    expect(document.body.style.overflow).toBe('')
  })

  it('keeps overlay when a real dialog is visible', () => {
    document.body.innerHTML = `
      <div class="el-overlay" style="display: block; position: fixed; inset: 0;">
        <div class="el-overlay-dialog">
          <div class="el-dialog" aria-modal="true" style="display: block; width: 300px; height: 200px;"></div>
        </div>
      </div>
    `
    document.body.classList.add('el-popup-parent--hidden')
    document.body.style.overflow = 'hidden'

    const cleaned = cleanupStaleElementOverlays(document)

    const overlay = document.querySelector('.el-overlay') as HTMLElement
    expect(cleaned).toBe(false)
    expect(overlay.style.display).toBe('block')
    expect(document.body.classList.contains('el-popup-parent--hidden')).toBe(true)
  })
})
