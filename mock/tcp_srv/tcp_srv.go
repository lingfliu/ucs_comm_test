package main

import (
	"os"
	"path"
	"time"

	"lingfliu.github.com/ucs_comm_test/conn"
	"lingfliu.github.com/ucs_comm_test/ulog"
	"lingfliu.github.com/ucs_comm_test/utils"
)

func _task_handle_recv(rx chan []byte, tx chan []byte, c *conn.TcpConn) {
	for c.Stat >= 0 {
		select {
		case rx_buff := <-rx:
			c.LastRecvAt = utils.CurrentTimeInMilli()
			tx <- rx_buff
		}
	}
}

func _task_handle_conn(c *conn.TcpConn) {
	time.Sleep(1 * time.Second)
	tic := utils.CurrentTimeInMilli()
	if tic-c.LastRecvAt > 1000*1000*1000*2 {
		c.Stat = -1
		c.Close()
	}
}

func main() {

	dir, err := os.Getwd()
	if err != nil {
		return
	}
	logPath := path.Join(dir, "log.log")
	ulog.Config(ulog.LOG_LEVEL_INFO, logPath, false)
	srvConn := conn.NewTcpConn("", 10071)

	chanC := make(chan *(conn.TcpConn))
	go srvConn.Accept(chanC)

	select {
	case c := <-chanC:
		c.Stat = 0
		tx := make(chan []byte)
		rx := make(chan []byte)

		c.StartRecv(rx)
		c.StartWrite(tx)
		go _task_handle_recv(rx, tx, c)
		go _task_handle_conn(c)
	}

	for {
		// ulog.Log().I("test tcp srv test", "sleep")
		time.Sleep(1 * time.Second)
	}
}
