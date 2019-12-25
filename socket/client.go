package socket

import (
	"bytes"
	log "github.com/sirupsen/logrus"
	"matrix/util"
	"net"
)

var Message chan string

func init() {
	Message = make(chan string, util.ConfigInfo.Socket.MaxMessage)
	go func() {
		defer close(Message)
		for m := range Message {
			conn, err := net.Dial("tcp", util.ConfigInfo.Socket.Raddr)
			if err != nil {
				log.Fatalf("socket connect err: %s", err.Error())
			}
			_, err = write(conn, m)
			if err != nil {
				log.Errorf("socket send message: %s,error: %s", m, err.Error())
			} else {
				log.Info("socket send message success")
			}
			conn.Close()
		}
	}()
}
func write(conn net.Conn, s string) (int, error) {
	var buffer bytes.Buffer
	buffer.WriteString(s + util.ConfigInfo.Socket.Delimiter)
	return conn.Write(buffer.Bytes())
}
