package main

import (
	"context"
	"encoding/binary"
	"flag"
	"fmt"
	"gateway/pkg/encoding"
	"io"
	"log"
	"net"
	"os/signal"
	"sync/atomic"
	"syscall"
	"time"
)

var StressType string
var TargetTCPPort, TargetHTTPPort uint

const (
	StressTypeGatewayTCP  = "gateway_tcp"  // 网关tcp性能
	StressTypeGatewayHTTP = "gateway_http" // 网关http性能
	StressTypeProxy       = "proxy"        // 转发性能
	StressTypePublish     = "publish"      // 推送性能
)

func main() {
	ctx, stop := signal.NotifyContext(context.Background(), syscall.SIGINT, syscall.SIGTERM)
	defer stop()

	// 开发环境配置
	flag.StringVar(&StressType, "stress_type", StressTypeProxy, "测试类型(gateway_tcp|gateway_http|proxy|publish)")
	flag.UintVar(&TargetTCPPort, "target_tcp_port", 18001, "要测试的tcp服务端口")
	flag.UintVar(&TargetHTTPPort, "target_http_port", 18081, "要测试的http服务端口")

	flag.Parse()

	if StressType == StressTypeGatewayTCP {
		stressGatewayTCP(5000)
	} else if StressType == StressTypeProxy {
		stressProxy(5000)
	} else {
		return
	}

	<-ctx.Done()
}

func stressGatewayTCP(connectionNums int) {
	// 发送 1001 协议
	ctx, stop := signal.NotifyContext(context.Background(), syscall.SIGINT, syscall.SIGTERM)
	defer stop()

	var connNum, errorNum, succNumTmp, succNum, sendNum, qps atomic.Uint64

	type EchoRequest struct {
		Message string
	}

	type EchoResponse struct {
		Message string
	}

	go func() {
		for {
			time.Sleep(5 * time.Second)
			// 计算qps
			lastNum := succNumTmp.Swap(0)
			qps.Store(lastNum / 5)
			log.Printf("connNum: %d errorNum: %d succNum: %d sendNum: %d qps: %d\n",
				connNum.Load(), errorNum.Load(), succNum.Load(), sendNum.Load(), qps.Load())
		}
	}()

	// 建立连接并发
	for i := 0; i < connectionNums; i++ {
		conn, err := net.Dial("tcp", fmt.Sprintf("43.138.221.243:%d", TargetTCPPort))
		if err != nil {
			log.Println(err)
			errorNum.Add(1)
			time.Sleep(50 * time.Millisecond)
			continue
		}
		connNum.Add(1)

		// 客户端逻辑
		go func() {
			<-ctx.Done()

			conn.Close()
		}()

		go func() {
			defer conn.Close()

			echoReq := EchoRequest{Message: "你好:" + time.Now().String()}
			msgHeader := make([]byte, 10)
			for {
				time.Sleep(2500 * time.Millisecond)
				data, err := encoding.Marshal(echoReq)
				if err != nil {
					log.Println("ma:", err)
					errorNum.Add(1)
					return
				}

				dataLen := len(data)
				sendBuf := make([]byte, dataLen+10)
				copy(sendBuf[10:], data)

				binary.LittleEndian.PutUint16(sendBuf[:2], 1001)
				binary.LittleEndian.PutUint32(sendBuf[2:6], uint32(dataLen+10))
				binary.LittleEndian.PutUint32(sendBuf[6:10], 0)

				if _, err := conn.Write(sendBuf); err != nil {
					log.Println("write:", err)
					errorNum.Add(1)
					return
				}

				sendNum.Add(1)
				conn.SetReadDeadline(time.Now().Add(60 * time.Second))
				n, err := io.ReadFull(conn, msgHeader)
				if err != nil {
					log.Println("read head:", err)
					// 发生错误
					errorNum.Add(1)
					return
				}

				if n != 10 {
					errorNum.Add(1)
					return
				}

				id := binary.LittleEndian.Uint16(msgHeader[:2])
				size := binary.LittleEndian.Uint32(msgHeader[2:6])
				bodySize := size - 10

				msgBody := make([]byte, bodySize)
				n, err = io.ReadFull(conn, msgBody)
				if err != nil {
					log.Println("read body:", err)
					errorNum.Add(1)
					return
				}

				if bodySize != uint32(n) {
					errorNum.Add(1)
					return
				}

				if id == 1001 {
					succNum.Add(1)
					succNumTmp.Add(1)
				}
			}
		}()

	}

	<-ctx.Done()

	time.Sleep(2 * time.Second)
}

func stressGatewayHTTP() {
}

func stressProxy(connectionNums int) {
	// 发送 6832 协议
	ctx, stop := signal.NotifyContext(context.Background(), syscall.SIGINT, syscall.SIGTERM)
	defer stop()

	var connNum, errorNum, succNumTmp, succNum, sendNum, qps atomic.Uint64

	type SyncTimeRequest struct {
	}

	type SyncTimeResponse struct {
		Timestamp int32
	}

	go func() {
		for {
			time.Sleep(5 * time.Second)
			// 计算qps
			lastNum := succNumTmp.Swap(0)
			qps.Store(lastNum / 5)
			log.Printf("connNum: %d errorNum: %d succNum: %d sendNum: %d qps: %d\n",
				connNum.Load(), errorNum.Load(), succNum.Load(), sendNum.Load(), qps.Load())
		}
	}()

	// 建立连接并发
	for i := 0; i < connectionNums; i++ {
		conn, err := net.Dial("tcp", fmt.Sprintf("43.138.221.243:%d", TargetTCPPort))
		if err != nil {
			log.Println(err)
			errorNum.Add(1)
			time.Sleep(50 * time.Millisecond)
			continue
		}
		connNum.Add(1)

		// 客户端逻辑
		go func() {
			<-ctx.Done()

			conn.Close()
		}()

		go func() {
			defer conn.Close()

			req := SyncTimeRequest{}
			msgHeader := make([]byte, 10)
			for {
				time.Sleep(2500 * time.Millisecond)
				data, err := encoding.Marshal(req)
				if err != nil {
					log.Println("ma:", err)
					errorNum.Add(1)
					return
				}

				dataLen := len(data)
				sendBuf := make([]byte, dataLen+10)
				copy(sendBuf[10:], data)

				binary.LittleEndian.PutUint16(sendBuf[:2], 6832)
				binary.LittleEndian.PutUint32(sendBuf[2:6], uint32(dataLen+10))
				binary.LittleEndian.PutUint32(sendBuf[6:10], 0)

				if _, err := conn.Write(sendBuf); err != nil {
					log.Println("write:", err)
					errorNum.Add(1)
					return
				}

				sendNum.Add(1)
				conn.SetReadDeadline(time.Now().Add(60 * time.Second))
				n, err := io.ReadFull(conn, msgHeader)
				if err != nil {
					log.Println("read head:", err)
					// 发生错误
					errorNum.Add(1)
					return
				}

				if n != 10 {
					errorNum.Add(1)
					return
				}

				id := binary.LittleEndian.Uint16(msgHeader[:2])
				size := binary.LittleEndian.Uint32(msgHeader[2:6])
				bodySize := size - 10

				msgBody := make([]byte, bodySize)
				n, err = io.ReadFull(conn, msgBody)
				if err != nil {
					log.Println("read body:", err)
					errorNum.Add(1)
					return
				}

				if bodySize != uint32(n) {
					errorNum.Add(1)
					return
				}

				if id == 7832 {
					succNum.Add(1)
					succNumTmp.Add(1)
				}
			}
		}()

	}

	<-ctx.Done()

	time.Sleep(2 * time.Second)
}

func stressPublish() {

}
