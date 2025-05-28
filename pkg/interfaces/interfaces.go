package interfaces

import "gateway/pkg/configs"

type Msg struct {
	ID   uint16
	Body []byte
}

type Gateway interface {
	GenerateAgentUID() string
	RemoveAgent(string) Agent
}

type Agent interface {
	Close()
	Enable()
	Disable()
	Write(uint16, []byte) error
	Get(string) (any, bool)
	Set(string, any)
	Address() string
	GetSID() string
	GetCID() string
	GetDiscoveryConfig() configs.DiscoveryConfig
	GetNodeInfoConfig() configs.NodeInfoConfig
}

// service discovery
type RegiserCallback func() error
type GetMetaData func() (string, error)

// hook
type HookHeader func(Agent, []byte) error
type HookBody func(Agent, []byte, []byte) error

// plugin & middleware
type EndPoint func(Agent, Msg) error
type Middleware func(next EndPoint) EndPoint

// Add middleware to the middleware chain
func Use(middlewares []Middleware, next EndPoint) EndPoint {
	// Recursively apply middleware
	for i := len(middlewares) - 1; i >= 0; i-- {
		next = middlewares[i](next)
	}

	return next
}
