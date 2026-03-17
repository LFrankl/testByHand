// 限时练习骨架：在 TCP 之上设计一个自定义应用层协议
//
// 协议格式（12 字节定长头部 + 变长 Body）：
//
//  +---------+---------+---------+---------+
//  | Magic   | Version |  Type   | Reserved|
//  | 2 bytes | 1 byte  | 1 byte  | 1 byte  |
//  +---------+---------+---------+---------+
//  | Seq             (4 bytes, big-endian) |
//  +---------------------------------------+
//  | BodyLen         (4 bytes, big-endian) |
//  +---------------------------------------+
//  | Body            (N bytes)             |
//  +---------------------------------------+
//
// 运行方式:
//   go run main.go -mode server
//   go run main.go -mode client -msg "hello"
package main

import (
	"bufio"
	"encoding/binary"
	"flag"
	"fmt"
	"io"
	"net"
	"os"
)

// -------- 协议常量 --------

const (
	Magic   uint16 = 0xCAFE
	Version uint8  = 0x01

	HeaderLen = 12 // 固定头部长度（字节）

	TypePing uint8 = 0x01
	TypePong uint8 = 0x02
	TypeData uint8 = 0x03
	TypeErr  uint8 = 0x04
)

// -------- 数据结构 --------

// Header 是协议固定头部。
type Header struct {
	Magic    uint16
	Version  uint8
	Type     uint8
	Reserved uint8 // 保留字段，填 0
	Seq      uint32
	BodyLen  uint32
}

// Message 是一个完整的协议消息。
type Message struct {
	Header
	Body []byte
}

// -------- 编解码 --------

// Encode 将 Message 序列化为字节流（大端序）。
// TODO: 实现该函数
func Encode(msg *Message) ([]byte, error) {
	panic("TODO")
}

// Decode 从 reader 中读取并反序列化一个 Message。
// 注意：需要先读固定头部，再按 BodyLen 读 Body，解决粘包问题。
// TODO: 实现该函数
func Decode(r io.Reader) (*Message, error) {
	panic("TODO")
}

// -------- 服务端 --------

// StartServer 在 addr 上监听 TCP 连接，对每条连接起一个 goroutine 处理。
// TODO: 实现该函数
func StartServer(addr string) error {
	panic("TODO")
}

// handleConn 处理单个客户端连接：
//   - 读取 Message → 根据 Type 分发处理 → 写回响应
//
// TODO: 实现该函数
func handleConn(conn net.Conn) {
	panic("TODO")
}

// processMessage 根据消息类型返回响应消息。
//   - TypePing  → TypePong（空 Body）
//   - TypeData  → TypeData（Echo Body）
//   - 其他      → TypeErr（Body="unknown type"）
//
// TODO: 实现该函数
func processMessage(req *Message) *Message {
	panic("TODO")
}

// -------- 客户端 --------

// SendMessage 连接 addr，发送一条 TypeData 消息，等待响应后打印并退出。
// TODO: 实现该函数
func SendMessage(addr, text string) error {
	panic("TODO")
}

// -------- main --------

func main() {
	mode := flag.String("mode", "server", "server | client")
	addr := flag.String("addr", "127.0.0.1:9000", "listen/connect address")
	msg := flag.String("msg", "hello", "message to send (client mode)")
	flag.Parse()

	var err error
	switch *mode {
	case "server":
		fmt.Printf("server listening on %s\n", *addr)
		err = StartServer(*addr)
	case "client":
		err = SendMessage(*addr, *msg)
	default:
		fmt.Fprintln(os.Stderr, "unknown mode:", *mode)
		os.Exit(1)
	}

	if err != nil {
		fmt.Fprintln(os.Stderr, "error:", err)
		os.Exit(1)
	}
}

// -------- 工具函数（已提供，无需修改）--------

// newBufReader 返回带缓冲的 Reader，方便读取 conn。
func newBufReader(conn net.Conn) *bufio.Reader {
	return bufio.NewReader(conn)
}

// byteOrder 统一使用大端序。
var byteOrder = binary.BigEndian
