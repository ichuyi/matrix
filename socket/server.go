package socket

//服务器向pc机发送消息
import (
	"bytes"
	log "github.com/sirupsen/logrus"
	"matrix/util"
	"net"
	"sync"
	"time"
)

var Message chan string
var client []net.Conn
var lock = new(sync.Mutex)

func init() {
	Message = make(chan string, util.ConfigInfo.Socket.MaxMessage)
	client = make([]net.Conn, 0)
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
			} else {
				lock.Lock()
				client = append(client, conn)
				lock.Unlock()
				log.Infof("success connect to %s", conn.RemoteAddr())
				log.Infof("there have %d connections", len(client))
			}
		}
	}()
	go func() {
		defer close(Message)
		for {
			select {
			case m := <-Message:
				if ok, message := util.CheckArr(m); ok {
					for i := 0; i < len(client); {
						_, err := write(client[i], message+util.ConfigInfo.Socket.Delimiter)
						if err != nil {
							log.Errorf("socket send message failed,error: %s", err.Error())
							log.Infof("close connection between :%s", client[i].RemoteAddr())
							_ = client[i].Close()
							lock.Lock()
							client = append(client[0:i], client[i+1:]...)
							lock.Unlock()
							log.Infof("current there have %d connections", len(client))
						} else {
							log.Infof("socket send message successfully to: %s", client[i].RemoteAddr())
							i++
						}
					}
				}
			default:
				time.Sleep(100 * time.Millisecond)
			}
		}
	}()
}

//func handleConn(conn net.Conn) {
//	for {
//		str, err := read(conn)
//		if err != nil {
//			if err == io.EOF {
//				log.Info("conn close")
//			} else {
//				log.Error("read error: %s", err.Error())
//			}
//			break
//		}
//		fmt.Println(str)
//		if str == util.ConfigInfo.Socket.Confirm {
//			var message string
//			if len(Message) > 0 {
//				message, _ = <-Message
//			} else {
//				message = ""
//			}
//			_, err := write(conn, message+util.ConfigInfo.Socket.Delimiter)
//			if err != nil {
//				log.Errorf("socket send message: %s,error: %s", message, err.Error())
//			} else {
//				log.Info("socket send message success")
//			}
//		}
//	}
//}
//
//func read(conn net.Conn) (string, error) {
//	readByte := make([]byte, 1)
//	var buffer bytes.Buffer
//	for {
//		_, err := conn.Read(readByte)
//		if err != nil {
//			return "", err
//		}
//		if string(readByte) == util.ConfigInfo.Socket.Delimiter {
//			break
//		}
//		buffer.Write(readByte)
//	}
//	return buffer.String(), nil
//
//}
func write(conn net.Conn, s string) (int, error) {
	var buffer bytes.Buffer
	buffer.WriteString(s + util.ConfigInfo.Socket.Delimiter)
	return conn.Write(buffer.Bytes())
}
