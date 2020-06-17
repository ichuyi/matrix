package main

//服务器向pc机发送消息
import (
	"bufio"
	"bytes"
	"context"
	"encoding/json"
	"errors"
	log "github.com/sirupsen/logrus"
	hooks "matrix/hook"
	"net"
	"os"
	"os/signal"
	"strings"
	"sync"
	"time"
)

type ServerConfig struct {
	LocalAddr      string         `json:"local_addr"`
	MaxCommand     int            `json:"max_command"`
	Duration       int64          `json:"duration"`
	MaxSendTimes   int            `json:"max_send_times"`
	ReportDuration int            `json:"report_duration"`
	MotorAddrList  map[string]string `json:"motor_addr_list"`
	MaxReconnectTimes int `json:"max_reconnect_times"`
}

var configPath = "matrixConfig.json"
var ConfigInfo ServerConfig
var quit chan os.Signal

type MotorResult struct {
	success bool
	lock    *sync.RWMutex
}
type Command struct {
	id      string
	content string
}
type MotorClient struct {
	command        map[string]chan *Command
	commandLock    *sync.RWMutex
	ctx            context.Context
	cancel         context.CancelFunc
	connectedMotor map[string]struct{}
	motorLock      *sync.RWMutex
}

func NewMotorClient() *MotorClient {
	m := new(MotorClient)
	m.command = make(map[string]chan *Command, ConfigInfo.MaxCommand)
	m.commandLock = new(sync.RWMutex)
	m.connectedMotor = make(map[string]struct{})
	m.motorLock = new(sync.RWMutex)
	m.ctx, m.cancel = context.WithCancel(context.Background())
	return m
}

var motorClient *MotorClient

func parse() {
	conf, err := os.Open(configPath)
	if err != nil {
		log.Fatalf(err.Error())
	}
	err = json.NewDecoder(conf).Decode(&ConfigInfo)
	if err != nil {
		log.Fatalf(err.Error())
	}
}
func initLog() {
	log.SetFormatter(&log.TextFormatter{
		DisableColors: false,
		FullTimestamp: true,
	})
	log.SetReportCaller(true)
	log.SetLevel(log.DebugLevel)
	log.AddHook(hooks.NewContextHook())
	f, err := os.Create("server.log")
	if err != nil {
		panic("创建日志文件失败")
	}
	log.SetOutput(f)
}
func init() {
	parse()
	initLog()
	motorClient = NewMotorClient()
}
func main() {
	defer func() {
		if err := recover(); err != nil {
			log.Error(err)
		}
	}()
	quit = make(chan os.Signal)
	listener, err := net.Listen("tcp", ConfigInfo.LocalAddr)
	if err != nil {
		log.Fatalf("%s listen error: %s", ConfigInfo.LocalAddr, err.Error())
	}
	log.Infof("start to listen, addr is: %s", ConfigInfo.LocalAddr)
	go func() {
		for {
			conn, err := listener.Accept()
			if err != nil {
				log.Errorf("%s accept error: %s", ConfigInfo.LocalAddr, err.Error())
				return
			} else {
				go handlerUnity(conn)
			}
		}
	}()
	go func() {
		for id := range ConfigInfo.MotorAddrList {
			go connectMotor(id, ConfigInfo.MotorAddrList[id])
			time.Sleep(50 * time.Millisecond)
		}
	}()
	signal.Notify(quit)
	select {
	case <-quit:
	case <-motorClient.ctx.Done():
	}
	motorClient.cancel()
	log.Println("start closing server...")
	if listener != nil {
		listener.Close()
	}
	time.Sleep(5 * time.Second)
	log.Println("closed server")
}
func connectMotor(number string, addr string) {
	var conn net.Conn
	var err error
	for {
		conn, err = net.Dial("tcp", addr)
		if err == nil {
			break
		} else {
			//log.Errorf("connect to motor %s error: %s, raddr is %s", number, err.Error(), addr)
		}
		time.Sleep(30 * time.Second)
	}
	log.Infof("success to connect to motor %s", number)
	motorClient.motorLock.Lock()
	motorClient.connectedMotor[number] = struct{}{}
	motorClient.motorLock.Unlock()
	r := &MotorResult{}
	r.lock = new(sync.RWMutex)
	command := make(chan *Command, ConfigInfo.MaxCommand)
	motorClient.commandLock.Lock()
	motorClient.command[number] = command
	motorClient.commandLock.Unlock()
	go handleMotor(conn, number, r, command)
}
func fastConnectMotor(number string, addr string) (net.Conn,error) {
	var conn net.Conn
	var err error
	i:=0
	for {
		conn, err = net.Dial("tcp", addr)
		if err == nil||i>ConfigInfo.MaxReconnectTimes {
			break
		} else {
			//log.Errorf("connect to motor %s error: %s, raddr is %s", number, err.Error(), addr)
		}
		i++
	}
	if err==nil {
		log.Infof("success to connect to motor %s", number)
	}else{
		log.Infof("failed to connect to motor %s",number)
	}
	return conn,err
}
func handlerUnity(conn net.Conn) {
	defer func() {
		if err := recover(); err != nil {
			log.Errorf("panic error: %v", err)
		}
	}()
	defer func() {
		conn.Close()
	}()
	ctx, cancel := context.WithCancel(motorClient.ctx)
	defer cancel()
	r := bufio.NewReader(conn)
	go func() {
		for {
			data, err := readData(r)
			if err != nil {
				log.Errorf("read data error: %s", err.Error())
				cancel()
				return
			} else if strings.Contains(data, "quit") {
				motorClient.cancel()
				return
			} else {
				cmd := strings.Split(data, ",")
				if len(cmd) != 2 {
					log.Errorf("incorrect data: %s", data)
				} else {
					log.Infof("receive command: %s", data)
					m := Command{
						id:      cmd[0],
						content: cmd[1],
					}
					motorClient.commandLock.RLock()
					c, ok := motorClient.command[m.id]
					motorClient.commandLock.RUnlock()
					if ok {
						select {
						case c <- &m:
						default:
							log.Infof("command channel has filled", m.id)
						}
					} else {
						log.Infof("motor %s doesn't exist", m.id)
					}
				}
			}
		}

	}()
	go func() {
		ticker := time.NewTicker(time.Duration(ConfigInfo.ReportDuration) * time.Second)
		for {
			select {
			case <-ctx.Done():
				ticker.Stop()
				return
			case <-ticker.C:
				motorClient.motorLock.RLock()
				motor := make([]string, 0, len(motorClient.connectedMotor))
				for k := range motorClient.connectedMotor {
					motor = append(motor, k)
				}
				motorClient.motorLock.RUnlock()
				writeData(conn, strings.Join(motor, ","))
			}
		}
	}()
	<-ctx.Done()

}
func handleMotor(c net.Conn, id string, result *MotorResult, command chan *Command) {
	defer func() {
		if err := recover(); err != nil {
			log.Errorf("panic error: %v", err)
		}
	}()
	defer func() {
		c.Close()
		close(command)
		motorClient.commandLock.Lock()
		delete(motorClient.command, id)
		motorClient.commandLock.Unlock()
		motorClient.motorLock.Lock()
		delete(motorClient.connectedMotor, id)
		motorClient.motorLock.Unlock()
	}()
	ctx, cancel := context.WithCancel(motorClient.ctx)
	defer cancel()
	go func() {
		for {
			select {
			case <-ctx.Done():
				return
			case m := <-command:
					if sendCommand(c, m.id, m.content, result, 0) != nil {
						var err error
						c,err=fastConnectMotor(id,ConfigInfo.MotorAddrList[id])
						if err!=nil{
							cancel()
							time.Sleep(5*time.Second)
							go connectMotor(id,ConfigInfo.MotorAddrList[id])
						}else{
							_=sendCommand(c, m.id, m.content, result, 0)
						}
					}

			}
		}
	}()
	go func() {
		for {
			r := bufio.NewReader(c)
			data, err := readData(r)
			if err != nil {
				log.Errorf("read motor: %s error: %s", id, err.Error())
				c.Close()
					select {
					case <-ctx.Done():
						return
					default:
						var err error
						c,err=fastConnectMotor(id,ConfigInfo.MotorAddrList[id])
						if err!=nil{
							cancel()
							time.Sleep(5*time.Second)
							go connectMotor(id,ConfigInfo.MotorAddrList[id])
						}
					}
			} else {
				log.Infof("receive data: %s from %s", data, id)
				if data == "<OK>\r" {
					result.lock.Lock()
					result.success = true
					result.lock.Unlock()
				}
			}

		}
	}()
	<-ctx.Done()
}

func sendCommand(conn net.Conn, id string, cmd string, result *MotorResult, times int) error {
	defer func() {
		if err := recover(); err != nil {
			log.Errorf("panic error: %v", err)
		}
	}()
	if times > ConfigInfo.MaxSendTimes {
		log.Infof("failed to send %s to %s, have send for %d times", cmd, id, times)
		return errors.New("failed to send")
	}
	err := writeMotor(conn, id, cmd, result)
	if err != nil {
		return err
	}
	ctx, cancel := context.WithTimeout(context.Background(), time.Duration(ConfigInfo.Duration)*time.Millisecond)
	defer cancel()
	flag := false
	for {
		select {
		case <-ctx.Done():
			return sendCommand(conn, id, cmd, result, times+1)
		default:
			result.lock.RLock()
			flag = result.success
			result.lock.RUnlock()
			if flag {
				log.Infof("success to send %s to %s, have send for %d times", cmd, id, times+1)
				return nil
			}
		}
	}
}
func readData(r *bufio.Reader) (string, error) {
	s, err := r.ReadString('\n')
	if err != nil {
		return "", err
	}
	return s[:len(s)-1], err
}
func writeData(conn net.Conn, s string) (int, error) {
	var buffer bytes.Buffer
	buffer.WriteString(s + "\n")
	return conn.Write(buffer.Bytes())
}
func writeMotor(c net.Conn, id string, cmd string, result *MotorResult) error {
	result.lock.Lock()
	result.success = false
	result.lock.Unlock()
	_, err := writeData(c, cmd)
	if err != nil {
		log.Errorf("failed to send command :%s to motor: %s", cmd, id)
		return err
	} else {
		log.Infof("send %s to motor %s", cmd, id)
		return nil
	}
}
