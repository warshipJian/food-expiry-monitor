# 食光聆讯

“食光聆讯”是一款微信小程序，用于记录已购买食品的保质期、存放位置和数量，并在到期前通过微信订阅消息提醒用户优先食用，减少食品浪费。

项目包含：

- `miniprogram/`：原生微信小程序页面；
- `cmd/server/`：Go + Gin 服务端入口；
- `internal/`：登录、食品、提醒任务和 SQLite 数据访问逻辑。

## 功能

- 微信登录与多用户数据隔离；
- 食品手动录入与商品条码扫描；
- 冷藏、冷冻、食品柜等存放位置记录；
- 临期、过期和“我吃完了”历史清单；临期为今天起 3 天内到期的食品；
- 支持长按食品详情中的到期日期 3 秒后修改日期；
- 家庭共享：通过 6 位邀请码加入家庭，成员共同管理食品与统计；
- 用户主动授权后的微信订阅消息提醒；
- Gin + SQLite 单机部署，适合小规模使用。

## 本地启动服务端

```bash
cp .env.example .env
go run ./cmd/server
```

默认监听 `http://localhost:8080`，SQLite 数据保存在 `./data/food-expiry.db`。

本地 `.env` 的常用配置：

```env
APP_ENV=development
HTTP_ADDR=:8080
DATABASE_PATH=./data/food-expiry.db
TOKEN_SECRET=请替换为随机长字符串
```

## 小程序本地调试

1. 启动服务端。
2. 用微信开发者工具打开本仓库；`project.config.json` 已将小程序根目录配置为 `miniprogram/`。
3. 在“详情 → 本地设置”开启“不校验合法域名、web-view（业务域名）、TLS 版本以及 HTTPS 证书”。
4. 复制环境示例，并生成小程序运行时配置：

```bash
cp miniprogram/.env.local.example miniprogram/.env.local
npm run miniprogram:config
```

`.env.local` 默认连接 `http://localhost:8080` 并使用开发登录接口。该文件和自动生成的 `miniprogram/utils/config.runtime.js` 都不会提交到 Git。

## 提交体验版或正式版

编辑 `miniprogram/.env.local`：注释开发环境的两行，取消生产环境的两行注释：

```env
MINIPROGRAM_ENVIRONMENT=production
MINIPROGRAM_BASE_URL=https://api.example.com
```

重新生成配置后，再在微信开发者工具中编译、上传：

```bash
npm run miniprogram:config
```

上传完成后，建议切回开发环境并再次执行生成命令。生产环境使用 `wx.login()` 登录，且微信公众平台必须将实际 API 域名（例如 `https://api.example.com`）同时添加为 **request 合法域名** 和 **uploadFile 合法域名**。头像使用 `wx.uploadFile` 上传；仅配置 request 合法域名时，微信客户端会在请求到达服务端前拦截上传。

## 接口概览

除登录接口外，所有 `/v1` 接口都需要携带：

```text
Authorization: Bearer <token>
```

| 方法 | 地址 | 说明 |
| --- | --- | --- |
| POST | `/v1/auth/wechat/login` | 用微信 `code` 换取应用登录令牌 |
| GET | `/v1/dashboard` | 获取在库、临期、过期统计 |
| GET / POST | `/v1/foods` | 查询或新增食品 |
| GET / PATCH | `/v1/foods/:id` | 查询或修改食品 |
| POST | `/v1/foods/:id/consume` | 标记为已吃完，保留历史记录 |
| POST | `/v1/foods/:id/discard` | 标记为丢弃 |
| POST | `/v1/foods/:id/reminder` | 创建一次到期提醒任务 |
| GET / POST | `/v1/family` | 查询或创建家庭 |
| POST | `/v1/family/join` | 使用 6 位邀请码加入家庭 |

食品日期格式为 `YYYY-MM-DD`。提醒在用户授权订阅消息后创建，服务端按提醒日期扫描任务并发送模板消息。

## 食品到期日与提醒

在未吃完食品的详情页，长按到期日期卡片 3 秒可进入编辑模式，再点击日期选择新的到期日。到期日可提前或延后，以便修正录入错误。

提醒提供“到期前 7 天”和“到期前 3 天”两个选项。修改食品信息或到期日期会取消该食品尚未发送的提醒，修改后请重新设置提醒。

## 家庭共享

用户可在“我 → 我的家庭”创建家庭，或输入家人分享的 6 位大写邀请码加入。创建或加入时，用户已有的食品会并入家庭；之后所有家庭成员可共同查看、添加、编辑、吃完或丢弃食品，首页统计和“我吃完了”历史均显示家庭总数据。一个用户目前仅能加入一个家庭。

## 微信订阅消息

当前使用“保质期到期提醒”模板，模板字段映射如下：

| 模板字段 | 内容 |
| --- | --- |
| `date1` | 有效期至 |
| `number2` | 剩余天数 |
| `thing6` | 食物名称 |
| `number7` | 物品数量 |

用户在食品详情页点击“设置提醒”时，必须主动允许该订阅消息。一次性订阅授权只对应一次提醒任务。

## 生产部署

生产 API 地址请通过服务器环境和小程序本地环境文件配置；本文以 `https://api.example.com` 为示例。

服务使用 systemd 管理，环境文件位于服务器：

```text
/opt/food-expiry-monitor/.env
```

生产环境至少需要配置：

```env
APP_ENV=production
HTTP_ADDR=127.0.0.1:18080
DATABASE_PATH=/var/lib/food-expiry-monitor/food-expiry.db
TOKEN_SECRET=随机长字符串
WECHAT_APP_ID=wx1fd8d94b1e65111a
WECHAT_APP_SECRET=微信后台密钥
WECHAT_TEMPLATE_ID=订阅消息模板ID
```

常用运维命令：

```bash
systemctl status food-expiry-monitor
systemctl restart food-expiry-monitor
journalctl -u food-expiry-monitor -f
```

SQLite 数据库必须放在服务器本地持久磁盘，不要存放在 NFS 等共享网络文件系统。请定期备份数据库文件。
