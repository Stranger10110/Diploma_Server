package handler

import (
	"../utils"
	"fmt"
	"io"
	"net"
	"os"
	"strconv"
	"time"
)

type SendFileClass struct {
	maxRam int

	oppositeBuff int
	buffs        [2][]byte
	sync         chan int

	file     *os.File
	fileSize int64
	ip       string
	port     int
}

func NewSendFile(maxRam float64) (s *SendFileClass) {
	s = new(SendFileClass)
	s.maxRam = int(maxRam * 1024 * 512) // (maxRam * 1024 * 1024) / 2
	s.oppositeBuff = 1
	s.sync = make(chan int)
	s.buffs[0] = make([]byte, s.maxRam)
	s.buffs[1] = make([]byte, s.maxRam)
	return
}

func (s *SendFileClass) SendFile(filePath string, ip string, port int) {
	s.ip = ip
	s.port = port

	// open file
	file, err := os.Open(filePath)
	utils.CheckError(err, "SendFile [1]", false)
	defer file.Close()
	s.file = file
	fileStat, err := file.Stat()
	utils.CheckError(err, "SendFile [2]", false)
	s.fileSize = fileStat.Size()
	if s.fileSize < int64(s.maxRam) {
		s.buffs[0] = make([]byte, s.fileSize)
		// s.buffs[1] = make([]byte, s.fileSize)
	}

	// first read
	_, err = file.Read(s.buffs[0])
	utils.CheckError(err, "SendFile [3]", false)

	go s.tcpSending()
	s.sync <- 1

	step := int64(s.maxRam)
	lastBytes := s.fileSize % int64(cap(s.buffs[0]))
	var n int
	for i := int64(0); i < s.fileSize-lastBytes; i += step {
		n, err = s.file.Read(s.buffs[s.oppositeBuff])
		if int64(n) < step {
			s.buffs[s.oppositeBuff] = s.buffs[s.oppositeBuff][:n]
		}
		fmt.Print("read ")
		fmt.Print(s.oppositeBuff)
		fmt.Print(s.buffs[s.oppositeBuff][:10])
		fmt.Println("\n")

		_ = <-s.sync
		s.oppositeBuff ^= 1
		s.sync <- 1
	}

	// read last bytes
	s.buffs[0] = s.buffs[0][:lastBytes]
	_, err = s.file.Read(s.buffs[0])
	if err != nil {
		if err != io.EOF {
			fmt.Println(err)
		}
		if ((s.fileSize / step) % 2) != 0 {
			s.buffs[0] = s.buffs[1][:lastBytes]
		}
	}

	s.sync <- 1
	_ = <-s.sync
	fmt.Print("Read done  ")
	fmt.Println(s.fileSize / step)
}

func (s *SendFileClass) tcpSending() {
	addr := s.ip + ":" + strconv.Itoa(s.port)
	tcpAddr, err := net.ResolveTCPAddr("tcp4", addr)
	utils.CheckError(err, "tcpSending [1]", false)

	conn, err := net.DialTCP("tcp", nil, tcpAddr)
	utils.CheckError(err, "tcpSending [2]", false)

	step := int64(s.maxRam)
	lastBytes := s.fileSize % int64(cap(s.buffs[0]))
	var n int
	start_time := time.Now()
	for i := int64(0); i < s.fileSize-lastBytes; i += step {
		_ = <-s.sync

		fmt.Print("write ")
		fmt.Print(s.oppositeBuff ^ 1)
		fmt.Print(s.buffs[s.oppositeBuff^1][:10])
		fmt.Println("\n")
		for written := int64(0); written < step; {
			n, err = conn.Write(s.buffs[s.oppositeBuff^1][written:])
			utils.CheckError(err, "tcpSending [3]", false)
			written += int64(n)
		}

		s.sync <- 1
	}

	// write last bytes
	_ = <-s.sync
	_ = <-s.sync
	for written := int64(0); written < lastBytes; {
		n, err = conn.Write(s.buffs[0][written:lastBytes])
		utils.CheckError(err, "tcpSending [4]", false)
		written += int64(n)
	}
	s.sync <- 1
	fmt.Println("Write done")
	fmt.Printf("Sending time %f sec", time.Since(start_time).Seconds())
}
