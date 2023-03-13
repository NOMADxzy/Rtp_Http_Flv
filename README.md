# 边缘节点

![system.png](https://s2.loli.net/2022/10/04/q2GfX9DdxPhsACH.png)

## 功能
- 接收rtp流`组播/单播`，解析得到flv内容队列
- 通过特定方式组织收到的rtp数据包，保证有序，发生丢包时通过`QUIC`协议重传rtp包
- 对收到的不同流，通过channel区分，通过`httpflv`服务向客户端提供不同channel的直播/点播

### 涉及的传输协议
- [RTMP](https://github.com/melpon/rfc/blob/master/rtmp.md)
- [RTP](https://www.rfc-editor.org/rfc/rfc3550.html)
- [QUIC](https://datatracker.ietf.org/doc/html/rfc9000)
- [HTTP-FLV](https://ossrs.io/lts/en-us/docs/v4/doc/delivery-http-flv)
- [HLS](https://www.rfc-editor.org/rfc/pdfrfc/rfc8216.txt.pdf)

## 安装

#### 使用预编译的可执行文件
[Releases](https://github.com/NOMADxzy/Rtp_Http_Flv/releases)
#### 从源码编译
1. 下载源码 `https://github.com/NOMADxzy/Rtp_Http_Flv.git`
2. 去 Rtp_Http_Flv 目录中 执行 `go build -o edgeserver.exe`

## 使用

#### 1. 启动[云端节点](https://github.com/NOMADxzy/Rtmp_Rtp_Flv)，监听rtmp`1935`端口;
`./cloudserver`

#### 2. 启动边缘节点，接收云端节点发过来的rtp流，并提供httpflv服务
`./edgeserver -httpflv_addr :7001 -hls_addr :7002 -udp_addr 127.0.0.1:5222 -pack_loss 0.002`

#### 3. 使用`ffmpeg`等工具推流到云端节点，命令`ffmpeg -re -i caton.mp4 -vcodec libx264 -acodec aac -f flv  rtmp://127.0.0.1:1935/live/movie`

#### 4. 通过以下方式播放
[flv 播放器](http://bilibili.github.io/flv.js/demo/)，输入播放地址播放：`http://127.0.0.1:7001/live/movie.flv` <br>

[hls 播放器](http://players.akamai.com/players/hlsjs)，输入播放地址播放：`http://127.0.0.1:7002/live/movie.m3u8`



### 主要参数配置

```bash
./Rtp_Http_Flv -h
Usage of ./Rtp_Http_Flv:
  -api_url string       云端节点http服务的接口("http://127.0.0.1:8090")
  -udp_addr string      监听udp的端口(":5222")
  -quic_addr string     云端节点quic服务的地址("127.0.0.1:4242")
  -httpflv_addr string  提供httpflv服务的地址(":7001")
  -disable_quic bool    是否停用quic重传(false)
  -padding_size int     rtp队列的缓冲长度(300)
  -queue_chan_size int  流的写入写出缓冲长度(100)
  -record_dir string    录制文件的存放目录("./record")
  -pack_loss float64    模拟丢包率(0.002)
  -enable_hls bool      开启hls服务(true)
  -hls_addr   string    hls服务地址(":7002")
```



## 项目结构
#### `cache`
- `cache.go`：主要是缓存 flvTag 的初始段 initialization segment，通常包含在首个音频和视频的 Tag 中，包含了媒体的基本信息，例如编解码格式以及采样率等，播放器必须拿到才能正确解码播放视频
- `RtpQueue.go`：用于缓存rtp包的队列，确保rtp包的有序和尽可能存在
- `FlvRecord.go`：解析flvTag的缓存，记录历史信息，每得到一个完整的tag后都会重新开始

#### `configure`
- 配置文件

#### `container`
- 一些协议的数据包格式，用来创建和解析不同协议的数据包

#### `protocol/httpflv`
- 提供httpflv服务的必要文件，向客户端传输数据包，主要数据结构是 flvWriter

#### `protocol/quic`
- quic 客户端，主要根据 sequence number 重传 rtp packet

#### `utils`
- tls、flv文件的读写、http请求等工具方法

#### `main.go`
- 主要代码逻辑，接收 rtp 数据, 解析、处理，提供http服务


### Todo

- [√] 效率问题？协程
- [√] 给出 flags 参数处理？
- [√] 代码逻辑优化？无用包
- [√] 多流处理