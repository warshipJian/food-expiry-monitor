const api = require('../../utils/api')
const { dateOnly, relativeExpiry } = require('../../utils/date')

Page({
  data: { food: null, id: '', reminderDays: 7, savingReminder: false },
  onLoad(options) { this.setData({ id: options.id }); this.load() },
  load() {
    api.request('GET', `/v1/foods/${this.data.id}`).then(food => this.setData({ food: { ...food, expiryLabel: dateOnly(food.expiryDate), relative: relativeExpiry(food.expiryDate) } })).catch(err => wx.showToast({ title: err.message, icon: 'none' }))
  },
  chooseReminder(e) { this.setData({ reminderDays: Number(e.detail.value) }) },
  saveReminder() {
    this.setData({ savingReminder: true })
    api.request('POST', `/v1/foods/${this.data.id}/reminder`, { daysBefore: this.data.reminderDays })
      .then(() => wx.showToast({ title: '提醒已设置', icon: 'success' }))
      .catch(err => wx.showToast({ title: err.message, icon: 'none' }))
      .finally(() => this.setData({ savingReminder: false }))
  },
  changeStatus(status) {
    wx.showModal({ title: status === 'consume' ? '确认已吃完？' : '确认丢弃？', content: '这件食品会从当前库存中移除。', success: result => {
      if (!result.confirm) return
      api.request('POST', `/v1/foods/${this.data.id}/${status}`).then(() => { wx.showToast({ title: '已更新', icon: 'success' }); setTimeout(() => wx.navigateBack(), 350) }).catch(err => wx.showToast({ title: err.message, icon: 'none' }))
    } })
  },
  consume() { this.changeStatus('consume') },
  discard() { this.changeStatus('discard') }
})
