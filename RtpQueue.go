package main

import (
	"fmt"
	"github.com/emirpasic/gods/lists/arraylist"
	"go-mpu/container/rtp"
	"sync"
)

//type rtpQueueItem struct {
//	packet *RTPPacket
//	seq    uint16
//}

//[1,2,3,0,0]

type queue struct {
	m sync.RWMutex
	//maxSize      int
	FirstSeq          uint16          //第一个Rtp包的序号
	PaddingWindowSize int             //滑动窗口大小
	queue             *arraylist.List //rtpPacket队列
	Conn              *conn           //维持Quic连接
	checked           bool
	readChan          chan *rtp.RtpPack
}

func newQueue(wz int) *queue {
	return &queue{queue: arraylist.New(), PaddingWindowSize: wz}
}

func (q *queue) Enqueue(rp *rtp.RtpPack) {
	q.m.Lock()
	defer q.m.Unlock()

	seq := rp.SequenceNumber
	if q.queue.Size() == 0 { //队列中还没有元素
		q.FirstSeq = seq
		q.queue.Add(rp)
	} else {
		relative := int(seq - q.FirstSeq)
		if relative < 0 { //过时的包
			return
		} else if relative <= q.queue.Size() { //没到队列终点
			q.queue.Set(relative, rp)
		} else {
			for i := q.queue.Size(); i <= relative; i++ {
				if i != relative {
					q.queue.Set(i, nil)
					continue
				}
				q.queue.Set(i, rp)
			}
		}

	}

}

func (q *queue) offerPacket() {
	for {
		rp, _ := q.queue.Get(0)
		if rp != nil {
			q.readChan <- rp.(*rtp.RtpPack)
		}
	}
}

func (q *queue) Dequeue() *rtp.RtpPack { //必须确保paddingsize位置处的rtp包已到达才能取包
	//确保窗口内的包都存在
	rp, _ := q.queue.Get(q.PaddingWindowSize)
	if rp == nil {
		//重传
		seq := q.FirstSeq + uint16(q.PaddingWindowSize)
		fmt.Println("序号为", seq, "的包丢失，进行quic重传")
		go q.GetByQuic(seq)
		//q.queue.Set(i, pkt)
	}

	var res interface{}
	for {
		res, _ = q.queue.Get(0)
		if res != nil {
			q.m.Lock()
			q.queue.Remove(0)
			q.FirstSeq += 1
			q.m.Unlock()
			return res.(*rtp.RtpPack)
		}
	}
}

func (q *queue) Check() int { //检查窗口内队列Rtp的存在性和有序性
	if q.Conn == nil {
		q.Conn = initQuic()
	}
	re_trans := 0
	//rtpParser := parser.NewRtpParser()
	for i := 0; i <= q.PaddingWindowSize; i++ {
		rp, _ := q.queue.Get(i)
		if rp == nil {
			//pkt := rtpParser.Parse([]byte{byte(128), byte(137), byte(16), byte(80), byte(14), byte(182),
			//	byte(27), byte(244), byte(0), byte(15), byte(145), byte(144), byte(8), byte(0), byte(1)}) //quic重传
			//q.queue.Set(i, pkt)
			fmt.Println("序号为", int(q.FirstSeq)+i, "的包丢失，进行quic重传")
			q.GetByQuic(q.FirstSeq + uint16(i))
			re_trans += 1
		}
		//if rp.(*rtp.RtpPack).SequenceNumber != q.FirstSeq+uint16(i) {
		//	fmt.Println("err ！Rtp Queue not sorted, FirstSeq:", q.FirstSeq, ", i:", i, ",SeqNum:", rp.(*rtp.RtpPack).SequenceNumber)
		//}
	}
	if re_trans == 0 {
		q.checked = true
	}
	return re_trans
}
func (q *queue) print() {
	fmt.Println("rtp队列长度：", q.queue.Size())
	fmt.Print("rtp队列：")
	for i := 0; i < q.queue.Size(); i++ {
		rp, _ := q.queue.Get(i)
		if rp == nil {
			fmt.Print(" nil")
		} else {
			fmt.Print(rp.(*rtp.RtpPack).SequenceNumber, " ")
		}
	}
	fmt.Println()

}
