package go25_net

import (
	"encoding/binary"
	"io"
	"net"
	"sync"
	"testing"
	"time"
)

// TestTCPEchoRoundTrip 用本机回环验证「客户端发送 → 服务器回显」的完整链路。
// 端口用 127.0.0.1:0 让内核分配，避免和固定端口冲突。
func TestTCPEchoRoundTrip(t *testing.T) {
	ln, err := net.Listen("tcp", "127.0.0.1:0")
	if err != nil {
		t.Fatalf("监听失败: %v", err)
	}
	defer ln.Close()

	// 服务器：读到 EOF 前一直回显
	go func() {
		conn, err := ln.Accept()
		if err != nil {
			return
		}
		defer conn.Close()
		_, _ = io.Copy(conn, conn)
	}()

	conn, err := net.Dial("tcp", ln.Addr().String())
	if err != nil {
		t.Fatalf("连接失败: %v", err)
	}
	defer conn.Close()

	const msg = "echo round-trip"
	if _, err := io.WriteString(conn, msg); err != nil {
		t.Fatalf("写入失败: %v", err)
	}
	// 半关闭写入（发 FIN），服务器 io.Copy 读到 EOF 后才回显完所有数据
	if tc, ok := conn.(*net.TCPConn); ok {
		_ = tc.CloseWrite()
	}

	got, err := io.ReadAll(conn)
	if err != nil {
		t.Fatalf("读取失败: %v", err)
	}
	if string(got) != msg {
		t.Fatalf("回显不符：got %q, want %q", got, msg)
	}
}

// TestHandleConnection 验证并发服务器的连接处理：读到什么就转大写回什么。
// net.Pipe 提供一条内存里的同步双向管道，比真实 socket 更快、更确定。
func TestHandleConnection(t *testing.T) {
	server, client := net.Pipe()
	defer client.Close()

	var wg sync.WaitGroup
	wg.Add(1)
	go handleConnection(server, &wg)

	if _, err := client.Write([]byte("hello")); err != nil {
		t.Fatalf("写入失败: %v", err)
	}
	buf := make([]byte, 1024)
	n, err := client.Read(buf)
	if err != nil {
		t.Fatalf("读取失败: %v", err)
	}
	if got, want := string(buf[:n]), "HELLO"; got != want {
		t.Fatalf("转大写失败：got %q, want %q", got, want)
	}
	wg.Wait()
}

// TestLengthPrefixRoundTrip 验证「4 字节大端长度 + 数据」的边界协议，回显一致。
func TestLengthPrefixRoundTrip(t *testing.T) {
	server, client := net.Pipe()
	defer server.Close()

	go func() {
		var length uint32
		if err := binary.Read(server, binary.BigEndian, &length); err != nil {
			return
		}
		data := make([]byte, length)
		if _, err := io.ReadFull(server, data); err != nil {
			return
		}
		_ = binary.Write(server, binary.BigEndian, length)
		_, _ = server.Write(data)
	}()

	const msg = "长度前缀协议"
	length := uint32(len(msg))
	if err := binary.Write(client, binary.BigEndian, length); err != nil {
		t.Fatalf("写长度失败: %v", err)
	}
	if _, err := client.Write([]byte(msg)); err != nil {
		t.Fatalf("写数据失败: %v", err)
	}

	var respLength uint32
	if err := binary.Read(client, binary.BigEndian, &respLength); err != nil {
		t.Fatalf("读长度失败: %v", err)
	}
	resp := make([]byte, respLength)
	if _, err := io.ReadFull(client, resp); err != nil {
		t.Fatalf("读数据失败: %v", err)
	}
	if string(resp) != msg {
		t.Fatalf("回显不符：got %q, want %q", resp, msg)
	}
}

// TestSetReadDeadlineTimeout 验证读超时：对端不写数据时，过了 deadline 的 Read
// 必须返回带 Timeout() 的 net.Error，而不是一直阻塞。
func TestSetReadDeadlineTimeout(t *testing.T) {
	a, b := net.Pipe()
	defer a.Close()
	defer b.Close()

	if err := a.SetReadDeadline(time.Now().Add(20 * time.Millisecond)); err != nil {
		t.Fatalf("设置 deadline 失败: %v", err)
	}
	buf := make([]byte, 16)
	_, err := a.Read(buf)

	ne, ok := err.(net.Error)
	if !ok || !ne.Timeout() {
		t.Fatalf("期望超时错误，得到 %v", err)
	}
}
