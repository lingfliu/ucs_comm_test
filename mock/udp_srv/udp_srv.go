package main

import (
	"flag"
	"fmt"
	"os"
	"os/signal"
	"time"

	"lingfliu.github.com/ucs_comm_test/conn"
	"lingfliu.github.com/ucs_comm_test/ulog"
	"lingfliu.github.com/ucs_comm_test/utils"
)

/**
 * A pingpong task that will send the received data back to the client
 */
func _task_handle_recv(rx chan []byte, tx chan []byte, c *conn.UdpConn) {
	for c.Stat >= 0 {
		select {
		case rx_buff, ok := <-rx:
			if !ok {
				ulog.Log().I("udp_srv", "receive channel closed")
				return
			}
			ulog.Log().I("udp_srv", fmt.Sprintf("received %d bytes, echoing back", len(rx_buff)))
			c.LastRecvAt = utils.CurrentTimeInMicro()
			tx <- rx_buff
		}
	}
}

func _task_handle_conn(c *conn.UdpConn) {
	for c.Stat >= 0 {
		time.Sleep(5 * time.Second)
		tic := utils.CurrentTimeInMicro()
		// Check if no data received for more than 10 seconds (10,000,000 microseconds)
		if tic-c.LastRecvAt > 10*1000*1000 {
			ulog.Log().I("udp_srv", "client timeout, closing connection")
			c.Stat = -1
			c.Close()
			return
		}
	}
}

func main() {

	var host_port int

	flag.IntVar(&host_port, "host_port", 10072, "port")
	flag.Parse()

	ulog.Config(ulog.LOG_LEVEL_INFO, "", false)

	srvConn := conn.NewUdpConn("", host_port)

	ulog.Log().I("udp_srv", fmt.Sprintf("starting listening at port %d", host_port))
	chanC := make(chan *(conn.UdpConn))
	go srvConn.Accept(chanC)

	s := make(chan os.Signal, 1)
	signal.Notify(s, os.Interrupt)

	for {
		select {
		case c := <-chanC:
			ulog.Log().I("udp_srv", "new client connected")
			c.Stat = 0
			c.LastRecvAt = utils.CurrentTimeInMicro()
			tx := make(chan []byte)
			rx := make(chan []byte)

			c.StartRecv(rx)
			c.StartWrite(tx)
			go _task_handle_recv(rx, tx, c)
			go _task_handle_conn(c)
		case <-s:
			fmt.Print("received interrupt, exiting\n")
			srvConn.Close()
			return
		}
	}
}
