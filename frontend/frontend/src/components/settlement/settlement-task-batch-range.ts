type RangeValue = [string, string] | null

type DailyRangeResult = { dates: string[]; error?: never } | { dates?: never; error: string }
type MonthlyRangeResult = { months: string[]; error?: never } | { months?: never; error: string }

function pad(value: number): string {
  return String(value).padStart(2, '0')
}

function formatDate(date: Date): string {
  return `${date.getFullYear()}-${pad(date.getMonth() + 1)}-${pad(date.getDate())}`
}

function formatMonth(date: Date): string {
  return `${date.getFullYear()}-${pad(date.getMonth() + 1)}`
}

function parseDate(value: string): Date | null {
  const match = String(value || '').match(/^(\d{4})-(\d{2})-(\d{2})/)
  if (!match) return null

  const date = new Date(Number(match[1]), Number(match[2]) - 1, Number(match[3]), 0, 0, 0, 0)
  if (date.getFullYear() !== Number(match[1]) || date.getMonth() !== Number(match[2]) - 1 || date.getDate() !== Number(match[3])) {
    return null
  }
  return date
}

function parseMonth(value: string): Date | null {
  const match = String(value || '').match(/^(\d{4})-(\d{2})/)
  if (!match) return null

  const date = new Date(Number(match[1]), Number(match[2]) - 1, 1, 0, 0, 0, 0)
  if (date.getFullYear() !== Number(match[1]) || date.getMonth() !== Number(match[2]) - 1) {
    return null
  }
  return date
}

export function expandNodeDailyTaskRange(range: RangeValue): DailyRangeResult {
  if (!range || range.length !== 2 || !range[0] || !range[1]) {
    return { error: '请选择完整的任务日期范围' }
  }

  const start = parseDate(range[0])
  const end = parseDate(range[1])
  if (!start || !end) {
    return { error: '任务日期格式错误' }
  }
  if (end.getTime() < start.getTime()) {
    return { error: '结束日期不能早于开始日期' }
  }

  const dates: string[] = []
  const cursor = new Date(start)
  while (cursor.getTime() <= end.getTime()) {
    dates.push(formatDate(cursor))
    if (dates.length > 31) {
      return { error: '节点日95一次最多创建31天任务' }
    }
    cursor.setDate(cursor.getDate() + 1)
  }

  return { dates }
}

export function expandNodeMonthlyTaskRange(range: RangeValue): MonthlyRangeResult {
  if (!range || range.length !== 2 || !range[0] || !range[1]) {
    return { error: '请选择完整的服务月份范围' }
  }

  const start = parseMonth(range[0])
  const end = parseMonth(range[1])
  if (!start || !end) {
    return { error: '服务月份格式错误' }
  }
  if (end.getTime() < start.getTime()) {
    return { error: '结束月份不能早于开始月份' }
  }

  const months: string[] = []
  const cursor = new Date(start)
  while (cursor.getTime() <= end.getTime()) {
    months.push(formatMonth(cursor))
    if (months.length > 12) {
      return { error: '节点月95一次最多创建12个月任务' }
    }
    cursor.setMonth(cursor.getMonth() + 1)
  }

  return { months }
}
