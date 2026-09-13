const api = require('../../utils/api')
const config = require('../../utils/config')
const { dateOnly } = require('../../utils/date')

function displayProfile(profile) {
  return { ...profile, avatarUrl: profile.avatarUrl ? `${config.baseURL}${profile.avatarUrl}` : '' }
}

Page({
  data: { profile: { nickname: '', avatarUrl: '' }, nickname: '', consumedFoods: [], saving: false },
  onLoad() { this.load() },
  load() {
    Promise.all([api.request('GET', '/v1/profile'), api.request('GET', '/v1/foods?status=consumed')])
      .then(([profile, result]) => this.setData({ profile: displayProfile(profile), nickname: profile.nickname || '', consumedFoods: result.items.map(food => ({ ...food, expiryLabel: dateOnly(food.expiryDate) })) }))
      .catch(err => wx.showToast({ title: err.message, icon: 'none' }))
  },
  chooseAvatar(event) {
    const avatarUrl = event.detail.avatarUrl
    if (!avatarUrl) return
    this.setData({ saving: true })
    api.uploadAvatar(avatarUrl)
      .then(profile => { this.setData({ profile: displayProfile(profile) }); wx.showToast({ title: '头像已保存', icon: 'success' }) })
      .catch(err => wx.showToast({ title: err.message, icon: 'none' }))
      .finally(() => this.setData({ saving: false }))
  },
  inputNickname(event) { this.setData({ nickname: event.detail.value }) },
  saveNickname() {
    const nickname = this.data.nickname.trim()
    if (!nickname) { wx.showToast({ title: '请输入昵称', icon: 'none' }); return }
    this.setData({ saving: true })
    api.request('PATCH', '/v1/profile', { nickname })
      .then(profile => { this.setData({ profile: displayProfile(profile), nickname: profile.nickname }); wx.showToast({ title: '昵称已保存', icon: 'success' }) })
      .catch(err => wx.showToast({ title: err.message, icon: 'none' }))
      .finally(() => this.setData({ saving: false }))
  },
  openFood(event) { wx.navigateTo({ url: `/pages/detail/index?id=${event.currentTarget.dataset.id}` }) }
})
