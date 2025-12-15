package request

import (
	"bytes"
	"context"
	"crypto/tls"
	"crypto/x509"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"mime/multipart"
	"net"
	"net/http"
	"net/url"
	"os"
	"time"
)

// Config 请求配置结构体
type Config struct {
	Timeout            time.Duration     // 超时时间
	RetryCount         int               // 重试次数
	RetryDelay         time.Duration     // 重试间隔
	Headers            map[string]string // 全局请求头
	Context            context.Context   // 上下文，可用于取消请求
	ProxyURL           string            // 代理URL，如 "http://127.0.0.1:8080"
	InsecureSkipVerify bool              // 是否跳过TLS证书验证（不安全，仅用于测试环境）
	TLSConfig          *tls.Config       // 自定义TLS配置
	ClientCertFile     string            // 客户端证书文件路径
	ClientKeyFile      string            // 客户端私钥文件路径
	CAFile             string            // CA证书文件路径
}

// Response 响应结构体
type Response struct {
	StatusCode int         // 状态码
	Headers    http.Header // 响应头
	Body       []byte      // 响应体
}

// Client 请求客户端
type Client struct {
	config     *Config
	httpClient *http.Client
}

// NewClient 创建新的客户端
func NewClient(config *Config) *Client {
	if config == nil {
		config = &Config{
			Timeout:            30 * time.Second,
			RetryCount:         0,
			RetryDelay:         1 * time.Second,
			Headers:            make(map[string]string),
			Context:            context.Background(),
			ProxyURL:           "",
			InsecureSkipVerify: false,
		}
	} else {
		// 确保Headers和Context不为nil
		if config.Headers == nil {
			config.Headers = make(map[string]string)
		}
		if config.Context == nil {
			config.Context = context.Background()
		}
	}

	// 创建TLS配置
	tlsConfig := &tls.Config{}
	if config.TLSConfig != nil {
		// 如果提供了自定义TLS配置，则使用它
		tlsConfig = config.TLSConfig
	} else {
		// 否则创建默认配置
		tlsConfig = &tls.Config{
			InsecureSkipVerify: config.InsecureSkipVerify,
			MinVersion:         tls.VersionTLS12,
		}
	}

	// 加载客户端证书
	if config.ClientCertFile != "" && config.ClientKeyFile != "" {
		cert, err := tls.LoadX509KeyPair(config.ClientCertFile, config.ClientKeyFile)
		if err != nil {
			fmt.Printf("Warning: Failed to load client certificate: %v\n", err)
		} else {
			tlsConfig.Certificates = []tls.Certificate{cert}
		}
	}

	// 加载CA证书
	if config.CAFile != "" {
		caCert, err := os.ReadFile(config.CAFile)
		if err != nil {
			fmt.Printf("Warning: Failed to read CA certificate file: %v\n", err)
		} else {
			caCertPool := x509.NewCertPool()
			if ok := caCertPool.AppendCertsFromPEM(caCert); ok {
				tlsConfig.RootCAs = caCertPool
			} else {
				fmt.Printf("Warning: Failed to append CA certificate\n")
			}
		}
	}

	// 创建带超时配置的Transport
	transport := &http.Transport{
		MaxIdleConns:        100,
		MaxIdleConnsPerHost: 10,
		IdleConnTimeout:     90 * time.Second,
		TLSHandshakeTimeout: 10 * time.Second,
		TLSClientConfig:     tlsConfig,
		// 连接超时设置
		DialContext: (&net.Dialer{
			Timeout:   30 * time.Second,
			KeepAlive: 30 * time.Second,
		}).DialContext,
	}

	// 设置代理
	if config.ProxyURL != "" {
		proxyURL, err := url.Parse(config.ProxyURL)
		if err != nil {
			// panic(fmt.Sprintf("Warning: Invalid proxy URL: %v\n", err))
		} else {
			transport.Proxy = http.ProxyURL(proxyURL)
		}
	}

	// 创建http客户端
	httpClient := &http.Client{
		Transport: transport,
		Timeout:   config.Timeout,
	}

	return &Client{
		config:     config,
		httpClient: httpClient,
	}

}

// setRequestHeaders 设置请求头
func (c *Client) setRequestHeaders(req *http.Request) {
	// 设置全局请求头
	for key, value := range c.config.Headers {
		req.Header.Set(key, value)
	}

	// 默认设置Content-Type为application/json
	if req.Header.Get("Content-Type") == "" && req.Method != "GET" && req.Body != nil {
		req.Header.Set("Content-Type", "application/json")
	}
}

// parseResponse 解析响应
func (c *Client) parseResponse(resp *http.Response) (*Response, error) {
	if resp == nil {
		return nil, errors.New("response is nil")
	}
	defer resp.Body.Close()

	body, err := io.ReadAll(resp.Body)
	if err != nil {
		return nil, fmt.Errorf("failed to read response body: %w", err)
	}

	return &Response{
		StatusCode: resp.StatusCode,
		Headers:    resp.Header,
		Body:       body,
	}, nil
}

// 可重试的HTTP状态码
var retryableStatusCodes = map[int]bool{
	http.StatusRequestTimeout:      true,
	http.StatusTooEarly:            true,
	http.StatusTooManyRequests:     true,
	http.StatusInternalServerError: true,
	http.StatusBadGateway:          true,
	http.StatusServiceUnavailable:  true,
	http.StatusGatewayTimeout:      true,
}

// isRetryableError 判断错误是否可以重试
func isRetryableError(err error) bool {
	// 网络错误通常是可重试的
	var netErr net.Error
	if errors.As(err, &netErr) {
		// 超时错误总是可重试的
		if netErr.Timeout() {
			return true
		}
		// 其他网络错误也视为可重试
		return true
	}
	return false
}

// Do 执行HTTP请求的通用方法（带重试机制）
func (c *Client) Do(req *http.Request) (*Response, error) {
	// 设置请求头
	c.setRequestHeaders(req)

	var lastErr error
	var lastResp *Response
	retryCount := 0

	// 执行请求，支持重试
	for retryCount <= c.config.RetryCount {
		// 如果不是第一次尝试，输出重试日志
		if retryCount > 0 {
			fmt.Printf("Retrying request to %s, attempt %d/%d\n", req.URL, retryCount, c.config.RetryCount)
		}

		// 复制请求体，因为body只能读取一次
		var reqBody io.ReadCloser
		if req.Body != nil {
			bodyBytes, err := io.ReadAll(req.Body)
			if err != nil {
				lastErr = fmt.Errorf("failed to read request body: %w", err)
				retryCount++
				continue
			}
			req.Body.Close()
			reqBody = io.NopCloser(bytes.NewBuffer(bodyBytes))
			req.Body = reqBody
		}

		// 执行请求
		resp, err := c.httpClient.Do(req)

		// 处理错误
		if err != nil {
			lastErr = fmt.Errorf("request failed: %w", err)

			// 如果错误可重试且还可以重试，等待后继续
			if isRetryableError(err) && retryCount < c.config.RetryCount {
				retryCount++
				time.Sleep(c.config.RetryDelay)
				continue
			}
			break
		}

		// 处理响应
		parsedResp, parseErr := c.parseResponse(resp)
		if parseErr != nil {
			lastErr = parseErr
			retryCount++
			continue
		}

		// 如果状态码是可重试的，且还可以重试，则重试
		if retryableStatusCodes[parsedResp.StatusCode] && retryCount < c.config.RetryCount {
			lastResp = parsedResp
			retryCount++
			time.Sleep(c.config.RetryDelay)
			continue
		}

		// 成功响应，直接返回
		return parsedResp, nil
	}

	// 所有重试都失败了，返回最后一次的错误或最后一次的响应
	if lastResp != nil {
		return lastResp, lastErr
	}
	return nil, lastErr
}

// Get 执行GET请求
func (c *Client) Get(url string, params map[string]string, headers map[string]string) (*Response, error) {
	// 构建带查询参数的URL
	fullURL := url
	if len(params) > 0 {
		query := ""
		for key, value := range params {
			if query == "" {
				query = "?"
			} else {
				query += "\u0026"
			}
			query += fmt.Sprintf("%s=%s", key, value)
		}
		fullURL += query
	}

	// 创建带超时的上下文
	ctx, cancel := context.WithTimeout(c.config.Context, c.config.Timeout)
	defer cancel()

	// 创建请求
	req, err := http.NewRequestWithContext(ctx, "GET", fullURL, nil)
	if err != nil {
		return nil, fmt.Errorf("failed to create request: %w", err)
	}

	// 设置自定义请求头
	for key, value := range headers {
		req.Header.Set(key, value)
	}

	// 执行请求
	return c.Do(req)
}

// GetJSON 执行GET请求并自动解析JSON响应
func (c *Client) GetJSON(url string, params map[string]string, headers map[string]string, result interface{}) error {
	resp, err := c.Get(url, params, headers)
	if err != nil {
		return err
	}

	// 检查状态码
	if resp.StatusCode != http.StatusOK {
		return fmt.Errorf("unexpected status code: %d, body: %s", resp.StatusCode, string(resp.Body))
	}

	// 解析JSON
	if err := json.Unmarshal(resp.Body, result); err != nil {
		return fmt.Errorf("failed to unmarshal JSON: %w", err)
	}

	return nil
}

// Post 执行POST请求
func (c *Client) Post(url string, body []byte, headers map[string]string) (*Response, error) {
	// 创建请求体
	var bodyReader io.Reader
	if body != nil {
		bodyReader = bytes.NewReader(body)
	}

	// 创建带超时的上下文
	ctx, cancel := context.WithTimeout(c.config.Context, c.config.Timeout)
	defer cancel()

	// 创建请求
	req, err := http.NewRequestWithContext(ctx, "POST", url, bodyReader)
	if err != nil {
		return nil, fmt.Errorf("failed to create request: %w", err)
	}

	// 设置自定义请求头
	for key, value := range headers {
		req.Header.Set(key, value)
	}

	// 执行请求
	return c.Do(req)
}

// PostJSON 执行POST请求并自动序列化为JSON，同时解析响应
func (c *Client) PostJSON(url string, data interface{}, headers map[string]string, result interface{}) error {
	// 序列化请求数据
	body, err := json.Marshal(data)
	if err != nil {
		return fmt.Errorf("failed to marshal request data: %w", err)
	}

	// 执行POST请求
	resp, err := c.Post(url, body, headers)
	if err != nil {
		return err
	}

	// 检查状态码
	if resp.StatusCode != http.StatusOK && resp.StatusCode != http.StatusCreated {
		return fmt.Errorf("unexpected status code: %d, body: %s", resp.StatusCode, string(resp.Body))
	}

	// 如果需要解析响应结果
	if err := json.Unmarshal(resp.Body, result); err != nil {
		return fmt.Errorf("failed to unmarshal response: %w", err)
	}

	return nil
}

// PostForm 执行表单POST请求
func (c *Client) PostForm(url string, form map[string]string, headers map[string]string) (*Response, error) {
	// 构建表单数据
	var formData bytes.Buffer
	for key, value := range form {
		if formData.Len() > 0 {
			formData.WriteString("\u0026")
		}
		formData.WriteString(fmt.Sprintf("%s=%s", key, value))
	}

	// 如果没有提供Content-Type，设置为表单格式
	if headers == nil {
		headers = make(map[string]string)
	}
	if headers["Content-Type"] == "" {
		headers["Content-Type"] = "application/x-www-form-urlencoded"
	}

	// 执行POST请求
	return c.Post(url, formData.Bytes(), headers)
}

// FileInfo 文件信息结构体
type FileInfo struct {
	FieldName string    // 表单字段名
	FileName  string    // 文件名
	FilePath  string    // 文件路径
	Reader    io.Reader // 文件内容读取器
	Size      int64     // 文件大小
}

// UploadFile 上传单个文件
func (c *Client) UploadFile(url string, file FileInfo, formData map[string]string, headers map[string]string) (*Response, error) {
	// 创建multipart表单
	var b bytes.Buffer
	w := multipart.NewWriter(&b)

	// 添加普通表单字段
	for key, value := range formData {
		if err := w.WriteField(key, value); err != nil {
			return nil, fmt.Errorf("failed to write form field %s: %w", key, err)
		}
	}

	// 添加文件字段
	var fileReader io.Reader
	var err error

	if file.Reader != nil {
		fileReader = file.Reader
	} else if file.FilePath != "" {
		fileReader, err = os.Open(file.FilePath)
		if err != nil {
			return nil, fmt.Errorf("failed to open file %s: %w", file.FilePath, err)
		}
		defer fileReader.(io.Closer).Close()
	} else {
		return nil, errors.New("file Reader or FilePath must be provided")
	}

	// 创建文件字段
	part, err := w.CreateFormFile(file.FieldName, file.FileName)
	if err != nil {
		return nil, fmt.Errorf("failed to create form file: %w", err)
	}

	// 复制文件内容
	if _, err = io.Copy(part, fileReader); err != nil {
		return nil, fmt.Errorf("failed to copy file content: %w", err)
	}

	// 完成multipart表单
	if err = w.Close(); err != nil {
		return nil, fmt.Errorf("failed to close multipart writer: %w", err)
	}

	// 设置Content-Type
	if headers == nil {
		headers = make(map[string]string)
	}
	headers["Content-Type"] = w.FormDataContentType()

	// 创建请求
	req, err := http.NewRequestWithContext(c.config.Context, "POST", url, &b)
	if err != nil {
		return nil, fmt.Errorf("failed to create request: %w", err)
	}

	// 设置请求头
	c.setRequestHeaders(req)
	for key, value := range headers {
		req.Header.Set(key, value)
	}

	// 执行请求
	return c.Do(req)
}

// UploadFiles 上传多个文件
func (c *Client) UploadFiles(url string, files []FileInfo, formData map[string]string, headers map[string]string) (*Response, error) {
	// 创建multipart表单
	var b bytes.Buffer
	w := multipart.NewWriter(&b)

	// 添加普通表单字段
	for key, value := range formData {
		if err := w.WriteField(key, value); err != nil {
			return nil, fmt.Errorf("failed to write form field %s: %w", key, err)
		}
	}

	// 添加所有文件
	for i, file := range files {
		var fileReader io.Reader
		var err error

		if file.Reader != nil {
			fileReader = file.Reader
		} else if file.FilePath != "" {
			fileReader, err = os.Open(file.FilePath)
			if err != nil {
				return nil, fmt.Errorf("failed to open file %s at index %d: %w", file.FilePath, i, err)
			}
			defer fileReader.(io.Closer).Close()
		} else {
			return nil, fmt.Errorf("file %d: Reader or FilePath must be provided", i)
		}

		// 创建文件字段
		part, err := w.CreateFormFile(file.FieldName, file.FileName)
		if err != nil {
			return nil, fmt.Errorf("failed to create form file for %s: %w", file.FileName, err)
		}

		// 复制文件内容
		if _, err := io.Copy(part, fileReader); err != nil {
			return nil, fmt.Errorf("failed to copy content for file %s: %w", file.FileName, err)
		}
	}

	// 完成multipart表单
	if err := w.Close(); err != nil {
		return nil, fmt.Errorf("failed to close multipart writer: %w", err)
	}

	// 设置Content-Type
	if headers == nil {
		headers = make(map[string]string)
	}
	headers["Content-Type"] = w.FormDataContentType()

	// 创建请求
	req, err := http.NewRequestWithContext(c.config.Context, "POST", url, &b)
	if err != nil {
		return nil, fmt.Errorf("failed to create request: %w", err)
	}

	// 设置请求头
	c.setRequestHeaders(req)
	for key, value := range headers {
		req.Header.Set(key, value)
	}

	// 执行请求
	return c.Do(req)
}

// UploadFileJSON 上传单个文件并自动解析JSON响应
func (c *Client) UploadFileJSON(url string, file FileInfo, formData map[string]string, headers map[string]string, result interface{}) error {
	resp, err := c.UploadFile(url, file, formData, headers)
	if err != nil {
		return err
	}

	// 检查状态码
	if resp.StatusCode != http.StatusOK && resp.StatusCode != http.StatusCreated {
		return fmt.Errorf("unexpected status code: %d, body: %s", resp.StatusCode, string(resp.Body))
	}

	// 解析JSON
	if err := json.Unmarshal(resp.Body, result); err != nil {
		return fmt.Errorf("failed to unmarshal JSON: %w", err)
	}

	return nil
}

// UploadFilesJSON 上传多个文件并自动解析JSON响应
func (c *Client) UploadFilesJSON(url string, files []FileInfo, formData map[string]string, headers map[string]string, result interface{}) error {
	resp, err := c.UploadFiles(url, files, formData, headers)
	if err != nil {
		return err
	}

	// 检查状态码
	if resp.StatusCode != http.StatusOK && resp.StatusCode != http.StatusCreated {
		return fmt.Errorf("unexpected status code: %d, body: %s", resp.StatusCode, string(resp.Body))
	}

	// 解析JSON
	if err := json.Unmarshal(resp.Body, result); err != nil {
		return fmt.Errorf("failed to unmarshal JSON: %w", err)
	}

	return nil
}
