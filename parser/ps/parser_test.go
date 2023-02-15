package ps

import (
	"bufio"
	"fmt"
	"os"
	"testing"
)

func TestPs(t *testing.T) {
	seq := []byte{
		0x00, 0x00, 0x01, 0xba, 0x44, 0x00, 0x04, 0x00, 0x04, 0x01, 0x00, 0x03, 0xff, 0xf8,
		0x00, 0x00, 0x01, 0xbb, 0x00, 0x0c, 0x04, 0x01, 0x00, 0x03, 0xff, 0xf8, 0x04, 0x01, 0x00, 0x03, 0xff, 0xf8,
		0x00, 0x00, 0x01, 0xbc, 0x00, 0x0c, 0xe0, 0xff, 0x00, 0x00, 0x00, 0x08, 0x90, 0xc0, 0x00, 0x00, 0x1b, 0xe0,
		0x45, 0xbd, 0xdc, 0xf4,
		0x00, 0x00, 0x01, 0xe0, 0x00, 0x12, 0xe0, 0xff, 0x02, 0x00, 0x00, 0x08, 0x90, 0xc0, 0x00, 0x00, 0x1b, 0xe1,
	}
	b := GetPsPayload(seq)

	//fmt.Println(len(seq))
	//fmt.Println(len(b))
	//fmt.Println(b)

	f, err := os.Create("./input.h264") //创建文件
	if err != nil {
		panic(err)
	}
	w := bufio.NewWriter(f) //创建新的 Writer 对象
	n4, err1 := w.Write(b)
	if err1 != nil {
		panic(err1)
	}
	fmt.Printf("写入 %d 个字节n", n4)
	w.Flush()
	f.Close()

	//fmt.Println(uint32(seq[0])<<8 | uint32(seq[1]))
}

func TestChan(t *testing.T) {

	//sq := []byte{0x05, 0x00, 0x00, 0x00, 0x01}

	//var buf = make([]byte, 2)
	//binary.BigEndian.PutUint16(buf, uint16(258))
	//return buf

	//b := uint32(258)
	//a := uint32(123)

	//b := utils.Uint32ToBytes(0)
	//fmt.Println(b)

	//data := [4]byte{0x6f, 0x32, 0x33, 0x34}
	////str := string(data[:])
	//str := *(*string)(unsafe.Pointer(&data))

	//m := flv.NewMetaTagData()
	//m.Init()
	//
	//metaTag := flv.NewTagPacket(flv.TAG_TYPE_SCRIPT)
	//metaTag.TagType = 0x12
	//meta := flv.NewMetaTagData()
	//meta.Init()
	//metaTag.SetPayload(meta)
	//fmt.Println(metaTag.Bytes())

	//var buf bytes.Buffer
	//var buff bytes.Buffer
	//buf.WriteString("abc")
	//buf.WriteTo(&buff)
	//
	//fmt.Println(buf.Bytes())
	//fmt.Println(buff.Bytes())

	p := []byte{0x2b, 0x5f, 0xdf, 0x5c, 0x95}

	var value int
	var b int

	b = int(p[0])
	// 第1个字节的第5、6、7位
	fmt.Println(uint64((b & 0x0e) << 29))
	value = int((b & 0x0e) << 29)

	//第2个字节的8位和第3个字节的前7位
	b = int(p[1] << 8)
	b += int(p[2] & 0xfe)
	value += int(b << 14)

	//第4个字节的8位和第5个字节的前7位

	b = int(p[3] << 8)
	b += int(p[4] & 0xfe)
	value += int(b >> 1)

	fmt.Println(value)

	//time.Sleep(1 * time.Second)
	//fmt.Println(uint32(seq[0])<<8 | uint32(seq[1]))
}
