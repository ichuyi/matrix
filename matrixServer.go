package main

//服务器向pc机发送消息
import (
	"bufio"
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	log "github.com/sirupsen/logrus"
	"math/rand"
	hooks "matrix/hook"
	"net"
	"os"
	"os/signal"
	"regexp"
	"strconv"
	"strings"
	"sync"
	"time"
)

type ServerConfig struct {
	LocalAddr         string            `json:"local_addr"`
	MaxCommand        int               `json:"max_command"`
	Duration          int64             `json:"execTime"`
	MaxSendTimes      int               `json:"max_send_times"`
	ReportDuration    int               `json:"report_duration"`
	MotorAddrList     map[string]string `json:"motor_addr_list"`
	MaxReconnectTimes int               `json:"max_reconnect_times"`
	SyncTimeMotors    []string          `json:"sync_time_motors"`
	SyncTimeGroupSize int               `json:"sync_time_group_size"`
	SendTimeDuration  int               `json:"send_time_duration"`
	SendGroupSize     int               `json:"send_group_size"`
}

var configPath = "matrixConfig.json"
var ConfigInfo ServerConfig
var quit chan os.Signal

type MotorResult struct {
	success bool
	lock    *sync.RWMutex
}
type Command struct {
	id         string
	content    string
	execTime   int64
	needResend bool
}
type MotorClient struct {
	command        map[string]chan *Command
	commandLock    *sync.RWMutex
	ctx            context.Context
	cancel         context.CancelFunc
	connectedMotor map[string]struct{}
	motorLock      *sync.RWMutex
	syncTime       []time.Time
	motorPowerOn   []int64
	startTime      time.Time
}

func NewMotorClient() *MotorClient {
	m := new(MotorClient)
	m.command = make(map[string]chan *Command, ConfigInfo.MaxCommand)
	m.commandLock = new(sync.RWMutex)
	m.connectedMotor = make(map[string]struct{})
	m.motorLock = new(sync.RWMutex)
	m.ctx, m.cancel = context.WithCancel(context.Background())
	m.syncTime = make([]time.Time, len(ConfigInfo.SyncTimeMotors))
	m.motorPowerOn = make([]int64, len(ConfigInfo.SyncTimeMotors))
	for i := 0; i < len(ConfigInfo.SyncTimeMotors); i++ {
		m.syncTime[i] = time.Now()
		m.motorPowerOn[i] = 0
	}
	m.startTime = time.Now()
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
	rand.Seed(time.Now().UnixNano())
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
	//	log.Infof("success to connect to motor %s", number)
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
func fastConnectMotor(number string, addr string) (net.Conn, error) {
	var conn net.Conn
	var err error
	i := 0
	for {
		conn, err = net.Dial("tcp", addr)
		if err == nil || i > ConfigInfo.MaxReconnectTimes {
			break
		} else {
			//log.Errorf("connect to motor %s error: %s, raddr is %s", number, err.Error(), addr)
		}
		i++
	}
	if err == nil {
		//	log.Infof("success to connect to motor %s", number)
	} else {
		//	log.Infof("failed to connect to motor %s", number)
	}
	return conn, err
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
		defer func() {
			if p := recover(); p != nil {
				log.Error("panic: ", p)
			}
		}()
		for {
			data, err := readData(r)
			if err != nil {
				//	log.Errorf("read data error: %s", err.Error())
				cancel()
				return
			} else if strings.Contains(data, "quit") {
				motorClient.cancel()
				return
			} else {
				cmd := strings.Split(data, ",")
				if len(cmd) < 2 {
					log.Errorf("incorrect data: %s", data)
				} else {
					log.Infof("receive command: %s", data)
					m := Command{
						id:         cmd[0],
						content:    cmd[1],
						needResend: true,
					}
					if len(cmd) > 2 {
						t, err := time.Parse("2006-01-02 15:04:05", cmd[2])
						if err != nil {
							log.Errorf("error: %s", err.Error())
						} else {
							d, err := strconv.Atoi(m.id)
							if err != nil {
								log.Errorf("parse id to int error: %s", err.Error())
								return
							}
							if ConfigInfo.SyncTimeGroupSize == 0 {
								log.Errorf("sync_time_group_size is 0")
								return
							}
							g := (d - 1) / ConfigInfo.SyncTimeGroupSize
							m.execTime = t.Sub(motorClient.syncTime[g]).Milliseconds() + motorClient.motorPowerOn[g]
						}
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
		defer func() {
			if p := recover(); p != nil {
				log.Error("panic: ", p)
			}
		}()
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
	d, err := strconv.Atoi(id)
	if err != nil {
		log.Errorf("parse id to int error: %s", err.Error())
		return
	}
	if ConfigInfo.SyncTimeGroupSize == 0 {
		log.Errorf("sync_time_group_size is 0")
	}
	g := (d - 1) / ConfigInfo.SyncTimeGroupSize
	go func() {
		defer func() {
			if p := recover(); p != nil {
				log.Error("panic: ", p)
			}
		}()
		timer := time.NewTimer(time.Duration(ConfigInfo.SendTimeDuration) * time.Millisecond)
		var totalSendDuration = (int64)(len(ConfigInfo.MotorAddrList) / ConfigInfo.SendGroupSize * ConfigInfo.SendTimeDuration)
		var motorSendDuration = (int64)((d - 1) / ConfigInfo.SendGroupSize * ConfigInfo.SendTimeDuration)
		defer timer.Stop()
		for {
			select {
			case <-ctx.Done():
				return
			case m := <-command:

				cmd := ""
				if m.execTime > 0 {
					cmd = fmt.Sprintf("$%d %s", m.execTime, m.content)
				} else {
					cmd = m.content
				}
				<-timer.C
				timer.Reset(time.Duration(ConfigInfo.SendTimeDuration) * time.Millisecond)
				n := time.Now().Sub(motorClient.startTime).Milliseconds()
				t := n%totalSendDuration - motorSendDuration
				for t > 0 && t < (int64)(ConfigInfo.SendTimeDuration) {
					time.Sleep(10 * time.Millisecond)
					n = time.Now().Sub(motorClient.startTime).Milliseconds()
					t = n%totalSendDuration - motorSendDuration
				}
				times := ConfigInfo.MaxSendTimes
				if m.needResend {
					times = 0
				}
				if sendCommand(c, m.id, cmd, result, times) != nil {
					var err error
					c, err = fastConnectMotor(id, ConfigInfo.MotorAddrList[id])
					if err != nil {
						log.Errorf("can't connect to motor %s", id)
						cancel()
						time.Sleep(5 * time.Second)
						go connectMotor(id, ConfigInfo.MotorAddrList[id])
					} else {
						_ = sendCommand(c, m.id, cmd, result, times)
					}
				}

			}
		}
	}()
	go func() {
		defer func() {
			if p := recover(); p != nil {
				log.Error("panic: ", p)
			}
		}()
		reg, err := regexp.Compile("<[0-9]+>")
		if err != nil {
			log.Errorf("regexp compile error: %s", err.Error())
			return
		}
		for {
			r := bufio.NewReader(c)
			data, err := readData(r)
			if err != nil {
				//	log.Errorf("read motor: %s error: %s", id, err.Error())
				c.Close()
				select {
				case <-ctx.Done():
					return
				default:
					var err error
					c, err = fastConnectMotor(id, ConfigInfo.MotorAddrList[id])
					if err != nil {
						cancel()
						time.Sleep(5 * time.Second)
						go connectMotor(id, ConfigInfo.MotorAddrList[id])
					}
				}
			} else {
				log.Infof("receive data: %s from %s", data, id)
				s := reg.FindString(data)
				if s != "" {
					s := s[1 : len(s)-1]
					d, err := strconv.ParseInt(s, 10, 64)
					if err != nil {
						log.Errorf("%s parse to int error: %s", s, err.Error())
						continue
					} else {
						//更新时间
						motorClient.motorPowerOn[g] = d
						motorClient.syncTime[g] = time.Now().UTC()
					}
				}
				if data == "<OK>\r" {
					result.lock.Lock()
					result.success = true
					result.lock.Unlock()
				}
			}

		}
	}()
	go func() {
		defer func() {
			if p := recover(); p != nil {
				log.Error("panic: ", p)
			}
		}()
		ticker := time.NewTicker(20 * time.Second)
		defer ticker.Stop()
		for {
			select {
			case <-ctx.Done():
				return
			case <-ticker.C:
				if len(command) == 0 {
					command <- &Command{
						id:      id,
						content: "greet",
					}
				}
			}
		}
	}()
	//分组时间同步
	if isExist(ConfigInfo.SyncTimeMotors, id) {
		go func() {
			defer func() {
				if p := recover(); p != nil {
					log.Error("panic: ", p)
				}
			}()
			if err := sendCommand(c, id, "gtime", result, 0); err != nil {
				log.Errorf("sync time error: %s", err.Error())
			}
			ticker := time.NewTicker(time.Hour * 6)
			for {
				select {
				case <-ctx.Done():
					ticker.Stop()
					return
				case <-ticker.C:
					if err := sendCommand(c, id, "gtime", result, 0); err != nil {
						log.Errorf("sync time error: %s", err.Error())
					}
				}
			}
		}()
	}

	<-ctx.Done()
}
func isExist(a []string, s string) bool {
	for i := range a {
		if a[i] == s {
			return true
		}
	}
	return false
}
func sendCommand(conn net.Conn, id string, cmd string, result *MotorResult, times int) error {
	defer func() {
		if err := recover(); err != nil {
			log.Errorf("panic error: %v", err)
		}
	}()
	if times > ConfigInfo.MaxSendTimes {
		log.Infof("failed to send %s to %s, have send for %d times", cmd, id, times)
		//return errors.New("failed to send")
		return nil
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
			time.Sleep(time.Duration(rand.Intn(100)) * time.Millisecond)
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
		//	log.Infof("send %s to motor %s", cmd, id)
		return nil
	}
}
