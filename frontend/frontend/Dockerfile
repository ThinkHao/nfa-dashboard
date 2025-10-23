# syntax=docker/dockerfile:1

FROM node:20-alpine AS build
WORKDIR /app
RUN apk add --no-cache tzdata \
  && ln -snf /usr/share/zoneinfo/Asia/Shanghai /etc/localtime \
  && echo Asia/Shanghai > /etc/timezone
ENV TZ=Asia/Shanghai
COPY package*.json ./
RUN npm ci
COPY . ./
RUN npm run build-only

FROM nginx:alpine
WORKDIR /usr/share/nginx/html
RUN apk add --no-cache tzdata \
  && ln -snf /usr/share/zoneinfo/Asia/Shanghai /etc/localtime \
  && echo Asia/Shanghai > /etc/timezone
ENV TZ=Asia/Shanghai
COPY --from=build /app/dist/ ./
# 默认反代配置，可被 compose 挂载覆盖
COPY docker/default.conf /etc/nginx/conf.d/default.conf
EXPOSE 80
CMD ["nginx", "-g", "daemon off;"]
