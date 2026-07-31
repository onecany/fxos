package coinank

import (
	"context"
	"errors"
	"fmt"
	"io"
	"net/http"
	"net/url"
	"fxos/httpclient"
	"time"
)

// CoinankClient coinank openapi url and apikey
type CoinankClient struct {
	Url    string
	Apikey string
}

// CoinankResponse coinank openapi common response
type CoinankResponse[T any] struct {
	Success bool   `json:"success"`
	Code    string `json:"code"`
	Data    T      `json:"data"`
}

// PageData coinank openapi pageData in response
type PageData[T any] struct {
	List       []T `json:"list"`
	Pagination struct {
		Current  int `json:"current"`
		Total    int `json:"total"`
		PageSize int `json:"pageSize"`
	} `json:"pagination"`
}

var HttpError error = errors.New("http client error")

// NewCoinankClient new coinank http client for coinank openapi
func NewCoinankClient(url, apikey string) *CoinankClient {
	return &CoinankClient{url, apikey}
}

// Get coinank openapi get request
func (c *CoinankClient) Get(ctx context.Context, path string, paramsMap map[string]string) (string, error) {
	data := url.Values{}
	for key, value := range paramsMap {
		data.Add(key, value)
	}
	fullURL := fmt.Sprintf("%s%s?%s", c.Url, path, data.Encode())
	resp, err := httpclient.Do(ctx, client, http.MethodGet, fullURL, nil, http.Header{
		"apikey": []string{c.Apikey},
	})
	if err != nil {
		return "", err
	}
	defer resp.Body.Close()
	body, err := io.ReadAll(resp.Body)
	if err != nil {
		return "", err
	}
	return string(body), nil
}

// Post coinank openapi post request
func (c *CoinankClient) Post(ctx context.Context, path string, data any) (string, error) {
	fullURL := fmt.Sprintf("%s%s", c.Url, path)
	resp, err := httpclient.DoJSON(ctx, client, http.MethodPost, fullURL, data, http.Header{
		"apikey": []string{c.Apikey},
	})
	if err != nil {
		return "", err
	}
	defer resp.Body.Close()
	body, err := io.ReadAll(resp.Body)
	if err != nil {
		return "", err
	}
	return string(body), nil
}

var client = httpclient.New(30 * time.Second)
