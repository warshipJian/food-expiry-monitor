const api = require('../../utils/api')
const { dateOnly, daysUntil, relativeExpiry } = require('../../utils/date')

Page({
  data: { loading: true, dashboard: { active: 0, expiring: 0, expired: 0 }, foods: [], consumedFoods: [], showConsumed: false },
  onShow() { this.load() },
  onPullDownRefresh() { this.load(true) },
  load(fromRefresh) {
    this.setData({ loading: true })
    getApp().ensureLogin()
      .then(() => Promise.all([api.request('GET', '/v1/dashboard'), api.request('GET', '/v1/foods?status=active'), api.request('GET', '/v1/foods?status=consumed')]))
      .then(([dashboard, result, consumedResult]) => {
        const foods = result.items.map(food => ({ ...food, expiryLabel: dateOnly(food.expiryDate), relative: relativeExpiry(food.expiryDate), urgency: daysUntil(food.expiryDate) <= 3 ? 'urgent' : 'safe' }))
        const consumedFoods = consumedResult.items.map(food => ({ ...food, expiryLabel: dateOnly(food.expiryDate) }))
        this.setData({ dashboard, foods, consumedFoods, loading: false })
      })
      .catch(err => { this.setData({ loading: false }); wx.showToast({ title: err.message, icon: 'none' }) })
      .finally(() => { if (fromRefresh) wx.stopPullDownRefresh() })
  },
  addFood() { wx.navigateTo({ url: '/pages/add/index' }) },
  openFood(event) { wx.navigateTo({ url: `/pages/detail/index?id=${event.currentTarget.dataset.id}` }) },
  toggleConsumed() { this.setData({ showConsumed: !this.data.showConsumed }) }
})
