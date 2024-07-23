package main

import (
	"encoding/binary"
	"fmt"
	"os"
	"path"
	"strconv"
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

func _task_write_pingpong(tx chan []byte, fps int) {
	idx := 0
	//convert fps to ms
	tic := time.NewTicker(time.Second / time.Duration(fps))

	for range tic.C {
		// fmt.Print("sending pingpong\n")
		idx++
		now := utils.CurrentTimeInMilli()
		bs := make([]byte, 16)
		binary.LittleEndian.PutUint64(bs, uint64(now))
		binary.LittleEndian.PutUint64(bs[8:], uint64(idx))
		tx <- bs
	}
}

func main() {
	var fps int
	var err error

	if len(os.Args) < 3 {
		return
	}

	host_addr := os.Args[1]
	fmt.Print("connecting to ", host_addr, "\n")

	fps, err = strconv.Atoi(os.Args[2])
	if err != nil {
		return
	}

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

	fmt.Print("sending pingpong at fps = ", fps)
	tx := make(chan []byte)
	rx := make(chan []byte)

	conn.StartRecv(rx)
	conn.StartWrite(tx)

	go _task_handle_recv(rx)

	go _task_write_pingpong(tx, fps)

	for {
		time.Sleep(1 * time.Second)
	}

}
