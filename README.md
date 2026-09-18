# Quiet Signal

一个基于 React + Go + Gin + GORM + SQLite 的个人博客 MVP。SQLite 使用 `github.com/glebarez/sqlite`，底层为纯 Go 实现，不依赖 CGO。

## 本地运行

```bash
cd frontend
npm install
npm run dev
```

前端默认地址：`http://localhost:5173`

启动后端会自动创建本地 SQLite 数据库文件，也可以通过环境变量修改数据库路径：

```bash
cd backend
go mod tidy
export DATABASE_PATH='./data/quietsig.db'
go run ./cmd/server
```

后台管理页：`http://localhost:5173/admin`

本地开发时，Vite 会把 `/api` 代理到 `http://localhost:8080`。请同时运行前端和后端。
如需修改代理目标，可设置 `VITE_API_PROXY_TARGET`。

## Docker

```bash
cd deploy
docker compose up --build
```

浏览器访问：`http://localhost`

默认管理员账号：`admin / admin123`。生产环境请替换 JWT 密钥和管理员密码。SQLite 数据会持久化到 Docker volume `sqlite_data`。
