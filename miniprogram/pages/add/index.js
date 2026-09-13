const api = require('../../utils/api')
const { todayPlus } = require('../../utils/date')

Page({
  data: { form: { name: '', barcode: '', category: 'other', storageLocation: 'fridge', quantity: 1, unit: '件', expiryDate: todayPlus(7) }, categories: [{ value: 'other', label: '其他' }, { value: 'dairy', label: '乳制品' }, { value: 'fruit', label: '蔬果' }, { value: 'meat', label: '肉蛋' }, { value: 'snack', label: '零食' }], locations: [{ value: 'fridge', label: '冷藏' }, { value: 'freezer', label: '冷冻' }, { value: 'cabinet', label: '食品柜' }], categoryIndex: 0, locationIndex: 0, submitting: false },
  inputName(e) { this.setData({ 'form.name': e.detail.value }) },
  inputQuantity(e) { this.setData({ 'form.quantity': Number(e.detail.value) || 1 }) },
  chooseCategory(e) { this.setData({ categoryIndex: e.detail.value, 'form.category': this.data.categories[e.detail.value].value }) },
  chooseLocation(e) { this.setData({ locationIndex: e.detail.value, 'form.storageLocation': this.data.locations[e.detail.value].value }) },
  chooseDate(e) { this.setData({ 'form.expiryDate': e.detail.value }) },
  scan() {
    wx.scanCode({
      onlyFromCamera: false,
      success: result => { this.setData({ 'form.barcode': result.result }); wx.showToast({ title: '条码已识别', icon: 'success' }) },
      fail: () => wx.showToast({ title: '未识别条码，可手动填写名称', icon: 'none' })
    })
  },
  submit() {
    const { form, submitting } = this.data
    if (submitting) return
    if (!form.name.trim()) { wx.showToast({ title: '请填写食品名称', icon: 'none' }); return }
    this.setData({ submitting: true })
    api.request('POST', '/v1/foods', { ...form, name: form.name.trim() })
      .then(() => { wx.showToast({ title: '已添加', icon: 'success' }); setTimeout(() => wx.navigateBack(), 400) })
      .catch(err => wx.showToast({ title: err.message, icon: 'none' }))
      .finally(() => this.setData({ submitting: false }))
  }
})
