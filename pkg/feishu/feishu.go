package feishu

import (
	"bytes"
	"crypto/hmac"
	"crypto/sha256"
	"encoding/base64"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"strconv"
	"strings"
	"time"

	"github.com/hashicorp/golang-lru/v2/expirable"
)

var (
	expireCache = expirable.NewLRU[string, struct{}](30000, nil, 30*time.Second)
)

// Feishu Request
type Request struct {
	Timestamp string `json:"timestamp"`
	Sign      string `json:"sign,omitempty"`
	MsgType   string `json:"msg_type,omitempty"`
	Content   struct {
		Text string `json:"text,omitempty"`
	} `json:"content,omitempty"`
}

// Feishu Response
type Response struct {
	StatusCode    int    `json:"StatusCode,omitempty"`
	StatusMessage string `json:"StatusMessage,omitempty"`
	Code          int    `json:"code,omitempty"`
	Msg           string `json:"msg,omitempty"`
}

func Alert(key, text, webhookUrl, secret string) error {
	if strings.TrimSpace(key) == "" || strings.TrimSpace(text) == "" || webhookUrl == "" || secret == "" {
		return nil
	}

	if _, ok := expireCache.Get(key); ok {
		return nil
	}
	expireCache.Add(key, struct{}{})

	sign, timestamp, err := genSign(secret)
	if err != nil {
		return err
	}

	msgReq := Request{
		Timestamp: strconv.FormatInt(timestamp, 10),
		Sign:      sign,
		MsgType:   "text",
		Content: struct {
			Text string `json:"text,omitempty"`
		}{Text: text},
	}

	bytesReq, err := json.Marshal(msgReq)
	if err != nil {
		return err
	}

	client := &http.Client{Timeout: 10 * time.Second}
	resp, err := client.Post(webhookUrl, "application/json", bytes.NewBuffer(bytesReq))
	if err != nil {
		return err
	}
	defer resp.Body.Close()

	bytesResp, err := io.ReadAll(resp.Body)
	if err != nil {
		return err
	}

	msgResp := Response{}
	err = json.Unmarshal(bytesResp, &msgResp)
	if err != nil {
		return err
	}

	if msgResp.Code != 0 {
		return fmt.Errorf("feishu error: %v", msgResp.Msg)
	}

	return nil
}

func genSign(secret string) (string, int64, error) {
	timestamp := time.Now().Unix()
	strSign := fmt.Sprintf("%v\n%s", timestamp, secret)

	var data []byte
	h := hmac.New(sha256.New, []byte(strSign))
	_, err := h.Write(data)
	if err != nil {
		return "", 0, err
	}

	sign := base64.StdEncoding.EncodeToString(h.Sum(nil))

	return sign, timestamp, nil
}
