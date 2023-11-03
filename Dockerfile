# Version 0.1

# 基础镜像
FROM golang:1.20

# 维护者信息
MAINTAINER zuyunxu@bupt.edu.cn

#镜像操作命令
WORKDIR /home/app

RUN apt update && apt install -y git vim

# pre-copy/cache go.mod for pre-downloading dependencies and only redownloading them in subsequent builds if they change
COPY go.mod go.sum ./
RUN go mod download && go mod verify

COPY . .
RUN go build -v -o /usr/local/bin/app ./main.go

EXPOSE 7001 5222

#容器启动命令
CMD ["app"]
