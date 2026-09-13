# Food Expiry Monitor API

Gin + SQLite backend for a WeChat Mini Program that tracks food expiry dates.

## Run locally

```bash
cp .env.example .env
go run ./cmd/server
```

The service starts at `http://localhost:8080`. SQLite data is persisted at `./data/food-expiry.db` by default.

## Mini Program local debugging

1. Start the API as above.
2. Open this repository in WeChat Developer Tools. The project root is `miniprogram/` (already declared in `project.config.json`).
3. In **Details → Local settings**, enable *Do not verify valid domains, web-view domains, TLS versions, and HTTPS certificates*.

For local debugging, change `environment` in `miniprogram/utils/config.js` to `development` and use `http://localhost:8080` as its `baseURL`. Development uses `/v1/auth/dev/login` automatically, so no WeChat AppID or AppSecret is needed. Production uses `wx.login()` and `https://api.example.com`; the development route does not exist when `APP_ENV=production`.

## API

All `/v1` food endpoints require `Authorization: Bearer <token>`.

| Method | Path | Purpose |
| --- | --- | --- |
| POST | `/v1/auth/wechat/login` | Exchange WeChat `code` for an app token |
| GET | `/v1/dashboard` | Count active, expiring and expired foods |
| GET/POST | `/v1/foods` | List or create foods |
| GET/PATCH | `/v1/foods/:id` | Read or edit a food |
| POST | `/v1/foods/:id/consume` | Mark consumed |
| POST | `/v1/foods/:id/discard` | Mark discarded |
| POST | `/v1/foods/:id/reminder` | Schedule one expiry reminder |

Date values use `YYYY-MM-DD`. A reminder is sent at 09:00 server-local time on the selected reminder date. The app must first obtain the user's WeChat subscription-message consent; the backend records and dispatches the resulting reminder job.

## Production notes

- Set a strong `TOKEN_SECRET` and set `APP_ENV=production`.
- Put the SQLite file on a local persistent volume; do not share it over NFS.
- The server enables WAL mode, foreign keys and a busy timeout at startup.
- Back up the SQLite database regularly. Configure WeChat credentials and the subscription template ID to enable real login and reminder delivery.
