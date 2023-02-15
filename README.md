![system.png](https://s2.loli.net/2022/10/04/q2GfX9DdxPhsACH.png)

主要工作：

**1，收rtmp数据后封装为flv_tag**

![在这里插入图片描述](https://img-blog.csdnimg.cn/bcd259043ee04567a27e28f577e15892.png)

**2，将flv_tag打包为rtp数据包并发送**

flv_tag超过1000字节则分片发送，mark位标志是否为一个flvtag的结束，接收端根据mark位重组得到flv_tag


**3，边缘节点收到rtp包后，排序，检查丢包，如果丢包使用quic重传rtp包**


**4，边缘节点得到flv_tag后通过http-flv服务发送到客户端flv.js**