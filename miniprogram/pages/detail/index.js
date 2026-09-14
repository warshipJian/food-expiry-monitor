const api = require('../../utils/api')
const config = require('../../utils/config')
const { dateOnly, relativeExpiry } = require('../../utils/date')

Page({
  data: { food: null, id: '', reminderDays: 7, savingReminder: false, savingExpiry: false, editingExpiry: false },
  onLoad(options) { this.setData({ id: options.id }); this.load() },
  onUnload() { this.cancelExpiryEdit() },
  load() {
    api.request('GET', `/v1/foods/${this.data.id}`).then(food => this.setData({ food: { ...food, expiryLabel: dateOnly(food.expiryDate), relative: relativeExpiry(food.expiryDate) } })).catch(err => wx.showToast({ title: err.message, icon: 'none' }))
  },
  chooseReminder(e) { this.setData({ reminderDays: Number(e.detail.value) }) },
  startExpiryEdit() {
    if (!this.data.food || this.data.food.status !== 'active' || this.data.editingExpiry) return
    this.cancelExpiryEdit()
    this.expiryEditTimer = setTimeout(() => {
      this.expiryEditTimer = null
      this.setData({ editingExpiry: true })
      wx.showToast({ title: '请选择新的到期日期', icon: 'none' })
    }, 3000)
  },
  cancelExpiryEdit() {
    if (this.expiryEditTimer) {
      clearTimeout(this.expiryEditTimer)
      this.expiryEditTimer = null
    }
  },
  exitExpiryEdit() { this.setData({ editingExpiry: false }) },
  correctExpiry(e) {
    const expiryDate = e.detail.value
    const food = this.data.food
    if (this.data.savingExpiry || !food) return
    if (expiryDate === food.expiryLabel) { this.exitExpiryEdit(); return }
    this.setData({ savingExpiry: true })
    api.request('PATCH', `/v1/foods/${this.data.id}`, {
      name: food.name,
      barcode: food.barcode,
      category: food.category,
      storageLocation: food.storageLocation,
      quantity: food.quantity,
      unit: food.unit,
      expiryDate
    }).then(updated => {
      this.setData({ food: { ...updated, expiryLabel: dateOnly(updated.expiryDate), relative: relativeExpiry(updated.expiryDate) }, editingExpiry: false })
      wx.showToast({ title: '到期日已更新', icon: 'success' })
    }).catch(err => wx.showToast({ title: err.message, icon: 'none' })).finally(() => this.setData({ savingExpiry: false }))
  },
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
