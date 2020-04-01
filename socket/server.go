package socket

//服务器向pc机发送消息
import (
	"bytes"
	"fmt"
	log "github.com/sirupsen/logrus"
	"io"
	"matrix/util"
	"net"
)

var Message chan string

func init() {
	Message = make(chan string, util.ConfigInfo.Socket.MaxMessage)
	go func() {
		listener, err := net.Listen("tcp", util.ConfigInfo.Socket.Laddr)
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
	}()
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
		if str == util.ConfigInfo.Socket.Confirm {
			var message string
			if len(Message) > 0 {
				message, _ = <-Message
			} else {
				message = ""
			}
			_, err := write(conn, message+util.ConfigInfo.Socket.Delimiter)
			if err != nil {
				log.Errorf("socket send message: %s,error: %s", message, err.Error())
			} else {
				log.Info("socket send message success")
			}
		}
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
func write(conn net.Conn, s string) (int, error) {
	var buffer bytes.Buffer
	buffer.WriteString(s + util.ConfigInfo.Socket.Delimiter)
	return conn.Write(buffer.Bytes())
}
