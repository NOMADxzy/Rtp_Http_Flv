package rtmp

import "rtp_http_flv/protocol/rtmp/core"

const (
	PUBLISH = "publish"
	PLAY    = "play"
)

// 实现 RTMP 客户端
type Client struct {
	conn *core.ConnClient
}

func (c *Client) Dial(url string, method string) error {

	connClient := core.NewConnClient()

	connClient.Start(url, method)

	return nil
}
