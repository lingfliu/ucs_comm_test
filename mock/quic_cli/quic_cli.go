package main

import (
	"encoding/binary"
	"flag"
	"fmt"
	"os"
	"os/signal"
	"path"
	"time"

	"lingfliu.github.com/ucs_comm_test/conn"
	"lingfliu.github.com/ucs_comm_test/ulog"
	"lingfliu.github.com/ucs_comm_test/utils"
)

func _task_handle_recv(rx chan []byte) {
	latency_buff := make([]int64, 0, 1)
	for {
		select {
		case rx_buff, ok := <-rx:
			if !ok {
				ulog.Log().I("quiccli", "receive channel closed")
				return
			}
			if len(rx_buff) < 16 {
				ulog.Log().I("quiccli", fmt.Sprintf("received invalid data length: %d", len(rx_buff)))
				continue
			}
			tic := binary.LittleEndian.Uint64(rx_buff[:8])
			idx := binary.LittleEndian.Uint64(rx_buff[8:])
			toc := utils.CurrentTimeInNano()
			latency := toc - int64(tic)
			latency_buff = append(latency_buff, latency)
			if len(latency_buff) > 100 {
				latency_buff = latency_buff[1:]
			}
			avg_latency := int64(0)
			for _, v := range latency_buff {
				avg_latency += v
			}
			avg_latency /= int64(len(latency_buff))
			ulog.Log().I("quiccli", fmt.Sprintf("recv pingpong idx = %d, latency = %d, avg_latency = %d", idx, latency, avg_latency))
		}
	}
}

func _task_write_pingpong(tx chan []byte, fps int) {
	idx := 0
	//convert fps to ms
	tic := time.NewTicker(time.Second / time.Duration(fps))

	for range tic.C {
		// ulog.Log().I("quiccli", fmt.Sprintf("sending pingpong idx = %d", idx))
		idx++
		now := utils.CurrentTimeInNano()
		bs := make([]byte, 16)
		binary.LittleEndian.PutUint64(bs, uint64(now))
		binary.LittleEndian.PutUint64(bs[8:], uint64(idx))
		tx <- bs
	}
}

func main() {
	var fps int
	var host_addr string
	var host_port int
	var logFile string

	var err error

	if len(os.Args) < 3 {
		return
	}

	flag.StringVar(&host_addr, "host_addr", "127.0.0.1", "host")
	flag.IntVar(&host_port, "host_port", 10074, "port")
	flag.IntVar(&fps, "fps", 10, "fps")
	flag.StringVar(&logFile, "log_file", fmt.Sprintf("%s_quic.log", time.Now().Format("20060102_150405")), "log_file")

	flag.Parse()

	fmt.Print("connecting to ", host_addr, ":", host_port, "\n")

	dir, err := os.Getwd()
	if err != nil {
		return
	}

	logPath := path.Join(dir, "log.log")
	ulog.Config(ulog.LOG_LEVEL_INFO, logPath, false)

	conn := conn.NewQuicConn(host_addr, host_port)

	ret := conn.Connect()
	if ret < 0 {
		fmt.Print("connect failed, exit\n")
		return
	}

	fmt.Print("connected, start pingpong at fps = ", fps, "\n")
	tx := make(chan []byte)
	rx := make(chan []byte)

	conn.StartRecv(rx)
	conn.StartWrite(tx)

	go _task_handle_recv(rx)

	go _task_write_pingpong(tx, fps)

	s := make(chan os.Signal, 1)
	signal.Notify(s, os.Interrupt)

	<-s
	fmt.Print("received interrupt, exiting\n")
	conn.Close()

}
