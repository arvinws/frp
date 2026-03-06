import { locale } from '../i18n'

export function formatDistanceToNow(date: Date): string {
  const seconds = Math.floor((Date.now() - date.getTime()) / 1000)
  const absSeconds = Math.abs(seconds)

  const units: Array<{ amount: number; unit: Intl.RelativeTimeFormatUnit }> = [
    { amount: 31536000, unit: 'year' },
    { amount: 2592000, unit: 'month' },
    { amount: 86400, unit: 'day' },
    { amount: 3600, unit: 'hour' },
    { amount: 60, unit: 'minute' },
    { amount: 1, unit: 'second' },
  ]

  const formatter = new Intl.RelativeTimeFormat(locale.value, {
    numeric: 'auto',
  })

  for (const entry of units) {
    if (absSeconds >= entry.amount || entry.unit === 'second') {
      const value = Math.floor(absSeconds / entry.amount)
      return formatter.format(-value, entry.unit)
    }
  }

  return formatter.format(0, 'second')
}

export function formatFileSize(bytes: number): string {
  if (!Number.isFinite(bytes) || bytes < 0) return '0 B'
  if (bytes === 0) return '0 B'
  const k = 1024
  const sizes = ['B', 'KB', 'MB', 'GB', 'TB', 'PB', 'EB', 'ZB', 'YB']
  const i = Math.floor(Math.log(bytes) / Math.log(k))
  // Prevent index out of bounds for extremely large numbers
  const unit = sizes[i] || sizes[sizes.length - 1]
  const val = bytes / Math.pow(k, i)

  return parseFloat(val.toFixed(2)) + ' ' + unit
}
