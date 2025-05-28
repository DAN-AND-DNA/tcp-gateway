package agent

import (
	"context"
	"encoding/binary"
	"errors"
	"fmt"
	"gateway/pkg/configs"
	"gateway/pkg/hot/hooks"
	"gateway/pkg/hot/plugins"
	"gateway/pkg/interfaces"
	"gateway/pkg/metric"
	"gateway/pkg/utils"
	"gateway/pkg/writer"
	"io"
	"net"
	"runtime/debug"
	"sync"
	"syscall"
	"time"
)

const (
	msgHeaderLen = 10
)

var (
	ErrWriteToSendQueueTimeout = errors.New("write to send queue timeout")
	ErrBadNetworkProtocol      = errors.New("bad network protocol")
	ErrIsClosed                = errors.New("is already closed")
)

type Agent struct {
	gateway interfaces.Gateway
	conn    net.Conn
	ctx     context.Context
	cancel  context.CancelFunc
	cid     string
	storage sync.Map
	address string
	disable bool
	w       *writer.Writer
	wd      chan struct{}
}

func New(gateway interfaces.Gateway, conn net.Conn, uid string) *Agent {
	agent := new(Agent)
	agent.conn = conn
	agent.ctx, agent.cancel = context.WithCancel(context.TODO())
	agent.cid = uid
	agent.gateway = gateway
	agent.w = writer.New(conn)
	agent.address = conn.RemoteAddr().String()
	agent.wd = make(chan struct{}, 5)

	return agent
}

func (agent *Agent) Run() {
	go func() {
		<-agent.ctx.Done()

		agent.finalizer()
	}()

	go func() {
		defer agent.cancel()

		// FIXME Only alert critical errors
		if err := agent.loopRead(); err != nil {
			if err == io.EOF || errors.Is(err, syscall.ECONNRESET) || errors.Is(err, net.ErrClosed) {
				return
			}

			var netError net.Error
			if errors.As(err, &netError) && netError.Timeout() {
				return
			}

			utils.AlertLowFrequency(err.Error(), fmt.Sprintf("agent: %s id: %s loop read exit: %v", agent.Address(), agent.GetCID(), err))
			return
		}
	}()

	go func() {
		defer agent.cancel()

		// FIXME Only alert critical errors
		if err := agent.loopWrite(); err != nil {
			if err == io.EOF || errors.Is(err, syscall.ECONNRESET) || errors.Is(err, net.ErrClosed) {
				return
			}

			utils.AlertLowFrequency(err.Error(), fmt.Sprintf("agent: %s id: %s loop write exit: %v", agent.Address(), agent.GetCID(), err))
			return
		}
	}()
}

func (agent *Agent) Close() {
	if agent == nil {
		return
	}

	agent.cancel()
}

func (agent *Agent) Disable() {
	if agent == nil {
		return
	}

	agent.disable = true
}

func (agent *Agent) Enable() {
	if agent == nil {
		return
	}

	agent.disable = false
}

func (agent *Agent) GetDiscoveryConfig() configs.DiscoveryConfig {
	return configs.GetDiscovery()
}

func (agent *Agent) GetNodeInfoConfig() configs.NodeInfoConfig {
	return configs.GetNodeInfo()
}

func (agent *Agent) GetSID() string {
	config := configs.GetNodeInfo()
	return config.ID
}

func (agent *Agent) GetCID() string {
	if agent == nil {
		return ""
	}

	return agent.cid
}

// Destruct
func (agent *Agent) finalizer() {
	defer func() {
		if err := recover(); err != nil {
			utils.AlertAuto(fmt.Sprintf("agent painc, id: %s ip: %s err: %v stack: %s", agent.GetCID(), agent.Address(), err, string(debug.Stack())))
		}
	}()

	// Cleanup
	metric.CountConnection.Add(-1)
	agent.gateway.RemoveAgent(agent.cid)
	agent.conn.Close()

	// Already expired, skip notification
	if agent.disable {
		return
	}

	agent.disable = true
	// Connection disconnected notification
	plugins.ForwadHttp(agent, interfaces.Msg{ID: 5006})
}

// Read coroutine logic
func (agent *Agent) loopRead() error {
	defer func() {
		if err := recover(); err != nil {
			utils.AlertAuto(fmt.Sprintf("agent painc, id: %s ip: %s err: %v stack: %s", agent.GetCID(), agent.Address(), err, string(debug.Stack())))
		}
	}()

	msgHeader := make([]byte, msgHeaderLen)
	for {
		// Read message timeout: 60 seconds
		agent.conn.SetReadDeadline(time.Now().Add(60 * time.Second))
		n, err := io.ReadFull(agent.conn, msgHeader)
		if err != nil {
			return err
		}

		if n != msgHeaderLen {
			return io.EOF
		}

		// Hook
		if err = hooks.HookHeader(agent, msgHeader); err != nil {
			return err
		}

		id := binary.LittleEndian.Uint16(msgHeader[:2])
		size := binary.LittleEndian.Uint32(msgHeader[2:6])
		bodySize := size - msgHeaderLen
		msgBody := make([]byte, bodySize)
		n, err = io.ReadFull(agent.conn, msgBody)
		if err != nil {
			return err
		}

		if bodySize != uint32(n) {
			return io.EOF
		}

		// Hook
		if err = hooks.HookBody(agent, msgHeader, msgBody); err != nil {
			return err
		}

		// If kicked, stop forwarding
		if !agent.disable {
			if err := plugins.PluginsMain(agent, interfaces.Msg{ID: id, Body: msgBody}); err != nil {
				return err
			}
		}
	}
}

// Write coroutine logic
func (agent *Agent) loopWrite() error {
	defer func() {
		if err := recover(); err != nil {
			utils.AlertAuto(fmt.Sprintf("agent painc, id: %s ip: %s err: %v stack: %s", agent.GetCID(), agent.Address(), err, string(debug.Stack())))
		}
	}()

	for {
		select {
		case _, ok := <-agent.wd:
			if ok {
				b, err := agent.w.Pop()
				if err != nil {
					return err
				}

				if len(b) == 0 {
					continue
				}

				agent.conn.SetWriteDeadline(time.Now().Add(60 * time.Second))
				if err := agent.w.Flush(b); err != nil {
					return err
				}
			}
		case <-agent.ctx.Done():
			return nil
		}
	}

}

func (agent *Agent) Write(msgID uint16, msg []byte) error {
	// First write to cache, then notify to ensure delivery
	err := agent.w.Write(msgID, msg, 0)
	if err != nil {
		return err
	}

	// If notification times out, discard (queue is full)
	tk := time.NewTicker(5 * time.Millisecond)
	defer tk.Stop()

	select {
	case <-tk.C:
	case agent.wd <- struct{}{}:
	}

	return nil
}

func (agent *Agent) Get(key string) (any, bool) {
	return agent.storage.Load(key)
}

func (agent *Agent) Set(key string, value any) {
	agent.storage.Store(key, value)
}

func (agent *Agent) Address() string {
	return agent.address
}
