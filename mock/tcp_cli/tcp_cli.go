package main

import (
	"encoding/binary"
	"fmt"
	"os"
	"path"
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

	if len(os.Args) < 2 {
		return
	}

	host_addr := os.Args[1]
	fmt.Print("connecting to ", host_addr, "\n")
	dir, err := os.Getwd()
	if err != nil {
		return
	}

	logPath := path.Join(dir, "log.log")
	ulog.Config(ulog.LOG_LEVEL_INFO, logPath, false)

	conn := conn.NewTcpConn(host_addr, 10071)

	ret := conn.Connect()
	if ret < 0 {
		fmt.Print("connect failed, exiting\n")
		return
	}

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
