package utils

import (
	"encoding/json"
	"fmt"
	"io/ioutil"
	"net/http"
	"os"
	"path/filepath"
	"sync"
	"time"
)

// HttpClient HTTP请求器（带缓存、速率限制、认证）
type HttpClient struct {
	cacheDir    string
	rateLimiter *RateLimiter
	client      *http.Client
	basicAuth   *BasicAuth
}

type BasicAuth struct {
	Username string
	Password string
}

type RateLimiter struct {
	mu           sync.Mutex
	requestCount int
	windowStart  time.Time
	maxPerMinute int
	backoffUntil time.Time
}

type CacheEntry struct {
	Data      json.RawMessage `json:"data"`
	UpdatedAt int64           `json:"updated_at"`
	ETag      string          `json:"etag"`
}

func NewRateLimiter(maxPerMinute int) *RateLimiter {
	return &RateLimiter{
		maxPerMinute: maxPerMinute,
		windowStart:  time.Now(),
	}
}


func (r *RateLimiter) Wait() {
	r.mu.Lock()
	defer r.mu.Unlock()

	now := time.Now()

	if now.Before(r.backoffUntil) {
		sleepTime := r.backoffUntil.Sub(now)
		r.mu.Unlock()
		time.Sleep(sleepTime)
		r.mu.Lock()
		now = time.Now()
	}

	if now.Sub(r.windowStart) >= time.Minute {
		r.requestCount = 0
		r.windowStart = now
	}

	if r.requestCount >= r.maxPerMinute {
		sleepTime := time.Minute - now.Sub(r.windowStart)
		r.mu.Unlock()
		time.Sleep(sleepTime)
		r.mu.Lock()
		r.requestCount = 0
		r.windowStart = time.Now()
	}

	r.requestCount++
}

func (r *RateLimiter) Backoff(seconds int) {
	r.mu.Lock()
	defer r.mu.Unlock()
	r.backoffUntil = time.Now().Add(time.Duration(seconds) * time.Second)
	r.maxPerMinute = r.maxPerMinute / 2
	if r.maxPerMinute < 10 {
		r.maxPerMinute = 10
	}
}

func NewHttpClient(cacheDir string, ratePerMinute int) *HttpClient {
	os.MkdirAll(cacheDir, os.ModePerm)
	return &HttpClient{
		cacheDir:    cacheDir,
		rateLimiter: NewRateLimiter(ratePerMinute),
		client:      &http.Client{Timeout: 30 * time.Second},
	}
}

func NewHttpClientWithAuth(cacheDir string, ratePerMinute int, username, password string) *HttpClient {
	c := NewHttpClient(cacheDir, ratePerMinute)
	c.basicAuth = &BasicAuth{Username: username, Password: password}
	return c
}


// FetchWithCache 从URL获取数据（带文件缓存）
func (c *HttpClient) FetchWithCache(url, cacheFile string, maxAge time.Duration, silent bool) ([]byte, error) {
	cachePath := filepath.Join(c.cacheDir, cacheFile)

	if info, err := os.Stat(cachePath); err == nil {
		if time.Since(info.ModTime()) < maxAge {
			if !silent {
				fmt.Printf("[*] GitHub 使用缓存: %s\n", cacheFile)
			}
			return ioutil.ReadFile(cachePath)
		}
	}

	if !silent {
		fmt.Printf("[*] GitHub 下载: %s\n", url)
	}

	resp, err := c.client.Get(url)
	if err != nil {
		if data, err := ioutil.ReadFile(cachePath); err == nil {
			if !silent {
				fmt.Println("[*] GitHub 网络错误，使用过期缓存")
			}
			return data, nil
		}
		return nil, err
	}
	defer resp.Body.Close()

	data, err := ioutil.ReadAll(resp.Body)
	if err != nil {
		return nil, err
	}

	ioutil.WriteFile(cachePath, data, 0644)
	return data, nil
}

// GetWithCache API请求（带缓存、ETag、认证）
func (c *HttpClient) GetWithCache(url, cacheKey string, headers map[string]string, maxAge time.Duration) ([]byte, bool, error) {
	cachePath := filepath.Join(c.cacheDir, cacheKey+".json")

	var cached CacheEntry
	if data, err := ioutil.ReadFile(cachePath); err == nil {
		json.Unmarshal(data, &cached)

		if time.Since(time.Unix(cached.UpdatedAt, 0)) < maxAge {
			return cached.Data, true, nil
		}

		if cached.ETag != "" {
			if headers == nil {
				headers = make(map[string]string)
			}
			headers["If-None-Match"] = cached.ETag
		}
	}

	c.rateLimiter.Wait()

	req, err := http.NewRequest("GET", url, nil)
	if err != nil {
		return cached.Data, len(cached.Data) > 0, err
	}

	for k, v := range headers {
		req.Header.Set(k, v)
	}

	if c.basicAuth != nil {
		req.SetBasicAuth(c.basicAuth.Username, c.basicAuth.Password)
	}

	resp, err := c.client.Do(req)
	if err != nil {
		return cached.Data, len(cached.Data) > 0, err
	}
	defer resp.Body.Close()

	if resp.StatusCode == 304 {
		cached.UpdatedAt = time.Now().Unix()
		if data, err := json.Marshal(cached); err == nil {
			ioutil.WriteFile(cachePath, data, 0644)
		}
		return cached.Data, true, nil
	}

	if resp.StatusCode == 429 {
		c.rateLimiter.Backoff(60)
		return cached.Data, len(cached.Data) > 0, fmt.Errorf("rate limited")
	}

	if resp.StatusCode != 200 {
		return cached.Data, len(cached.Data) > 0, fmt.Errorf("status: %d", resp.StatusCode)
	}

	body, err := ioutil.ReadAll(resp.Body)
	if err != nil {
		return cached.Data, len(cached.Data) > 0, err
	}

	newCache := CacheEntry{
		Data:      body,
		UpdatedAt: time.Now().Unix(),
		ETag:      resp.Header.Get("ETag"),
	}
	if data, err := json.Marshal(newCache); err == nil {
		ioutil.WriteFile(cachePath, data, 0644)
	}

	return body, false, nil
}
