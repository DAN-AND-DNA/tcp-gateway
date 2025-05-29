package main

import (
	"context"
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
	ctx, stop := signal.NotifyContext(context.Background(), syscall.SIGINT, syscall.SIGTERM)
	defer stop()

	flag.StringVar(&configs.Entry.Discovery.Type, "discovery_type", "redis", "Service discovery type")
	flag.StringVar(&configs.Entry.Discovery.RedisAddress, "discovery_redis_address", "127.0.0.1:6379", "Service discovery Redis address")
	flag.StringVar(&configs.Entry.Discovery.RedisPassword, "discovery_redis_password", "12345678", "Service discovery Redis password")
	flag.StringVar(&configs.Entry.Discovery.RedisRegisterKey, "discovery_redis_register_key", "gateway_registered_nodes", "Service discovery Redis registration key")
	flag.Uint64Var(&configs.Entry.NodeInfo.PublicTcpPort, "node_info_public_tcp_port", 18001, "TCP port for client-facing services")
	flag.Uint64Var(&configs.Entry.NodeInfo.PrivateHttpPort, "node_info_private_http_port", 18081, "HTTP port for service-facing RPC")
	flag.StringVar(&configs.Entry.NodeInfo.ServiceAPIURL, "node_info_service_api_url", "http://127.0.0.1:80", "API service Address")
	flag.Parse()

	// init config
	if err := configs.Init(); err != nil {
		utils.AlertPanic("configs init fail: " + err.Error())
	} else {
		strConfigs := configs.Show()
		log.Println(strConfigs)
		utils.AlertAuto(strConfigs)
	}

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
