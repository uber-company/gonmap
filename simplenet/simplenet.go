package simplenet

import (
	"crypto/tls"
	"errors"
	"io"
	"net"
	"strconv"
	"strings"
	"syscall"
	"time"
)

func tcpSend(protocol string, netloc string, data string, duration time.Duration, size int) (string, error) {
	protocol = strings.ToLower(protocol)
	var conn net.Conn
	var err error
	if protocol == "tcp" {
		arr := strings.Split(netloc, ":")
		if len(arr) != 2 {
			panic("have error")
		} else {
			port, _ := strconv.Atoi(arr[1])
			conn, err = ConnTcpWithPorts(arr[0], port, duration)
		}

	} else {
		conn, err = net.DialTimeout(protocol, netloc, duration)
	}

	if err != nil {
		//fmt.Println(conn)
		return "", errors.New(err.Error() + " STEP1:CONNECT")
	}
	defer conn.Close()
	_, err = conn.Write([]byte(data))
	if err != nil {
		return "", errors.New(err.Error() + " STEP2:WRITE")
	}
	//读取数据
	var buf []byte              // big buffer
	var tmp = make([]byte, 256) // using small tmo buffer for demonstrating
	var length int
	for {
		//设置读取超时Deadline
		_ = conn.SetReadDeadline(time.Now().Add(time.Second * 3))
		length, err = conn.Read(tmp)
		buf = append(buf, tmp[:length]...)
		if length < len(tmp) {
			break
		}
		if err != nil {
			break
		}
		if len(buf) > size {
			break
		}
	}
	if err != nil && err != io.EOF {
		return "", errors.New(err.Error() + " STEP3:READ")
	}
	if len(buf) == 0 {
		return "", errors.New("STEP3:response is empty")
	}
	return string(buf), nil
}

func tlsSend(protocol string, netloc string, data string, duration time.Duration, size int) (string, error) {
	protocol = strings.ToLower(protocol)
	config := &tls.Config{
		InsecureSkipVerify: true,
		MinVersion:         tls.VersionTLS10,
	}
	/* 	dialer := &net.Dialer{
		Timeout:  duration,
		Deadline: time.Now().Add(duration * 2),
	} */
	sourcePort := GetAvilableport()
	sourceAddr := &net.TCPAddr{
		IP:   net.IPv4(0, 0, 0, 0),
		Port: sourcePort,
	}
	dialer := &net.Dialer{
		LocalAddr: sourceAddr,
		Deadline:  time.Now().Add(duration * 2),
		Timeout:   duration, // 设置超时时间
		Control: func(network, address string, c syscall.RawConn) error {
			return c.Control(func(fd uintptr) {
				// 在这里可以调用 syscalls 设置套接字属性，例如:
				if err := syscall.SetsockoptInt(int(fd), syscall.SOL_SOCKET, syscall.SO_REUSEADDR, 1); err != nil {

				}
				linger := syscall.Linger{
					Onoff:  1, // 启用 SO_LINGER
					Linger: 0, // 立即关闭，不等待未发送的数据
				}
				if err := syscall.SetsockoptLinger(int(fd), syscall.SOL_SOCKET, syscall.SO_LINGER, &linger); err != nil {
				}
			})
		},
	}
	conn, err := tls.DialWithDialer(dialer, protocol, netloc, config)
	if err != nil {
		return "", errors.New(err.Error() + " STEP1:CONNECT")
	}
	defer conn.Close()
	_, err = io.WriteString(conn, data)
	if err != nil {
		return "", errors.New(err.Error() + " STEP2:WRITE")
	}
	//读取数据
	var buf []byte              // big buffer
	var tmp = make([]byte, 256) // using small tmo buffer for demonstrating
	var length int
	for {
		//设置读取超时Deadline
		_ = conn.SetReadDeadline(time.Now().Add(time.Second * 3))
		length, err = conn.Read(tmp)
		buf = append(buf, tmp[:length]...)
		if length < len(tmp) {
			break
		}
		if err != nil {
			break
		}
		if len(buf) > size {
			break
		}
	}
	if err != nil && err != io.EOF {
		return "", errors.New(err.Error() + " STEP3:READ")
	}
	if len(buf) == 0 {
		return "", errors.New("STEP3:response is empty")
	}
	return string(buf), nil
}

func Send(protocol string, tls bool, netloc string, data string, duration time.Duration, size int) (string, error) {
	if tls {
		return tlsSend(protocol, netloc, data, duration, size)
	} else {
		return tcpSend(protocol, netloc, data, duration, size)
	}
}

func GetAvilableport() int {
	var fd int
	var err error
	var localPort int

	fd, err = syscall.Socket(syscall.AF_INET, syscall.SOCK_STREAM, syscall.IPPROTO_TCP)
	if err != nil {
		return 0
	}
	defer func() {
		if fd != 0 {
			syscall.Close(fd)
		}
	}()

	for {
		localPort = Acquire() // 获取端口
		localAddr := syscall.SockaddrInet4{Port: localPort}
		copy(localAddr.Addr[:], net.ParseIP("0.0.0.0").To4())
		if err = syscall.Bind(fd, &localAddr); err != nil {
			//Release(localPort)
			continue // 绑定失败，继续尝试下一个端口
		}
		break
	}
	return localPort
}

func ConnTcpWithPorts(targetIP string, Port int, timeout time.Duration) (tcpConn net.Conn, err error) {

	// 获取一个可用的源端口
	sourcePort := GetAvilableport()

	// 创建源地址
	sourceAddr := &net.TCPAddr{
		IP:   net.IPv4(0, 0, 0, 0), // 0.0.0.0 表示任意可用的本地 IP 地址
		Port: sourcePort,           // 指定源端口
	}

	// 创建目标地址
	targetAddr := &net.TCPAddr{
		IP:   net.ParseIP(targetIP),
		Port: Port,
	}

	// 创建 Dialer 实例
	dialer := &net.Dialer{
		LocalAddr: sourceAddr,
		Timeout:   timeout, // 设置超时时间
		Control: func(network, address string, c syscall.RawConn) error {
			return c.Control(func(fd uintptr) {
				// 在这里可以调用 syscalls 设置套接字属性，例如:
				syscall.SetsockoptInt(int(fd), syscall.SOL_SOCKET, syscall.SO_REUSEADDR, 1)
				linger := syscall.Linger{
					Onoff:  1, // 启用 SO_LINGER
					Linger: 0, // 立即关闭，不等待未发送的数据
				}
				syscall.SetsockoptLinger(int(fd), syscall.SOL_SOCKET, syscall.SO_LINGER, &linger)
			})
		},
	}
	conn, err := dialer.Dial("tcp", targetAddr.String())
	if conn != nil {
		err = conn.SetDeadline(time.Now().Add(timeout))
	}
	// 连接到目标地址
	return conn, err

}
