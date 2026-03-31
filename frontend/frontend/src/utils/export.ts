export function csvEscapeCell(value: unknown): string {
  if (value == null) return ''
  let s = String(value)
  if (s.includes('"')) s = s.replace(/"/g, '""')
  if (/[",\n]/.test(s)) s = `"${s}"`
  return s
}

export function buildCsvContent(headers: string[], rows: Array<Array<unknown>>): string {
  const headerLine = headers.map(csvEscapeCell).join(',')
  const lines = rows.map((row) => row.map(csvEscapeCell).join(','))
  return ['\uFEFF' + headerLine, ...lines].join('\n')
}

export function formatExportFilename(prefix: string, ext: 'csv' | 'xlsx', now: Date = new Date()): string {
  const yyyy = now.getFullYear()
  const mm = String(now.getMonth() + 1).padStart(2, '0')
  const dd = String(now.getDate()).padStart(2, '0')
  const hh = String(now.getHours()).padStart(2, '0')
  const mi = String(now.getMinutes()).padStart(2, '0')
  const ss = String(now.getSeconds()).padStart(2, '0')
  return `${prefix}_${yyyy}${mm}${dd}_${hh}${mi}${ss}.${ext}`
}

export function triggerBlobDownload(blob: Blob, filename: string): void {
  const url = URL.createObjectURL(blob)
  const a = document.createElement('a')
  a.href = url
  a.download = filename
  document.body.appendChild(a)
  a.click()
  document.body.removeChild(a)
  URL.revokeObjectURL(url)
}

export function downloadCsv(headers: string[], rows: Array<Array<unknown>>, filename: string): void {
  const content = buildCsvContent(headers, rows)
  const blob = new Blob([content], { type: 'text/csv;charset=utf-8;' })
  triggerBlobDownload(blob, filename)
}

