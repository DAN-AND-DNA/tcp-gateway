package plugins

import (
	"encoding/base64"
	"encoding/json"
	"errors"
	"fmt"
	"gateway/pkg/interfaces"
	"gateway/pkg/metric"
	"gateway/pkg/utils"
	"gateway/pkg/version"
	"io"
	"math"
	"net/http"
	"net/url"
	"runtime/debug"
	"strconv"
	"strings"
	"sync/atomic"
	"time"
)

var (
	PluginsMain = interfaces.Use([]interfaces.Middleware{RateLimitMiddle, ConcurrentMiddle, RateLimitEndMiddle, StressTest, LogMiddle}, ForwadHttp)

	ErrServiceAPIReturn4xx = errors.New("service api return status code 4xx")
	ErrServiceAPIReturn5xx = errors.New("service api return status code 5xx")
)

type LuaMsg struct {
	SequenceID uint32 `json:"sequenceID"`
	ServerID   string `json:"serverID"`
	ConnID     string `json:"connID"`
	MsgID      uint16 `json:"msgID"`
	Bytes      string `json:"bytes"`
}

func RecoverMiddle(next interfaces.EndPoint) interfaces.EndPoint {
	return func(agent interfaces.Agent, msg interfaces.Msg) error {
		defer func() {
			if r := recover(); r != nil {
				utils.AlertAuto(fmt.Sprintf("public tcp service panic, conn id: %s client ip: %s msg: %v err: %v stack: %s", agent.GetCID(), agent.Address(), msg, r, string(debug.Stack())))
			}
		}()

		return next(agent, msg)
	}
}

// Rate Limiting
func RateLimitMiddle(next interfaces.EndPoint) interfaces.EndPoint {
	return func(agent interfaces.Agent, msg interfaces.Msg) error {
		metric.CountPublicTCPRequest.Add(1)
		_, ok := agent.Get("concurrent")
		if !ok {
			agent.Set("concurrent", new(atomic.Int32))
		}

		v, ok := agent.Get("concurrent")
		if ok {
			if concurrent, ok := v.(*atomic.Int32); ok {
				if concurrent.Load() > 10 {
					// TODO Send client error message indicating flow control
					utils.AlertAuto(fmt.Sprintf("public tcp service speed limit, conn id: %s client ip: %s", agent.GetCID(), agent.Address()))
					return nil
				}

				concurrent.Add(1)
			}
		}

		// Add request sequence number
		if _, ok = agent.Get("sequenceID"); !ok {
			agent.Set("sequenceID", new(atomic.Uint32))
		}
		return next(agent, msg)
	}
}

// Concurrent Requests
func ConcurrentMiddle(next interfaces.EndPoint) interfaces.EndPoint {
	return func(agent interfaces.Agent, msg interfaces.Msg) error {
		go func() {
			defer func() {
				if r := recover(); r != nil {
					utils.AlertAuto(fmt.Sprintf("public tcp service panic, conn id: %s client ip: %s msg: %v err: %v stack: %s", agent.GetCID(), agent.Address(), msg, r, string(debug.Stack())))
				}
			}()
			metric.CountPublicHTTPRequest.Add(1)

			if err := next(agent, msg); err != nil {
				utils.AlertAuto(fmt.Sprintf("public tcp service error, conn id: %s client ip: %s err: %v msg: %v", agent.GetCID(), agent.Address(), err, msg))
				return
			}
		}()

		return nil
	}
}

// Rate Limiting Ended
func RateLimitEndMiddle(next interfaces.EndPoint) interfaces.EndPoint {
	return func(agent interfaces.Agent, msg interfaces.Msg) error {
		defer func() {
			if v, ok := agent.Get("concurrent"); ok {
				if concurrent_req, ok := v.(*atomic.Int32); ok {
					concurrent_req.Add(-1)
				}
			}
		}()

		return next(agent, msg)
	}
}

// Logging
func LogMiddle(next interfaces.EndPoint) interfaces.EndPoint {
	return func(agent interfaces.Agent, msg interfaces.Msg) error {
		start := time.Now()
		err := next(agent, msg)
		cost := time.Now().Sub(start).Milliseconds()

		// Statistics
		proto := strconv.FormatUint(uint64(msg.ID), 10)
		metric.P99PublicHTTPRequestLatency.In(proto, cost)

		if cost >= 4000 {
			utils.AlertAuto(fmt.Sprintf("public tcp service slow request, conn id: %s client ip: %s msg: %v cost: %d ms", agent.GetCID(), agent.Address(), msg, cost))
		}
		return err
	}
}

func StressTest(next interfaces.EndPoint) interfaces.EndPoint {
	return func(agent interfaces.Agent, msg interfaces.Msg) error {
		if version.ENV == version.EnvDev && msg.ID == 1001 {
			// Hijack Protocol 1001
			err := agent.Write(1001, msg.Body)
			if err != nil {
				return err
			}

			return nil
		}

		return next(agent, msg)
	}
}

// Forward via HTTP
func ForwadHttp(agent interfaces.Agent, msg interfaces.Msg) error {
	// Request queued for backend HTTP service
	client := http.Client{
		Timeout: 10 * time.Second,
	}

	v, ok := agent.Get("sequenceID")
	var newSeq uint32 = 0
	if ok {
		sequenceID, _ := v.(*atomic.Uint32)
		sequenceID.CompareAndSwap(math.MaxInt32-10000, 0)
		newSeq = sequenceID.Add(1)
	}

	reqMsg, _ := json.Marshal(LuaMsg{
		SequenceID: newSeq,
		ServerID:   agent.GetSID(),
		ConnID:     agent.GetCID(),
		MsgID:      msg.ID,
		Bytes:      base64.StdEncoding.EncodeToString(msg.Body),
	})

	nodeInfoConfig := agent.GetNodeInfoConfig()

	form := url.Values{}
	form.Set("proto_type", "stream")
	form.Set("msg_id", strconv.FormatUint(uint64(msg.ID), 10))
	form.Set("msg", string(reqMsg))

	req, err := http.NewRequest(http.MethodPost, nodeInfoConfig.ServiceAPIURL, strings.NewReader(form.Encode()))
	if err != nil {
		return err
	}
	req.Header.Set("Content-Type", "application/x-www-form-urlencoded")
	req.Header.Set("X-Real-IP", agent.Address())
	req.Header.Set("This-Is-Secret", agent.Address())

	resp, err := client.Do(req)
	if err != nil {
		return err
	}
	defer resp.Body.Close()

	bytesBody, err := io.ReadAll(resp.Body)
	if err != nil {
		return err
	}

	if resp.StatusCode != http.StatusOK {
		if resp.StatusCode >= http.StatusInternalServerError {
			return ErrServiceAPIReturn5xx
		}

		if resp.StatusCode >= http.StatusBadRequest {
			return ErrServiceAPIReturn4xx
		}

		return nil
	}

	respLuaMsg := LuaMsg{}
	json.Unmarshal(bytesBody, &respLuaMsg)

	if respLuaMsg.MsgID == 0 {
		//log.Println("gateway <== php: discard: ", string(bytesBody))
	} else {
		bytes, _ := base64.StdEncoding.DecodeString(respLuaMsg.Bytes)
		agent.Write(respLuaMsg.MsgID, bytes)
	}

	return nil
}
