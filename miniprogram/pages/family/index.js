const api = require('../../utils/api')
const config = require('../../utils/config')

function displayFamily(family) {
  if (!family) return null
  return { ...family, members: family.members.map(member => ({ ...member, avatarUrl: member.avatarUrl ? `${config.baseURL}${member.avatarUrl}` : '' })) }
}

Page({
  data: { family: null, familyName: '', inviteCode: '', saving: false },
  onShow() { this.load() },
  load() {
    api.request('GET', '/v1/family').then(result => this.setData({ family: displayFamily(result.family) })).catch(err => wx.showToast({ title: err.message, icon: 'none' }))
  },
  inputName(event) { this.setData({ familyName: event.detail.value }) },
  inputCode(event) { this.setData({ inviteCode: event.detail.value.toUpperCase().replace(/[^A-Z0-9]/g, '').slice(0, 6) }) },
  createFamily() {
    const name = this.data.familyName.trim()
    if (!name) return wx.showToast({ title: '请输入家庭名称', icon: 'none' })
    wx.showModal({ title: '创建家庭', content: '你当前的食品清单将共享给家庭成员。', success: result => {
      if (!result.confirm) return
      this.setData({ saving: true })
      api.request('POST', '/v1/family', { name }).then(family => { this.setData({ family: displayFamily(family) }); wx.showToast({ title: '家庭已创建', icon: 'success' }) }).catch(err => wx.showToast({ title: err.message, icon: 'none' })).finally(() => this.setData({ saving: false }))
    } })
  },
  joinFamily() {
    const inviteCode = this.data.inviteCode
    if (!/^[A-Z0-9]{6}$/.test(inviteCode)) return wx.showToast({ title: '请输入 6 位邀请码', icon: 'none' })
    wx.showModal({ title: '加入家庭', content: '你当前的食品清单将共享给家庭成员。', success: result => {
      if (!result.confirm) return
      this.setData({ saving: true })
      api.request('POST', '/v1/family/join', { inviteCode }).then(family => { this.setData({ family: displayFamily(family) }); wx.showToast({ title: '已加入家庭', icon: 'success' }) }).catch(err => wx.showToast({ title: err.message, icon: 'none' })).finally(() => this.setData({ saving: false }))
    } })
  },
  copyCode() { wx.setClipboardData({ data: this.data.family.inviteCode }) }
})
