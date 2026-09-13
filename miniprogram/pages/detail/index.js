const api = require('../../utils/api')
const config = require('../../utils/config')
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
    this.requestSubscription()
      .then(() => api.request('POST', `/v1/foods/${this.data.id}/reminder`, { daysBefore: this.data.reminderDays }))
      .then(() => wx.showToast({ title: '提醒已设置', icon: 'success' }))
      .catch(err => wx.showToast({ title: err.message, icon: 'none' }))
      .finally(() => this.setData({ savingReminder: false }))
  },
  requestSubscription() {
    if (config.environment === 'development') return Promise.resolve()
    return new Promise((resolve, reject) => {
      wx.requestSubscribeMessage({
        tmplIds: [config.subscribeTemplateID],
        success: result => result[config.subscribeTemplateID] === 'accept' ? resolve() : reject(new Error('需允许订阅提醒后才能设置')),
        fail: () => reject(new Error('无法请求提醒授权'))
      })
    })
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
