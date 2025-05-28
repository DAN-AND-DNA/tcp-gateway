package main

import (
	"context"
	"encoding/json"
	"flag"
	"gateway/pkg/configs"
	"gateway/pkg/discovery"
	"gateway/pkg/gateway"
	"gateway/pkg/metric"
	"gateway/pkg/utils"
	"log"
	"os/signal"
	"syscall"
	"time"
)

func main() {
	configs.InitLogFormat()
	ctx, stop := signal.NotifyContext(context.Background(), syscall.SIGINT, syscall.SIGTERM)
	defer stop()

	// Development environment configuration
	flag.StringVar(&configs.Entry.Dev.Discovery.Type, "discovery_type", "redis", "Service discovery type (for dev env")
	flag.StringVar(&configs.Entry.Dev.Discovery.RedisAddress, "discovery_redis_address", "127.0.0.1:6379", "Service discovery Redis address (for dev env)")
	flag.StringVar(&configs.Entry.Dev.Discovery.RedisPassword, "discovery_redis_password", "12345678", "Service discovery Redis password (for dev env)")
	flag.StringVar(&configs.Entry.Dev.Discovery.RedisRegisterKey, "discovery_redis_register_key", "gateway_registered_nodes", "Service discovery Redis registration key (for dev env)")
	flag.Uint64Var(&configs.Entry.Dev.NodeInfo.PublicTcpPort, "node_info_public_tcp_port", 18001, "TCP port for client-facing services (for dev env)")
	flag.Uint64Var(&configs.Entry.Dev.NodeInfo.PrivateHttpPort, "node_info_private_http_port", 18081, "HTTP port for service-facing RPC (for dev env)")
	flag.StringVar(&configs.Entry.Dev.NodeInfo.ServiceAPIURL, "node_info_service_api_url", "http://127.0.0.1:80", "API service Address (for dev env)")

	// Test and production environment configuration
	flag.StringVar(&configs.Entry.CosFilePath, "entry_cos_file_path", "", "Tencent Cloud COS path for the main configuration (used in debug and release environments)")

	flag.Parse()

	// Listen for configuration changes
	err := configs.Watch(60*time.Second, utils.AlertAuto)
	if err != nil {
		utils.AlertPanic("watch configs fail: " + err.Error())
	}

	nodeInfo := configs.GetNodeInfo()
	discoveryInfo := configs.GetDiscovery()

	// Send a success alert message to the Feishu group upon successful loading
	strNodeInfo, _ := json.MarshalIndent(nodeInfo, "", "	")
	discoveryInfo.RedisPassword = "***"
	strdiscoveryInfo, _ := json.MarshalIndent(discoveryInfo, "", "	")
	log.Println("load node info:\n" + string(strNodeInfo) + "\n\nload discovery info:\n" + string(strdiscoveryInfo))
	utils.AlertAuto("load node info:\n" + string(strNodeInfo) + "\n\nload discovery info:\n" + string(strdiscoveryInfo))

	// Report every 30 minutes
	metric.Report(1800)

	// Start the gateway service
	gateway := gateway.New()
	gateway.Run()
	defer gateway.Close()

	// TODO Perform a self-check and wait temporarily
	time.Sleep(2 * time.Second)

	// Register itself to service discovery
	if err := discovery.RegisterSelf(); err != nil {
		utils.AlertPanic("discovery register fail: " + err.Error())
	}
	log.Println("discovery register success")

	<-ctx.Done()
}
