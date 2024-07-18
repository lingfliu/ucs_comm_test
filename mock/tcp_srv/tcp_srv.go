package main

import (
	"time"

	"lingfliu.github.com/ucs_comm_test/conn"
	"lingfliu.github.com/ucs_comm_test/ulog"
)

func _task_handle_recv(rx chan []byte, tx chan []byte) {
	for {
		select {
		case rx_buff := <-rx:
			tx <- rx_buff
		}
	}
}

func main() {
	ulog.Config(ulog.LOG_LEVEL_INFO, "", false)

	conn := conn.NewTcpConn("", 10071)

	conn.Accept()

	tx := make(chan []byte)
	rx := make(chan []byte)

	conn.StartRecv(rx)
	conn.StartWrite(tx)

	go _task_handle_recv(rx, tx)

	for {
		ulog.Log().I("test tcp srv test", "sleep")
		time.Sleep(1 * time.Second)
	}
}
