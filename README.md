# 北邮空教室查询系统

Go + MySQL + Vue 3/Vant 的空教室查询项目骨架，包含北邮教务移动端接口登录加密、每日同步、后端 API、移动端页面和 Docker 部署。

## 本地启动

1. 准备配置：

```powershell
Copy-Item .env.example .env
```

填写 `.env` 中的 `BUPT_USER_NO`，并优先填写已生成的 `BUPT_ENCRYPTED_PWD`。也可以只填 `BUPT_PASSWORD`，后端会按 AES-128-ECB + PKCS7 + 双层 Base64 实时生成密文。

2. 启动 MySQL 后运行后端：

```powershell
go mod tidy
go run .\cmd\server
```

3. 启动前端：

```powershell
cd frontend
npm install
npm run dev
```

前端默认访问 `http://localhost:5173`，后端默认访问 `http://localhost:8080`。

## Docker 启动

```powershell
docker compose up --build
```

前端入口为 `http://localhost:3000`，后端 API 为 `http://localhost:8080`。

## API

- `GET /healthz`
- `GET /api/campuses`
- `GET /api/slots`
- `GET /api/classrooms?campusId=0&date=2026-05-06&slot=6`
- `POST /api/sync`
- `POST /api/sync?campusId=1`

`campusId=0` 为西土城，`campusId=1` 为沙河。`slot` 为第 1 到第 14 节，不传时后端会按当前时间选择当前或下一节。

## 定时同步

服务默认每天 `05:30` 触发同步。失败后按 `5min -> 10min -> 15min` 重试，最后一次会落在 `06:00` 左右。

## 安全说明

不要提交真实学号、明文密码或 `.env` 文件。仓库默认忽略 `.env`，生产环境建议只注入 `BUPT_ENCRYPTED_PWD`。
