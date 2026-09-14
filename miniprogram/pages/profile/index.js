const api = require('../../utils/api')
const config = require('../../utils/config')
const { dateOnly } = require('../../utils/date')

function displayProfile(profile) {
  return { ...profile, avatarUrl: profile.avatarUrl ? `${config.baseURL}${profile.avatarUrl}` : '' }
}

Page({
  data: { profile: { nickname: '', avatarUrl: '' }, family: null, consumedFoods: [] },
  onShow() { this.load() },
  load() {
    Promise.all([api.request('GET', '/v1/profile'), api.request('GET', '/v1/family'), api.request('GET', '/v1/foods?status=consumed')])
      .then(([profile, familyResult, result]) => this.setData({ profile: displayProfile(profile), family: familyResult.family, consumedFoods: result.items.map(food => ({ ...food, expiryLabel: dateOnly(food.expiryDate) })) }))
      .catch(err => wx.showToast({ title: err.message, icon: 'none' }))
  },
  openEdit() { wx.navigateTo({ url: '/pages/profile-edit/index' }) },
  openFamily() { wx.navigateTo({ url: '/pages/family/index' }) },
  openFood(event) { wx.navigateTo({ url: `/pages/detail/index?id=${event.currentTarget.dataset.id}` }) }
})
