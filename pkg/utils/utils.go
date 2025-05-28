package utils

import (
	"fmt"
	"gateway/pkg/configs"
	"gateway/pkg/feishu"
	"gateway/pkg/version"
	"log"
	"runtime/debug"
	"strings"
)

func AlertLowFrequency(key string, detailMsg string) {
	key = strings.TrimSpace(key)
	if key == "" {
		return
	}

	nodeInfo := configs.GetNodeInfo()
	fullInfo := fmt.Sprintf("env: %s instance_name: %s instance_id: %s local_ip: %s uid: %s\n\n%s",
		nodeInfo.Env, nodeInfo.InstanceName, nodeInfo.InstanceID, nodeInfo.LocalIP, nodeInfo.ID, detailMsg)

	go func() {
		if version.ENV == version.EnvRelease {
			if err := feishu.Alert(key, fullInfo, "https://open.feishu.cn/open-apis/bot/v2/hook/xxxx", "xxxx"); err != nil {
				log.Println(err)
			}
		} else {
			if err := feishu.Alert(key, fullInfo, "https://open.feishu.cn/open-apis/bot/v2/hook/xxxx", "xxxx"); err != nil {
				log.Println(err)
			}
		}
	}()
}

func AlertAuto(detailMsg string) {
	nodeInfo := configs.GetNodeInfo()
	fullInfo := fmt.Sprintf("env: %s instance_name: %s instance_id: %s local_ip: %s uid: %s\n\n%s",
		nodeInfo.Env, nodeInfo.InstanceName, nodeInfo.InstanceID, nodeInfo.LocalIP, nodeInfo.ID, detailMsg)

	go func() {
		if version.ENV == version.EnvRelease {
			if err := feishu.Alert(detailMsg, fullInfo, "https://open.feishu.cn/open-apis/bot/v2/hook/xxxx", "xxxx"); err != nil {
				log.Println(err)
			}
		} else {
			if err := feishu.Alert(detailMsg, fullInfo, "https://open.feishu.cn/open-apis/bot/v2/hook/xxxx", "xxxx"); err != nil {
				log.Println(err)
			}
		}
	}()
}

func AlertPanic(detailMsg string) {
	nodeInfo := configs.GetNodeInfo()

	fullInfo := fmt.Sprintf("env: %s instance_name: %s instance_id: %s local_ip: %s uid: %s\n\ngo panic: %s\n\n%s",
		nodeInfo.Env, nodeInfo.InstanceName, nodeInfo.InstanceID, nodeInfo.LocalIP, nodeInfo.ID, detailMsg, string(debug.Stack()))

	if version.ENV == version.EnvRelease {
		if err := feishu.Alert(detailMsg, fullInfo, "https://open.feishu.cn/open-apis/bot/v2/hook/xxxx", "xxxx"); err != nil {
			log.Println(err)
		}
	} else {
		if err := feishu.Alert(detailMsg, fullInfo, "https://open.feishu.cn/open-apis/bot/v2/hook/xxxx", "xxxx"); err != nil {
			log.Println(err)
		}
	}

	panic(detailMsg)
}
