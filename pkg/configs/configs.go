package configs

import (
	"encoding/json"
	"errors"
	"fmt"
	"gateway/pkg/version"
	"io"
	"net/http"
	"strings"
	"sync"
)

var (
	Entry         = EntryConfig{}
	metaDataCache sync.Map

	ErrorNeedCosFilePathOfEntry = errors.New("need cos file path of entry")
	ErrorEmptyEntryConfig       = errors.New("empty entry config")
)

type DiscoveryConfig struct {
	Type             string `json:"type"`               // Service discovery type (redis or etcd)
	RedisRegisterKey string `json:"redis_register_key"` // Redis registration key
	RedisAddress     string `json:"redis_address"`      // Redis address
	RedisPassword    string `json:"redis_password"`     // Redis password
}

// Node info
type NodeInfoConfig struct {
	Env             string `json:"env"`               // Runtime environment
	PublicIP        string `json:"public_ip"`         // Public IP address
	LocalIP         string `json:"local_ip"`          // Local IP address
	InstanceName    string `json:"name"`              // Node instance name
	InstanceID      string `json:"instance_id"`       // Node instance ID
	ID              string `json:"id"`                // Unique service ID
	ServiceType     string `json:"service_type"`      // Service type
	ExpireTime      uint64 `json:"expire_time"`       // Heartbeat expiration time
	ConnectionNum   uint64 `json:"connection_num"`    // Connection count
	PublicTcpPort   uint64 `json:"public_tcp_port"`   // TCP port for client-facing services
	PrivateHttpPort uint64 `json:"private_http_port"` // HTTP port for internal service communication
	ServiceAPIURL   string `json:"service_api_url"`   // Service API URL
	BuildVersion    string `json:"build_version"`     // Build version
	GitVersion      string `json:"git_version"`       // Git commit
	MetricData      string `json:"metric_data"`       // Statistics data
}

type EntryConfig struct {
	Discovery DiscoveryConfig `json:"discovery"`
	NodeInfo  NodeInfoConfig  `json:"node_info"`
	Env       string          `json:"env"`
}

func Init() error {
	// Load node information on first startup
	if err := LoadNodeInfo(); err != nil {
		return err
	}

	// Check configuration
	if err := validate(); err != nil {
		return err
	}

	return nil
}

func validate() error {
	// TODO Validate configuration items

	return nil
}

func Show() string {
	nodeInfo := GetNodeInfo()
	discoveryInfo := GetDiscovery()

	strNodeInfo, _ := json.MarshalIndent(nodeInfo, "", "	")
	discoveryInfo.RedisPassword = "***"
	strdiscoveryInfo, _ := json.MarshalIndent(discoveryInfo, "", "	")
	return fmt.Sprintf("load node info:\n%s\n\nload discovery info:\n%s\n", string(strNodeInfo), string(strdiscoveryInfo))
}

// Set node information
func LoadNodeInfo() error {
	var err error
	publicIP, err := get("http://metadata.tencentyun.com/latest/meta-data/public-ipv4")
	if err != nil {
		return err
	}

	localIP, err := get("http://metadata.tencentyun.com/latest/meta-data/local-ipv4")
	if err != nil {
		return err
	}

	instanceName, err := get("http://metadata.tencentyun.com/latest/meta-data/instance-name")
	if err != nil {
		return err
	}

	instanceID, err := get("http://metadata.tencentyun.com/latest/meta-data/instance-id")
	if err != nil {
		return err
	}

	Entry.NodeInfo.Env = version.ENV
	Entry.NodeInfo.PublicIP = publicIP
	Entry.NodeInfo.LocalIP = localIP
	Entry.NodeInfo.InstanceName = instanceName
	Entry.NodeInfo.InstanceID = instanceID
	Entry.NodeInfo.ID = fmt.Sprintf("%s_%d_%d", instanceID, Entry.NodeInfo.PublicTcpPort, Entry.NodeInfo.PrivateHttpPort)
	Entry.NodeInfo.BuildVersion = version.BUILD_DATE
	Entry.NodeInfo.GitVersion = version.VERSION

	return nil
}

func get(url string) (string, error) {
	url = strings.TrimSpace(url)
	if url == "" {
		return "", nil
	}
	rawVal, ok := metaDataCache.Load(url)
	if ok {
		return rawVal.(string), nil
	}

	resp, err := http.Get(url)
	if err != nil {
		return "", err
	}
	defer resp.Body.Close()

	body, err := io.ReadAll(resp.Body)
	if err != nil {
		return "", err
	}
	val := string(body)
	metaDataCache.Store(url, val)

	return val, nil
}

func GetNodeInfo() NodeInfoConfig {
	return Entry.NodeInfo
}

func GetDiscovery() DiscoveryConfig {
	return Entry.Discovery
}
