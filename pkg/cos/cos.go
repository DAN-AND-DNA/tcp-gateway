package cos

import (
	"bytes"
	"context"
	"io"
	"net/http"
	"net/url"
	"time"

	"github.com/tencentyun/cos-go-sdk-v5"
)

type COS struct {
	client *cos.Client
}

func New(URL, SecretID, SecretKey string) *COS {
	c := &COS{}
	u, _ := url.Parse(URL)
	b := &cos.BaseURL{BucketURL: u}
	c.client = cos.NewClient(b, &http.Client{
		Transport: &cos.AuthorizationTransport{
			SecretID:  SecretID,
			SecretKey: SecretKey,
		},
	})

	return c
}

func (c *COS) Put(path string, buf []byte) error {
	// Max 5 seconds
	uploadCosCtx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()

	// Block and wait here
	resp, err := c.client.Object.Put(uploadCosCtx, path, bytes.NewBuffer(buf), nil)
	if err != nil {
		return err
	}
	defer resp.Body.Close()

	return nil
}

func (c *COS) Head(path string) (string, error) {
	// Max 5 seconds
	uploadCosCtx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()

	// Block and wait here
	resp, err := c.client.Object.Head(uploadCosCtx, path, nil)
	if err != nil {
		return "", err
	}
	defer resp.Body.Close()
	etag := resp.Header.Get("ETag")

	return etag, nil
}

func (c *COS) Get(path string) ([]byte, string, error) {
	// Max 5 seconds
	uploadCosCtx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()

	// Block and wait here
	resp, err := c.client.Object.Get(uploadCosCtx, path, nil)
	if err != nil {
		return nil, "", err
	}
	defer resp.Body.Close()

	buf, err := io.ReadAll(resp.Body)
	if err != nil {
		return nil, "", err
	}

	etag := resp.Header.Get("ETag")

	return buf, etag, nil
}
