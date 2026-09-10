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
        <div class="el-overlay-dialog" role="dialog" aria-modal="true">
          <div class="el-dialog" style="display: block; width: 300px; height: 200px;"></div>
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

  it('cleans an overlay whose dialog is already hidden', () => {
    document.body.innerHTML = `
      <div class="el-overlay" style="display: none; position: fixed; inset: 0;">
        <div class="el-overlay-dialog" role="dialog" aria-modal="true">
          <div class="el-dialog"></div>
        </div>
      </div>
    `
    document.body.classList.add('el-popup-parent--hidden')

    const cleaned = cleanupStaleElementOverlays(document)

    const overlay = document.querySelector('.el-overlay') as HTMLElement
    expect(cleaned).toBe(true)
    expect(overlay.style.display).toBe('none')
    expect(document.body.classList.contains('el-popup-parent--hidden')).toBe(false)
  })

  it('keeps visible drawer and message box overlays', () => {
    document.body.innerHTML = `
      <div class="el-overlay" style="display: block;"><div class="el-drawer" role="dialog" aria-modal="true"></div></div>
      <div class="el-overlay" style="display: block;"><div class="el-overlay-message-box" role="dialog" aria-modal="true"></div></div>
    `
    document.body.classList.add('el-popup-parent--hidden')

    const cleaned = cleanupStaleElementOverlays(document)

    expect(cleaned).toBe(false)
    expect(document.querySelectorAll('.el-overlay[style*="display: block"]').length).toBe(2)
    expect(document.body.classList.contains('el-popup-parent--hidden')).toBe(true)
  })
})
