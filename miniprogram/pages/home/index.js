const api = require('../../utils/api')
const config = require('../../utils/config')
const { dateOnly, daysUntil, relativeExpiry } = require('../../utils/date')

Page({
  data: { loading: true, dashboard: { active: 0, expiring: 0, expired: 0 }, foods: [], profile: { nickname: '', avatarUrl: '' } },
  onShow() { this.load() },
  onPullDownRefresh() { this.load(true) },
  load(fromRefresh) {
    this.setData({ loading: true })
    getApp().ensureLogin()
      .then(() => Promise.all([api.request('GET', '/v1/dashboard'), api.request('GET', '/v1/foods?status=active'), api.request('GET', '/v1/profile')]))
      .then(([dashboard, result, profileResult]) => {
        const foods = result.items.map(food => ({ ...food, expiryLabel: dateOnly(food.expiryDate), relative: relativeExpiry(food.expiryDate), urgency: daysUntil(food.expiryDate) <= 3 ? 'urgent' : 'safe' }))
        const profile = { ...profileResult, avatarUrl: profileResult.avatarUrl ? `${config.baseURL}${profileResult.avatarUrl}` : '' }
        this.setData({ dashboard, foods, profile, loading: false })
      })
      .catch(err => { this.setData({ loading: false }); wx.showToast({ title: err.message, icon: 'none' }) })
      .finally(() => { if (fromRefresh) wx.stopPullDownRefresh() })
  },
  addFood() { wx.navigateTo({ url: '/pages/add/index' }) },
  openFood(event) { wx.navigateTo({ url: `/pages/detail/index?id=${event.currentTarget.dataset.id}` }) },
  openProfile() { wx.navigateTo({ url: '/pages/profile/index' }) }
})
