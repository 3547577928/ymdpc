FROM node:22-alpine AS builder
# APP_VERSION 由 CI（git tag）或本地构建传入，TARGETPLATFORM 由 buildx 自动注入，用于关于页展示构建信息
ARG APP_VERSION
ARG TARGETPLATFORM
ENV VITE_APP_VERSION=$APP_VERSION VITE_BUILD_PLATFORM=$TARGETPLATFORM
WORKDIR /app
COPY package*.json ./
RUN npm ci
COPY . .
RUN npm run build

FROM nginx:1.27-alpine
COPY --from=builder /app/dist /usr/share/nginx/html
COPY nginx.conf /etc/nginx/conf.d/default.conf
