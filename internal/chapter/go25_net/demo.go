// Package go25_net 演示网络编程基础与 TCP/UDP。
//
// 本章覆盖：
//   - net 包基础：Dial、Listen、Accept
//   - TCP 服务器与客户端
//   - 粘包问题与自定义协议（长度前缀、换行分隔）
//   - Deadline 超时控制
//   - UDP 通信
//   - DNS 解析与网络地址
//   - 连接关闭与错误处理
package go25_net

import (
	"bufio"
	"encoding/binary"
	"fmt"
	"io"
	"net"
	"strings"
	"sync"
	"time"
)

// Demo 是本章的统一入口。
func Demo() {
	fmt.Println("========== go25_net: 网络编程基础与 TCP/UDP ==========")

	fmt.Println("\n--- 25.1 net 包基础与 TCP Echo 服务 ---")
	demoTCPEcho()

	fmt.Println("\n--- 25.2 粘包问题与换行分隔协议 ---")
	demoLineProtocol()

	fmt.Println("\n--- 25.3 长度前缀协议 ---")
	demoLengthPrefixProtocol()

	fmt.Println("\n--- 25.4 Deadline 超时控制 ---")
	demoDeadline()

	fmt.Println("\n--- 25.5 UDP 通信 ---")
	demoUDP()

	fmt.Println("\n--- 25.6 DNS 解析与网络地址 ---")
	demoDNS()

	fmt.Println("\n--- 25.7 连接关闭与半关闭 ---")
	demoConnectionClose()

	fmt.Println("\n--- 25.8 并发服务器模式 ---")
	demoConcurrentServer()

	fmt.Println("\n========== 网络编程基础与 TCP/UDP 演示结束 ==========")
}

// demoTCPEcho 演示基础的 TCP Echo 服务器和客户端。
func demoTCPEcho() {
	// 启动服务器
	listener, err := net.Listen("tcp", "127.0.0.1:0") // 端口 0 自动分配
	if err != nil {
		fmt.Printf("监听失败：%v\n", err)
		return
	}
	defer listener.Close()

	addr := listener.Addr().String()
	fmt.Printf("TCP Echo 服务器监听在 %s\n", addr)

	// 在 goroutine 中处理连接
	go func() {
		conn, err := listener.Accept()
		if err != nil {
			return
		}
		defer conn.Close()

		// Echo：读取数据并原样返回
		buf := make([]byte, 1024)
		n, err := conn.Read(buf)
		if err != nil {
			return
		}
		conn.Write(buf[:n])
	}()

	// 客户端连接
	conn, err := net.Dial("tcp", addr)
	if err != nil {
		fmt.Printf("连接失败：%v\n", err)
		return
	}
	defer conn.Close()

	// 发送数据
	message := "Hello, TCP!"
	conn.Write([]byte(message))

	// 接收响应
	buf := make([]byte, 1024)
	n, err := conn.Read(buf)
	if err != nil {
		fmt.Printf("读取失败：%v\n", err)
		return
	}

	fmt.Printf("发送：%s\n", message)
	fmt.Printf("接收：%s\n", string(buf[:n]))
}

// demoLineProtocol 演示换行分隔协议（解决粘包）。
func demoLineProtocol() {
	listener, err := net.Listen("tcp", "127.0.0.1:0")
	if err != nil {
		fmt.Printf("监听失败：%v\n", err)
		return
	}
	defer listener.Close()

	addr := listener.Addr().String()
	fmt.Printf("换行分隔协议服务器监听在 %s\n", addr)

	// 服务器：逐行读取并回显
	go func() {
		conn, err := listener.Accept()
		if err != nil {
			return
		}
		defer conn.Close()

		scanner := bufio.NewScanner(conn)
		for scanner.Scan() {
			line := scanner.Text()
			fmt.Printf("  服务器收到：%s\n", line)
			fmt.Fprintf(conn, "ECHO: %s\n", line)
		}
	}()

	// 客户端：发送多行
	time.Sleep(10 * time.Millisecond) // 等待服务器启动
	conn, err := net.Dial("tcp", addr)
	if err != nil {
		fmt.Printf("连接失败：%v\n", err)
		return
	}
	defer conn.Close()

	messages := []string{"第一行", "第二行", "第三行"}
	for _, msg := range messages {
		fmt.Fprintf(conn, "%s\n", msg)
	}

	// 读取响应
	scanner := bufio.NewScanner(conn)
	for i := 0; i < len(messages) && scanner.Scan(); i++ {
		fmt.Printf("客户端收到：%s\n", scanner.Text())
	}
}

// demoLengthPrefixProtocol 演示长度前缀协议（4 字节长度 + 数据）。
func demoLengthPrefixProtocol() {
	listener, err := net.Listen("tcp", "127.0.0.1:0")
	if err != nil {
		fmt.Printf("监听失败：%v\n", err)
		return
	}
	defer listener.Close()

	addr := listener.Addr().String()
	fmt.Printf("长度前缀协议服务器监听在 %s\n", addr)

	// 服务器：读取长度前缀 + 数据
	go func() {
		conn, err := listener.Accept()
		if err != nil {
			return
		}
		defer conn.Close()

		for {
			// 读取 4 字节长度
			var length uint32
			err := binary.Read(conn, binary.BigEndian, &length)
			if err != nil {
				if err != io.EOF {
					fmt.Printf("  读取长度失败：%v\n", err)
				}
				return
			}

			// 读取指定长度的数据
			data := make([]byte, length)
			_, err = io.ReadFull(conn, data)
			if err != nil {
				fmt.Printf("  读取数据失败：%v\n", err)
				return
			}

			fmt.Printf("  服务器收到（长度 %d）：%s\n", length, string(data))

			// 回显：长度 + 数据
			binary.Write(conn, binary.BigEndian, length)
			conn.Write(data)
		}
	}()

	// 客户端：发送长度前缀 + 数据
	time.Sleep(10 * time.Millisecond)
	conn, err := net.Dial("tcp", addr)
	if err != nil {
		fmt.Printf("连接失败：%v\n", err)
		return
	}
	defer conn.Close()

	messages := []string{"短消息", "这是一条稍微长一点的消息"}
	for _, msg := range messages {
		// 写入长度
		length := uint32(len(msg))
		binary.Write(conn, binary.BigEndian, length)
		// 写入数据
		conn.Write([]byte(msg))

		// 读取响应
		var respLength uint32
		binary.Read(conn, binary.BigEndian, &respLength)
		respData := make([]byte, respLength)
		io.ReadFull(conn, respData)
		fmt.Printf("客户端收到（长度 %d）：%s\n", respLength, string(respData))
	}
}

// demoDeadline 演示 Deadline 超时控制。
func demoDeadline() {
	listener, err := net.Listen("tcp", "127.0.0.1:0")
	if err != nil {
		fmt.Printf("监听失败：%v\n", err)
		return
	}
	defer listener.Close()

	addr := listener.Addr().String()
	fmt.Printf("Deadline 演示服务器监听在 %s\n", addr)

	// 服务器：延迟 200ms 才响应
	go func() {
		conn, err := listener.Accept()
		if err != nil {
			return
		}
		defer conn.Close()

		buf := make([]byte, 1024)
		n, err := conn.Read(buf)
		if err != nil {
			return
		}

		time.Sleep(200 * time.Millisecond) // 模拟慢响应
		conn.Write(buf[:n])
	}()

	// 客户端：设置 100ms 读超时
	time.Sleep(10 * time.Millisecond)
	conn, err := net.Dial("tcp", addr)
	if err != nil {
		fmt.Printf("连接失败：%v\n", err)
		return
	}
	defer conn.Close()

	conn.Write([]byte("超时测试"))

	// 设置读 deadline：100ms 后超时
	conn.SetReadDeadline(time.Now().Add(100 * time.Millisecond))

	buf := make([]byte, 1024)
	_, err = conn.Read(buf)
	if err != nil {
		if netErr, ok := err.(net.Error); ok && netErr.Timeout() {
			fmt.Println("读取超时（预期行为）")
		} else {
			fmt.Printf("读取失败：%v\n", err)
		}
	}
}

// demoUDP 演示 UDP 通信。
func demoUDP() {
	// UDP 服务器
	serverAddr, err := net.ResolveUDPAddr("udp", "127.0.0.1:0")
	if err != nil {
		fmt.Printf("解析地址失败：%v\n", err)
		return
	}

	serverConn, err := net.ListenUDP("udp", serverAddr)
	if err != nil {
		fmt.Printf("监听失败：%v\n", err)
		return
	}
	defer serverConn.Close()

	actualAddr := serverConn.LocalAddr().String()
	fmt.Printf("UDP 服务器监听在 %s\n", actualAddr)

	// 服务器：接收数据并回显
	go func() {
		buf := make([]byte, 1024)
		n, clientAddr, err := serverConn.ReadFromUDP(buf)
		if err != nil {
			return
		}
		fmt.Printf("  服务器收到来自 %s 的 UDP 数据：%s\n", clientAddr, string(buf[:n]))
		serverConn.WriteToUDP(buf[:n], clientAddr)
	}()

	// UDP 客户端
	time.Sleep(10 * time.Millisecond)
	clientConn, err := net.DialUDP("udp", nil, serverConn.LocalAddr().(*net.UDPAddr))
	if err != nil {
		fmt.Printf("连接失败：%v\n", err)
		return
	}
	defer clientConn.Close()

	message := "Hello, UDP!"
	clientConn.Write([]byte(message))

	buf := make([]byte, 1024)
	clientConn.SetReadDeadline(time.Now().Add(100 * time.Millisecond))
	n, err := clientConn.Read(buf)
	if err != nil {
		fmt.Printf("读取失败：%v\n", err)
		return
	}

	fmt.Printf("客户端收到 UDP 响应：%s\n", string(buf[:n]))
}

// demoDNS 演示 DNS 解析与网络地址。
func demoDNS() {
	// 解析域名
	ips, err := net.LookupIP("localhost")
	if err != nil {
		fmt.Printf("DNS 解析失败：%v\n", err)
		return
	}
	fmt.Printf("localhost 解析结果：%v\n", ips)

	// 解析 TCP 地址
	tcpAddr, err := net.ResolveTCPAddr("tcp", "127.0.0.1:8080")
	if err != nil {
		fmt.Printf("解析 TCP 地址失败：%v\n", err)
		return
	}
	fmt.Printf("TCP 地址：%s（网络 %s，IP %s，端口 %d）\n",
		tcpAddr.String(), tcpAddr.Network(), tcpAddr.IP, tcpAddr.Port)

	// 本机网络接口
	interfaces, err := net.Interfaces()
	if err != nil {
		fmt.Printf("获取网络接口失败：%v\n", err)
		return
	}
	fmt.Printf("本机网络接口数量：%d\n", len(interfaces))
	if len(interfaces) > 0 {
		fmt.Printf("  第一个接口：%s（索引 %d，MTU %d）\n",
			interfaces[0].Name, interfaces[0].Index, interfaces[0].MTU)
	}
}

// demoConnectionClose 演示连接关闭与半关闭。
func demoConnectionClose() {
	listener, err := net.Listen("tcp", "127.0.0.1:0")
	if err != nil {
		fmt.Printf("监听失败：%v\n", err)
		return
	}
	defer listener.Close()

	addr := listener.Addr().String()
	fmt.Printf("连接关闭演示服务器监听在 %s\n", addr)

	// 服务器：读取直到 EOF
	go func() {
		conn, err := listener.Accept()
		if err != nil {
			return
		}
		defer conn.Close()

		data, err := io.ReadAll(conn)
		if err != nil {
			fmt.Printf("  服务器读取失败：%v\n", err)
			return
		}
		fmt.Printf("  服务器读取完毕，共 %d 字节：%s\n", len(data), string(data))

		// 回显
		conn.Write(data)
	}()

	// 客户端：发送数据后关闭写入（半关闭）
	time.Sleep(10 * time.Millisecond)
	conn, err := net.Dial("tcp", addr)
	if err != nil {
		fmt.Printf("连接失败：%v\n", err)
		return
	}
	defer conn.Close()

	conn.Write([]byte("数据1 "))
	conn.Write([]byte("数据2 "))
	conn.Write([]byte("数据3"))

	// 关闭写入（发送 FIN），但仍可读取
	if tcpConn, ok := conn.(*net.TCPConn); ok {
		tcpConn.CloseWrite()
		fmt.Println("客户端关闭写入（半关闭）")
	}

	// 读取响应
	buf := make([]byte, 1024)
	n, err := conn.Read(buf)
	if err != nil {
		fmt.Printf("客户端读取失败：%v\n", err)
		return
	}
	fmt.Printf("客户端收到响应：%s\n", string(buf[:n]))
}

// demoConcurrentServer 演示并发服务器模式（每连接一个 goroutine）。
func demoConcurrentServer() {
	listener, err := net.Listen("tcp", "127.0.0.1:0")
	if err != nil {
		fmt.Printf("监听失败：%v\n", err)
		return
	}
	defer listener.Close()

	addr := listener.Addr().String()
	fmt.Printf("并发服务器监听在 %s\n", addr)

	var wg sync.WaitGroup

	// 服务器：每个连接启动一个 goroutine
	go func() {
		for i := 0; i < 3; i++ {
			conn, err := listener.Accept()
			if err != nil {
				return
			}

			// 每连接一个 goroutine
			go handleConnection(conn, &wg)
		}
	}()

	// 启动 3 个客户端
	time.Sleep(10 * time.Millisecond)
	for i := 1; i <= 3; i++ {
		wg.Add(1)
		go func(id int) {
			conn, err := net.Dial("tcp", addr)
			if err != nil {
				fmt.Printf("客户端 %d 连接失败：%v\n", id, err)
				wg.Done()
				return
			}
			defer conn.Close()

			message := fmt.Sprintf("客户端 %d 的消息", id)
			conn.Write([]byte(message))

			buf := make([]byte, 1024)
			n, _ := conn.Read(buf)
			fmt.Printf("客户端 %d 收到：%s\n", id, string(buf[:n]))
		}(i)
	}

	wg.Wait()
}

// handleConnection 处理单个连接（用于并发服务器）。
func handleConnection(conn net.Conn, wg *sync.WaitGroup) {
	defer conn.Close()
	defer wg.Done()

	buf := make([]byte, 1024)
	n, err := conn.Read(buf)
	if err != nil {
		return
	}

	// 转大写并回显
	response := strings.ToUpper(string(buf[:n]))
	conn.Write([]byte(response))
}
