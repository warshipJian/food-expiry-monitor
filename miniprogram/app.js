const api = require('./utils/api')

App({
  globalData: { token: wx.getStorageSync('token') || '' },

  ensureLogin() {
    if (this.globalData.token) return Promise.resolve(this.globalData.token)
    return api.devLogin().then((token) => {
      this.globalData.token = token
      wx.setStorageSync('token', token)
      return token
    })
  }
})
