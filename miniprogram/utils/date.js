function dateOnly(value) { return value ? value.slice(0, 10) : '' }
function daysUntil(value) {
  const date = new Date(`${dateOnly(value)}T00:00:00`)
  const today = new Date(); today.setHours(0, 0, 0, 0)
  return Math.round((date - today) / 86400000)
}
function relativeExpiry(value) {
  const days = daysUntil(value)
  if (days < 0) return `已过期 ${Math.abs(days)} 天`
  if (days === 0) return '今天到期'
  return `还剩 ${days} 天`
}
function todayPlus(days) {
  const date = new Date(); date.setDate(date.getDate() + days)
  const month = `${date.getMonth() + 1}`.padStart(2, '0')
  return `${date.getFullYear()}-${month}-${`${date.getDate()}`.padStart(2, '0')}`
}
module.exports = { dateOnly, daysUntil, relativeExpiry, todayPlus }
