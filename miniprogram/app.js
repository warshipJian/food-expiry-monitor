const api = require('./utils/api')
const config = require('./utils/config')

App({
  globalData: { token: wx.getStorageSync('token') || '' },

  ensureLogin() {
    if (this.globalData.token) return Promise.resolve(this.globalData.token)
    const login = config.environment === 'production' ? api.wechatLogin : api.devLogin
    return login().then((token) => {
      this.globalData.token = token
      wx.setStorageSync('token', token)
      return token
    })
  }
})
