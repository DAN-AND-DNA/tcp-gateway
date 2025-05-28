package gateway

import (
	"encoding/base64"
	"errors"
	"fmt"
	"gateway/pkg/agent"
	"gateway/pkg/configs"
	"gateway/pkg/hot/middlewares"
	"gateway/pkg/interfaces"
	"gateway/pkg/metric"
	"gateway/pkg/utils"
	"gateway/pkg/version"
	"math"
	"math/rand/v2"
	"net"
	"net/http"
	"strings"
	"sync"
	"sync/atomic"
	"time"

	"github.com/gin-gonic/gin"
)

var (
	ErrAgentUIDDuplicated = errors.New("agent uid is duplicated")
	ErrBadAgentUID        = errors.New("bad agent uid")
)

type Gateway struct {
	sync.RWMutex

	agents             map[string]interfaces.Agent
	agentUIDBase       atomic.Uint64
	privateHttpService *http.Server
	publicTcpService   net.Listener
}

func New() *Gateway {
	gateway := &Gateway{
		agents: make(map[string]interfaces.Agent),
	}
	return gateway
}

func (gateway *Gateway) Run() {
	if version.ENV != version.EnvDev {
		gin.SetMode(gin.ReleaseMode)
	}

	// Add plugins
	r := gin.New()
	r.Use(middlewares.CustomRecovery(), middlewares.LogMiddle())

	// Capture 404 errors
	r.NoRoute(func(ctx *gin.Context) {
		path := ""
		if ctx.Request.URL != nil {
			path = ctx.Request.URL.Path
		}
		method := ctx.Request.Method
		utils.AlertAuto(fmt.Sprintf("unkown private path: %s %s", path, method))
		ctx.JSON(http.StatusNotFound, gin.H{
			"code":    1,
			"message": "bad path",
		})
	})

	// Kick out (forcefully disconnect)
	r.POST("/agent/v1/close", func(ctx *gin.Context) {
		jsonMsg := struct {
			ConnID string `json:"connID"`
		}{}
		err := ctx.ShouldBindJSON(&jsonMsg)
		if err != nil {
			ctx.JSON(200, gin.H{
				"code":    1,
				"message": err,
			})
			return
		}

		agent := gateway.GetAgent(jsonMsg.ConnID)
		if agent == nil {
			ctx.JSON(200, gin.H{
				"code":    0,
				"message": "success",
			})
			return
		}

		agent.Disable()
		agent.Close()

		ctx.JSON(200, gin.H{
			"code":    0,
			"message": "success",
		})
	})

	r.POST("/agent/v1/send", func(ctx *gin.Context) {
		jsonMsg := struct {
			ConnID string `json:"connID"`
			Bytes  string `json:"bytes"`
			MsgID  uint16 `json:"msgID"`
		}{}
		err := ctx.ShouldBindJSON(&jsonMsg)
		if err != nil {
			ctx.JSON(200, gin.H{
				"code":    1,
				"message": err,
			})
			return
		}

		agent := gateway.GetAgent(jsonMsg.ConnID)
		if agent == nil {
			ctx.JSON(200, gin.H{
				"code":    0,
				"message": "success",
			})
			return
		}

		bytes, _ := base64.StdEncoding.DecodeString(jsonMsg.Bytes)
		agent.Write(jsonMsg.MsgID, bytes)
		ctx.JSON(200, gin.H{
			"code":    0,
			"message": "success",
		})
	})

	r.POST("/agent/v1/sendAndClose", func(ctx *gin.Context) {
		jsonMsg := struct {
			ConnID string `json:"connID"`
			Bytes  string `json:"bytes"`
			MsgID  uint16 `json:"msgID"`
		}{}
		err := ctx.ShouldBindJSON(&jsonMsg)
		if err != nil {
			ctx.JSON(200, gin.H{
				"code":    1,
				"message": err,
			})
			return
		}

		agent := gateway.GetAgent(jsonMsg.ConnID)
		if agent == nil {
			ctx.JSON(200, gin.H{
				"code":    0,
				"message": "success",
			})
			return
		}

		bytes, _ := base64.StdEncoding.DecodeString(jsonMsg.Bytes)
		agent.Write(jsonMsg.MsgID, bytes)

		go func() {
			// 先标记失效，不再转发消息
			agent.Disable()

			// 延迟关闭
			time.Sleep(1 * time.Second)
			agent.Close()
		}()

		ctx.JSON(200, gin.H{
			"code":    0,
			"message": "success",
		})
	})

	r.POST("/agent/v1/broadcast", func(ctx *gin.Context) {
		jsonMsg := struct {
			ConnIDs         []string `json:"connIDs"`
			Bytes           string   `json:"bytes"`
			MsgID           uint16   `json:"msgID"`
			DurationSeconds int      `json:"durationSeconds"` // Send duration (report time taken)
		}{}
		err := ctx.ShouldBindJSON(&jsonMsg)
		if err != nil {
			ctx.JSON(200, gin.H{
				"code":    1,
				"message": err,
			})
			return
		}

		if len(jsonMsg.ConnIDs) == 0 {
			ctx.JSON(200, gin.H{
				"code":    0,
				"message": "success",
			})
			return
		}

		bytes, _ := base64.StdEncoding.DecodeString(jsonMsg.Bytes)
		interval := jsonMsg.DurationSeconds * 1000 / len(jsonMsg.ConnIDs)

		go func() {
			for _, connId := range jsonMsg.ConnIDs {
				agent := gateway.GetAgent(connId)
				if agent == nil {
					continue
				}

				agent.Write(jsonMsg.MsgID, bytes)
				time.Sleep(time.Duration(interval) * time.Millisecond)
			}
		}()

		ctx.JSON(200, gin.H{
			"code":    0,
			"message": "success",
		})
	})

	// Start private HTTP service
	nodeInfoConfig := configs.GetNodeInfo()
	gateway.privateHttpService = &http.Server{
		Addr:         fmt.Sprintf("0.0.0.0:%d", nodeInfoConfig.PrivateHttpPort),
		Handler:      r,
		ReadTimeout:  20 * time.Second,
		WriteTimeout: 20 * time.Second,
	}

	go func() {
		defer func() {
			if err := recover(); err != nil {
				utils.AlertPanic(fmt.Sprintf("private http service listen panic: %v", err))
			}
		}()

		if err := gateway.privateHttpService.ListenAndServe(); err != nil {
			if errors.Is(err, http.ErrServerClosed) {
				return
			}

			panic(err)
		}
	}()

	// Start public TCP service
	var err error
	gateway.publicTcpService, err = net.Listen("tcp4", fmt.Sprintf("0.0.0.0:%d", nodeInfoConfig.PublicTcpPort))
	if err != nil {
		utils.AlertPanic(fmt.Sprintf("public tcp service listen fail: %v", err))
	}

	go func() {
		defer func() {
			if err := recover(); err != nil {
				utils.AlertPanic(fmt.Sprintf("public tcp service accept fail: %v", err))
			}
		}()

		for {
			newConn, err := gateway.publicTcpService.Accept()
			if err != nil {
				if errors.Is(err, net.ErrClosed) {
					return
				}
				continue
			}

			agentUID := gateway.GenerateAgentUID()
			agent := agent.New(gateway, newConn, agentUID)
			if err := gateway.AddAgent(agentUID, agent); err != nil {
				utils.AlertAuto(fmt.Sprintf("add agent: %s fail: %v", newConn.RemoteAddr(), err))

				newConn.Close()
				continue
			}

			metric.CountConnection.Add(1)
			agent.Run()
		}
	}()
}

func (gateway *Gateway) Close() {
	if gateway == nil {
		return
	}

	if gateway.publicTcpService != nil {
		gateway.publicTcpService.Close()
	}

	if gateway.privateHttpService != nil {
		gateway.privateHttpService.Close()
	}

	gateway.Lock()
	defer gateway.Unlock()

	for _, agent := range gateway.agents {
		agent.Close()
	}

	gateway.agents = nil
}

func (gateway *Gateway) GenerateAgentUID() string {
	gateway.agentUIDBase.CompareAndSwap(math.MaxUint64-1000, 0)

	uid := gateway.agentUIDBase.Add(1) // ID may be duplicated after restart
	now := time.Now().UnixNano()       // Avoid ID duplication after restart
	randomID := rand.Uint32N(9999999)  // Avoid ID duplication after restart

	return fmt.Sprintf("%d_%d_%d", uid, now, randomID)
}

func (gateway *Gateway) AddAgent(id string, agent *agent.Agent) error {
	id = strings.TrimSpace(id)
	if id == "" {
		return ErrBadAgentUID
	}

	gateway.Lock()
	defer gateway.Unlock()

	_, ok := gateway.agents[id]
	if ok {
		return ErrAgentUIDDuplicated
	}

	gateway.agents[id] = agent

	return nil
}

func (gateway *Gateway) RemoveAgent(id string) interfaces.Agent {
	if strings.TrimSpace(id) == "" {
		return nil
	}

	gateway.Lock()
	defer gateway.Unlock()

	agent, ok := gateway.agents[id]
	if !ok {
		return nil
	}

	delete(gateway.agents, id)

	return agent
}

func (gateway *Gateway) GetAgent(id string) interfaces.Agent {
	if strings.TrimSpace(id) == "" {
		return nil
	}

	gateway.RLock()
	defer gateway.RUnlock()

	agent, ok := gateway.agents[id]
	if ok {
		return agent
	}

	return nil
}

func (gateway *Gateway) GetAgents() map[string]interfaces.Agent {
	if gateway == nil {
		return nil
	}

	gateway.RLock()
	defer gateway.RUnlock()

	ret := make(map[string]interfaces.Agent, len(gateway.agents))
	for id, agent := range gateway.agents {
		ret[id] = agent
	}

	return ret
}
