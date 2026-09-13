const { baseURL } = require('./config')

function request(method, path, data) {
  const app = getApp()
  return new Promise((resolve, reject) => {
    wx.request({
      url: `${baseURL}${path}`,
      method,
      data,
      header: app.globalData.token ? { Authorization: `Bearer ${app.globalData.token}` } : {},
      success(res) {
        if (res.statusCode >= 200 && res.statusCode < 300) return resolve(res.data)
        reject(new Error(res.data && res.data.error ? res.data.error : '请求失败'))
      },
      fail() { reject(new Error('无法连接本地服务，请确认后端正在运行')) }
    })
  })
}

function devLogin() {
  let deviceId = wx.getStorageSync('devDeviceId')
  if (!deviceId) { deviceId = `dev-${Date.now()}-${Math.random().toString(16).slice(2)}`; wx.setStorageSync('devDeviceId', deviceId) }
  return request('POST', '/v1/auth/dev/login', { deviceId }).then(res => res.token)
}

function wechatLogin() {
  return new Promise((resolve, reject) => {
    wx.login({
      success(result) {
        if (!result.code) return reject(new Error('微信登录未返回 code'))
        request('POST', '/v1/auth/wechat/login', { code: result.code }).then(response => resolve(response.token)).catch(reject)
      },
      fail() { reject(new Error('微信登录失败，请稍后再试')) }
    })
  })
}

module.exports = { request, devLogin, wechatLogin }
