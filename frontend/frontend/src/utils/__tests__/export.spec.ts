import { describe, expect, it } from 'vitest'
import { buildCsvContent, formatExportFilename } from '@/utils/export'

describe('export utils', () => {
  it('buildCsvContent uses UTF-8 BOM and escapes csv cells', () => {
    const content = buildCsvContent(['学校', '备注'], [['A', '包含,逗号'], ['B', '包含"引号"']])
    expect(content.startsWith('\uFEFF学校,备注\n')).toBe(true)
    expect(content).toContain('A,"包含,逗号"')
    expect(content).toContain('B,"包含""引号"""')
  })

  it('formatExportFilename returns chinese prefix with timestamp', () => {
    const name = formatExportFilename('结算数据明细', 'csv', new Date('2026-03-27T09:00:05'))
    expect(name).toBe('结算数据明细_20260327_090005.csv')
  })
})
