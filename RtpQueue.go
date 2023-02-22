package main

import (
	"fmt"
	"github.com/emirpasic/gods/lists/arraylist"
	"rtp_http_flv/container/rtp"
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
	readChan          chan interface{}
	reading           bool
}

func newQueue(wz int) *queue {
	return &queue{queue: arraylist.New(), PaddingWindowSize: wz, readChan: make(chan interface{}, 1)}
}

func (q *queue) Enqueue(rp *rtp.RtpPack) {
	q.m.Lock()
	defer q.m.Unlock()

	seq := rp.SequenceNumber
	if q.queue.Size() == 0 { // 队列中还没有元素
		q.FirstSeq = seq
		q.queue.Add(rp)
	} else {
		var relative int
		if q.FirstSeq > seq {
			if int(q.FirstSeq-seq) > 60000 { // 序列号到头
				relative = 65536 - int(q.FirstSeq) + int(seq)
			} else { // 过时的包
				fmt.Println("过时的包 ", seq, " ", q.FirstSeq)
				return
			}
		} else {
			relative = int(seq - q.FirstSeq)
		}
		if relative <= q.queue.Size() { //没到队列终点
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

// channel方式，多协程读取队列中的包，已弃用
func (q *queue) offerPacket() {
	q.reading = true
	for {
		rpEnd, _ := q.queue.Get(q.PaddingWindowSize - 1)
		if rpEnd == nil {
			continue
		}
		rp0, _ := q.queue.Get(0)
		if q.queue.Size() > q.PaddingWindowSize {

			//fmt.Println(rp0)
			q.readChan <- rp0

			q.m.Lock()
			rp, _ := q.queue.Get(q.PaddingWindowSize)
			if rp == nil {
				//重传
				seq := q.FirstSeq + uint16(q.PaddingWindowSize)
				fmt.Println("序号为", seq, "的包丢失，进行quic重传")
				go q.GetByQuic(seq)
				//q.queue.Set(i, pkt)
			}
			q.queue.Remove(0)
			q.FirstSeq += 1
			q.m.Unlock()
		}
	}
}

// Dequeue 必须确保 paddingsize 位置处的 rtp包已到达才能取包
func (q *queue) Dequeue() interface{} {
	// 确保窗口内的包都存在
	if q.queue.Size() < q.PaddingWindowSize+1 {
		return nil
	}
	rp, _ := q.queue.Get(q.PaddingWindowSize)
	if rp == nil {
		// 重传
		seq := q.FirstSeq + uint16(q.PaddingWindowSize)
		fmt.Println("序号为", seq, "的包丢失，进行quic重传")
		q.GetByQuic(seq)
		// q.queue.Set(i, pkt)
	}

	var res interface{}
	for {
		res, _ = q.queue.Get(0)
		//if res != nil {
		if true {
			q.m.Lock()
			q.queue.Remove(0)
			q.FirstSeq += 1
			q.m.Unlock()
			return res
		}
	}
}

// Check 检查窗口内队列Rtp的存在性和有序性
func (q *queue) Check() int {
	if q.Conn == nil {
		q.Conn = initQuic()
	}
	reTrans := 0
	//rtpParser := parser.NewRtpParser()
	for i := 0; i <= q.PaddingWindowSize; i++ {
		rp, _ := q.queue.Get(i)
		if rp == nil {
			//pkt := rtpParser.Parse([]byte{byte(128), byte(137), byte(16), byte(80), byte(14), byte(182),
			//	byte(27), byte(244), byte(0), byte(15), byte(145), byte(144), byte(8), byte(0), byte(1)}) //quic重传
			//q.queue.Set(i, pkt)
			fmt.Println("序号为", int(q.FirstSeq)+i, "的包丢失，进行quic重传")
			q.GetByQuic(q.FirstSeq + uint16(i))
			reTrans += 1
		}
		//if rp.(*rtp.RtpPack).SequenceNumber != q.FirstSeq+uint16(i) {
		//	fmt.Println("err ！Rtp Queue not sorted, FirstSeq:", q.FirstSeq, ", i:", i, ",SeqNum:", rp.(*rtp.RtpPack).SequenceNumber)
		//}
	}
	if reTrans == 0 {
		q.checked = true
	}
	return reTrans
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
