# Quiet Signal

一个基于 React + Go + Gin + GORM + SQLite 的轻量论坛。用户可以注册、登录、发文章、评论、点赞、关注作者，并在个人主页查看自己的社区数据。SQLite 使用 `github.com/glebarez/sqlite`，底层为纯 Go 实现，不依赖 CGO。

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

普通用户入口：

- 注册：`http://localhost:5173/register`
- 登录：`http://localhost:5173/login`
- 社区信息流：`http://localhost:5173/posts`
- 写文章：登录后访问 `http://localhost:5173/write`

写作页支持 Markdown 编辑/预览、本地草稿自动保存与恢复、文章目录、阅读进度、版本历史恢复、封面图片上传和定时发布。定时发布由后端每分钟扫描一次，到时间后自动公开文章并通知关注者。

图片默认保存到 `./data/uploads`，可通过 `UPLOAD_DIR` 修改。部署时需要持久化该目录，避免容器重建后图片丢失。

管理员后台保留文章编辑能力，并新增用户与社区统计：用户数、文章数、阅读量、点赞数、评论数、关注关系，以及每个用户的文章数、获赞数、粉丝数和关注数。

本地开发时，Vite 会把 `/api` 代理到 `http://localhost:8080`。请同时运行前端和后端。
如需修改代理目标，可设置 `VITE_API_PROXY_TARGET`。

## Docker

```bash
cd deploy
cp .env.example .env
# 编辑 .env，设置 JWT_SECRET、ADMIN_PASSWORD 和实际访问地址
docker compose up --build
```

浏览器访问：`http://localhost`

管理员用户名默认为 `admin`，密码和 JWT 密钥必须在 `deploy/.env` 中设置。SQLite 数据会持久化到 Docker volume `sqlite_data`；修改 `ADMIN_PASSWORD` 后重启后端即可轮换管理员密码。首次启动会自动把现有文章归属到管理员账号。

## 本地构建镜像并上传服务器

本地构建时，请根据服务器架构选择 `linux/amd64` 或 `linux/arm64`。常见云服务器一般使用 `linux/amd64`。

```bash
docker buildx build --platform linux/amd64 -t quietsignal-backend:latest --load ./backend
docker buildx build --platform linux/amd64 -t quietsignal-frontend:latest --load ./frontend
docker pull --platform linux/amd64 nginx:1.27-alpine
docker save -o quietsignal-images.tar quietsignal-backend:latest quietsignal-frontend:latest nginx:1.27-alpine
gzip quietsignal-images.tar
```

上传 `quietsignal-images.tar.gz`、`deploy/` 和 `deploy/.env` 到服务器后：

```bash
gunzip quietsignal-images.tar.gz
docker load -i quietsignal-images.tar
cd deploy
docker compose -f docker-compose.images.yml up -d
```
