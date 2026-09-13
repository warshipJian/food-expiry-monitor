const api = require('../../utils/api')
const { dateOnly, daysUntil, relativeExpiry } = require('../../utils/date')

Page({
  data: { loading: true, dashboard: { active: 0, expiring: 0, expired: 0 }, foods: [] },
  onShow() { this.load() },
  onPullDownRefresh() { this.load(true) },
  load(fromRefresh) {
    this.setData({ loading: true })
    getApp().ensureLogin()
      .then(() => Promise.all([api.request('GET', '/v1/dashboard'), api.request('GET', '/v1/foods?status=active')]))
      .then(([dashboard, result]) => {
        const foods = result.items.map(food => ({ ...food, expiryLabel: dateOnly(food.expiryDate), relative: relativeExpiry(food.expiryDate), urgency: daysUntil(food.expiryDate) <= 3 ? 'urgent' : 'safe' }))
        this.setData({ dashboard, foods, loading: false })
      })
      .catch(err => { this.setData({ loading: false }); wx.showToast({ title: err.message, icon: 'none' }) })
      .finally(() => { if (fromRefresh) wx.stopPullDownRefresh() })
  },
  addFood() { wx.navigateTo({ url: '/pages/add/index' }) },
  openFood(event) { wx.navigateTo({ url: `/pages/detail/index?id=${event.currentTarget.dataset.id}` }) }
})
