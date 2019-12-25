package main

//一个示例服务端读取消息
import (
	"bytes"
	"fmt"
	log "github.com/sirupsen/logrus"
	"io"
	"matrix/util"
	"net"
)

func main() {
	listener, err := net.Listen("tcp", util.ConfigInfo.Socket.Raddr)
	if err != nil {
		log.Fatalf("listen error: %s", err.Error())
	}
	defer listener.Close()
	for {
		conn, err := listener.Accept()
		if err != nil {
			log.Fatalf("accept error: %s", err.Error())
		}
		go handleConn(conn)
	}
}
func handleConn(conn net.Conn) {
	for {
		str, err := read(conn)
		if err != nil {
			if err == io.EOF {
				log.Info("conn close")
			} else {
				log.Error("read error: %s", err.Error())
			}
			break
		}
		fmt.Println(str)
	}
}

func read(conn net.Conn) (string, error) {
	readByte := make([]byte, 1)
	var buffer bytes.Buffer
	for {
		_, err := conn.Read(readByte)
		if err != nil {
			return "", err
		}
		if string(readByte) == util.ConfigInfo.Socket.Delimiter {
			break
		}
		buffer.Write(readByte)
	}
	return buffer.String(), nil

}
