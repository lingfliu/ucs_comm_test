package main

import (
	"encoding/binary"
	"fmt"
	"time"

	"lingfliu.github.com/ucs_comm_test/conn"
	"lingfliu.github.com/ucs_comm_test/ulog"
	"lingfliu.github.com/ucs_comm_test/utils"
)

func _task_handle_recv(rx chan []byte) {
	for {
		select {
		case rx_buff := <-rx:

			tic := binary.LittleEndian.Uint64(rx_buff[:8])
			idx := binary.LittleEndian.Uint64(rx_buff[8:])
			toc := utils.CurrentTimeInMilli()
			latency := toc - int64(tic)
			ulog.Log().I("tcpcli", fmt.Sprintf("recv pingpong idx = %d, latency = %d", idx, latency))
		}
	}
}

func _task_write_pingpong(tx chan []byte) {
	idx := 0
	tic := time.NewTicker(1 * time.Second)

	for range tic.C {
		idx++
		now := utils.CurrentTimeInMilli()
		bs := make([]byte, 16)
		binary.LittleEndian.PutUint64(bs, uint64(now))
		binary.LittleEndian.PutUint64(bs[8:], uint64(idx))
		tx <- bs
	}
}

func main() {
	ulog.Config(ulog.LOG_LEVEL_INFO, "", false)

	conn := conn.NewTcpConn("localhost", 10071)

	conn.Connect()

	tx := make(chan []byte)
	rx := make(chan []byte)

	conn.StartRecv(rx)
	conn.StartWrite(tx)

	go _task_handle_recv(rx)

	go _task_write_pingpong(tx)

	for {
		time.Sleep(1 * time.Second)
	}

}
