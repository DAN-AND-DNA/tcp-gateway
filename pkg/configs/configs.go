package configs

import (
	"encoding/json"
	"errors"
	"fmt"
	"gateway/pkg/cos"
	"gateway/pkg/version"
	"io"
	"log"
	"net/http"
	"runtime/debug"
	"strings"
	"sync"
	"sync/atomic"
	"time"
)

var (
	Entry            = EntryConfig{}
	EntryConfigStore atomic.Value
	metaDataCache    sync.Map

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
	Dev struct {
		Discovery DiscoveryConfig `json:"discovery"`
		NodeInfo  NodeInfoConfig  `json:"node_info"`
	} `json:"dev,omitempty"`

	Debug struct {
		Discovery DiscoveryConfig `json:"discovery"`
		NodeInfo  NodeInfoConfig  `json:"node_info"`
	} `json:"debug,omitempty"`

	Release struct {
		Discovery DiscoveryConfig `json:"discovery"`
		NodeInfo  NodeInfoConfig  `json:"node_info"`
	} `json:"release,omitempty"`

	CosBasePath  string `json:"cos_base_path"`
	CosFilePath  string `json:"cos_file_path"`
	CosSecretID  string `json:"-"`
	CosSecretKey string `json:"-"`
	Etag         string `json:"etag"`
	UpdateTime   string `json:"update_time"`
	Env          string `json:"env"`
}

func Watch(interval time.Duration, alert func(string)) error {
	EntryConfigStore.Store(Entry)

	// Load node information on first startup
	if err := LoadNodeInfo(); err != nil {
		return err
	}
	EntryConfigStore.Store(Entry)

	// Load entry configuration on first startup
	if err := LoadEntry(); err != nil {
		return err
	}

	// Override node information
	if err := LoadNodeInfo(); err != nil {
		return err
	}

	// Check configuration
	if err := validate(); err != nil {
		return err
	}

	EntryConfigStore.Store(Entry)

	// Use COS configuration in production environment
	if version.ENV == version.EnvDev {
		return nil
	}

	// Listen for remote configuration changes
	go func() {
		defer func() {
			if r := recover(); r != nil {
				alert(fmt.Sprintf("update config panic: %v stack: %s", r, string(debug.Stack())))
			}
		}()

		c := cos.New(Entry.CosBasePath, Entry.CosSecretID, Entry.CosSecretKey)
		for {
			time.Sleep(interval)

			etag, err := c.Head(Entry.CosFilePath)
			if err != nil {
				alert("cos sdk head fail: " + err.Error())
			} else {
				if etag != "" && Entry.Etag != etag {
					if err := LoadEntry(); err != nil {
						alert("update entry config fail: " + err.Error())
						continue
					}

					if err := LoadNodeInfo(); err != nil {
						alert("update node info fail: " + err.Error())
						continue
					}

					if err := validate(); err != nil {
						alert("validate config fail: " + err.Error())
						continue
					}

					// 拷贝
					EntryConfigStore.Store(Entry)

					nodeInfo := GetNodeInfo()
					discoveryInfo := GetDiscovery()
					discoveryInfo.RedisPassword = "***"
					strNodeInfo, _ := json.MarshalIndent(nodeInfo, "", "	")
					strDiscoveryInfo, _ := json.MarshalIndent(discoveryInfo, "", "	")
					alert("update node info:\n" + string(strNodeInfo) + "\n\nupdate discovery info:\n" + string(strDiscoveryInfo))
				}
			}

		}
	}()

	return nil
}

func validate() error {
	// TODO Validate configuration items

	return nil
}

func InitLogFormat() {
	log.SetFlags(log.LstdFlags | log.Lmicroseconds)
}

// Load entry configuration
func LoadEntry() error {
	Entry.Env = version.ENV
	if version.ENV == version.EnvDebug {
		Entry.CosBasePath = "https://xxxx.myqcloud.com"
		Entry.CosSecretID = "xxxx"
		Entry.CosSecretKey = "xxxx"
	}

	if version.ENV == version.EnvRelease {
		Entry.CosBasePath = "https://xxxx.myqcloud.com"
		Entry.CosSecretID = "xxxx"
		Entry.CosSecretKey = "xxxx"
	}

	// Use only remote configuration in debug and release environments
	if version.ENV == version.EnvDebug || version.ENV == version.EnvRelease {
		Entry.CosFilePath = strings.TrimSpace(Entry.CosFilePath)
		if Entry.CosFilePath == "" {
			return ErrorNeedCosFilePathOfEntry
		}

		c := cos.New(Entry.CosBasePath, Entry.CosSecretID, Entry.CosSecretKey)
		data, etag, err := c.Get(Entry.CosFilePath)
		if err != nil {
			return err
		}

		Entry.Etag = etag
		Entry.UpdateTime = time.Now().Format(time.DateTime)

		if len(data) == 0 {
			return ErrorEmptyEntryConfig
		}

		if version.ENV == version.EnvDebug {
			if err = json.Unmarshal(data, &Entry.Debug); err != nil {
				return err
			}
		}

		if version.ENV == version.EnvRelease {
			if err = json.Unmarshal(data, &Entry.Release); err != nil {
				return err
			}
		}
	}

	// dev配置
	return nil
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

	if version.ENV == version.EnvDev {
		Entry.Dev.NodeInfo.Env = version.EnvDev
		Entry.Dev.NodeInfo.PublicIP = publicIP
		Entry.Dev.NodeInfo.LocalIP = localIP
		Entry.Dev.NodeInfo.InstanceName = instanceName
		Entry.Dev.NodeInfo.InstanceID = instanceID
		Entry.Dev.NodeInfo.ID = fmt.Sprintf("%s_%d_%d", instanceID, Entry.Dev.NodeInfo.PublicTcpPort, Entry.Dev.NodeInfo.PrivateHttpPort)
		Entry.Dev.NodeInfo.BuildVersion = version.BUILD_DATE
		Entry.Dev.NodeInfo.GitVersion = version.VERSION
	}

	if version.ENV == version.EnvDebug {
		Entry.Debug.NodeInfo.Env = version.EnvDebug
		Entry.Debug.NodeInfo.PublicIP = publicIP
		Entry.Debug.NodeInfo.LocalIP = localIP
		Entry.Debug.NodeInfo.InstanceName = instanceName
		Entry.Debug.NodeInfo.InstanceID = instanceID
		Entry.Debug.NodeInfo.ID = fmt.Sprintf("%s_%d_%d", instanceID, Entry.Debug.NodeInfo.PublicTcpPort, Entry.Debug.NodeInfo.PrivateHttpPort)
		Entry.Debug.NodeInfo.BuildVersion = version.BUILD_DATE
		Entry.Debug.NodeInfo.GitVersion = version.VERSION
	}

	if version.ENV == version.EnvRelease {
		Entry.Release.NodeInfo.Env = version.EnvRelease
		Entry.Release.NodeInfo.PublicIP = publicIP
		Entry.Release.NodeInfo.LocalIP = localIP
		Entry.Release.NodeInfo.InstanceName = instanceName
		Entry.Release.NodeInfo.InstanceID = instanceID
		Entry.Release.NodeInfo.ID = fmt.Sprintf("%s_%d_%d", instanceID, Entry.Release.NodeInfo.PublicTcpPort, Entry.Release.NodeInfo.PrivateHttpPort)
		Entry.Release.NodeInfo.BuildVersion = version.BUILD_DATE
		Entry.Release.NodeInfo.GitVersion = version.VERSION
	}

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
	config := EntryConfigStore.Load().(EntryConfig)

	if version.ENV == version.EnvDebug {
		return config.Debug.NodeInfo
	}

	if version.ENV == version.EnvRelease {
		return config.Release.NodeInfo
	}

	return config.Dev.NodeInfo
}

func GetDiscovery() DiscoveryConfig {
	config := EntryConfigStore.Load().(EntryConfig)

	if version.ENV == version.EnvDebug {
		return config.Debug.Discovery
	}

	if version.ENV == version.EnvRelease {
		return config.Release.Discovery
	}

	return config.Dev.Discovery
}
