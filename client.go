package main

import (
	"bytes"
	"fmt"
	log "github.com/sirupsen/logrus"
	"io"
	"net"
	"os"
	"time"
)

func init() {
	log.SetFormatter(&log.TextFormatter{
		DisableColors: false,
		FullTimestamp: true,
	})
	log.SetReportCaller(true)
	log.SetOutput(os.Stdout)
	log.SetLevel(log.InfoLevel)
	f, err := os.Create("client.log")
	if err != nil {
		panic("创建日志文件失败")
	}
	log.SetOutput(f)
}
func main() {
	for i := 0; i < 600; i++ {
		go func(i int) {
			var conn net.Conn
			var err error
			for {
				conn, err = net.Dial("tcp", "127.0.0.1:5577")
				if err != nil {
					//log.Errorf("socket connect err: %s", err.Error())
				} else {
					break
				}
			}
			//	defer conn.Close()
			for {
				//_, err = write(conn, util.ConfigInfo.Socket.Confirm)
				//if err != nil {
				//	log.Errorf("socket send message: %s,error: %s", util.ConfigInfo.Socket.Confirm, err.Error())
				//	break
				//} else {
				//	//	log.Info("socket send message success")
				//}
				str, err := read(conn)
				if err != nil {
					if err == io.EOF {
						log.Info("conn close")
					} else {
						log.Error("read error: %s", err.Error())
					}
					break
				}
				if str != "" {
					log.Infof("receive:%s", str)
				}
				if str == "sid" {
					_, _ = write(conn, fmt.Sprintf("<%d>\r", i+1))
				} else {
					time.Sleep(300 * time.Millisecond)
					//_, _ = write(conn, "<OK>\r")
				}
				if i > 490 && i < 501 {
					conn.Close()
				}

			}
		}(i)
	}
	time.Sleep(10 * time.Second)
	go func() {
		var conn net.Conn
		var err error
		for {
			conn, err = net.Dial("tcp", "127.0.0.1:5577")
			if err != nil {
				//log.Errorf("socket connect err: %s", err.Error())
			} else {
				break
			}
		}

		//	defer conn.Close()
		go func() {
			for {
				write(conn, "1,mr")
				time.Sleep(5 * time.Second)
			}
		}()
		for {
			//_, err = write(conn, util.ConfigInfo.Socket.Confirm)
			//if err != nil {
			//	log.Errorf("socket send message: %s,error: %s", util.ConfigInfo.Socket.Confirm, err.Error())
			//	break
			//} else {
			//	//	log.Info("socket send message success")
			//}
			str, err := read(conn)
			if err != nil {
				if err == io.EOF {
					log.Info("conn close")
				} else {
					log.Error("read error: %s", err.Error())
				}
				break
			}
			if str != "" {
				log.Infof("receive:%s", str)
			}
			if str == "sid" {
				_, _ = write(conn, fmt.Sprintf("<%d>", 0))
			} else {
				time.Sleep(300 * time.Millisecond)
				//	_, _ = write(conn, "<OK>")
			}
		}
	}()
	select {}
}
func write(conn net.Conn, s string) (int, error) {
	var buffer bytes.Buffer
	buffer.WriteString(s + "\n")
	return conn.Write(buffer.Bytes())
}
func read(conn net.Conn) (string, error) {
	readByte := make([]byte, 1)
	var buffer bytes.Buffer
	for {
		_, err := conn.Read(readByte)
		if err != nil {
			return "", err
		}
		if string(readByte) == "\n" {
			break
		}
		buffer.Write(readByte)
	}
	return buffer.String(), nil

}
