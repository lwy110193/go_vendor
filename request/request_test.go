package request

import (
	"bytes"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"net/http/httptest"
	"sync"
	"testing"
	"time"
)

// MockResponse 模拟响应结构
type MockResponse struct {
	Message string `json:"message"`
	Code    int    `json:"code"`
}

// TestGet 测试GET请求功能
func TestGet(t *testing.T) {
	// 创建测试服务器
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		// 验证请求方法和路径
		if r.Method != "GET" {
			t.Errorf("Expected GET method, got %s", r.Method)
		}
		if r.URL.Path != "/test" {
			t.Errorf("Expected path /test, got %s", r.URL.Path)
		}

		// 验证查询参数
		if r.URL.Query().Get("param1") != "value1" {
			t.Errorf("Expected param1=value1, got %s", r.URL.Query().Get("param1"))
		}

		// 返回JSON响应
		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusOK)
		json.NewEncoder(w).Encode(MockResponse{Message: "success", Code: 200})
	}))
	defer server.Close()

	// 创建客户端
	client := NewClient(nil, nil)

	// 执行GET请求
	params := map[string]string{"param1": "value1"}
	resp, err := client.Get(server.URL+"/test", params, nil)
	if err != nil {
		t.Fatalf("Get request failed: %v", err)
	}

	// 验证响应
	if resp.StatusCode != http.StatusOK {
		t.Errorf("Expected status code 200, got %d", resp.StatusCode)
	}

	// 解析响应体
	var mockResp MockResponse
	if err := json.Unmarshal(resp.Body, &mockResp); err != nil {
		t.Fatalf("Failed to unmarshal response: %v", err)
	}
	if mockResp.Message != "success" || mockResp.Code != 200 {
		t.Errorf("Expected success response, got %+v", mockResp)
	}
}

// TestGetJSON 测试GET请求自动解析JSON功能
func TestGetJSON(t *testing.T) {
	// 创建测试服务器
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusOK)
		json.NewEncoder(w).Encode(MockResponse{Message: "success", Code: 200})
	}))
	defer server.Close()

	// 创建客户端
	client := NewClient(nil, nil)

	// 执行GET请求并解析JSON
	var mockResp MockResponse
	err := client.GetJSON(server.URL+"/test", nil, nil, &mockResp)
	if err != nil {
		t.Fatalf("GetJSON request failed: %v", err)
	}

	// 验证解析结果
	if mockResp.Message != "success" || mockResp.Code != 200 {
		t.Errorf("Expected success response, got %+v", mockResp)
	}
}

// TestPost 测试POST请求功能
func TestPost(t *testing.T) {
	// 创建测试服务器
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		// 验证请求方法和路径
		if r.Method != "POST" {
			t.Errorf("Expected POST method, got %s", r.Method)
		}

		// 验证请求体
		var body MockResponse
		if err := json.NewDecoder(r.Body).Decode(&body); err != nil {
			t.Fatalf("Failed to decode request body: %v", err)
		}
		if body.Message != "test" {
			t.Errorf("Expected message 'test', got %s", body.Message)
		}

		// 返回成功响应
		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusOK)
		json.NewEncoder(w).Encode(MockResponse{Message: "created", Code: 201})
	}))
	defer server.Close()

	// 创建客户端
	client := NewClient(nil, nil)

	// 执行POST请求
	requestBody := []byte(`{"message":"test","code":1}`)
	resp, err := client.Post(server.URL+"/test", requestBody, nil)
	if err != nil {
		t.Fatalf("Post request failed: %v", err)
	}

	// 验证响应
	if resp.StatusCode != http.StatusOK {
		t.Errorf("Expected status code 200, got %d", resp.StatusCode)
	}
}

// TestPostJSON 测试POST请求自动序列化和解析JSON功能
func TestPostJSON(t *testing.T) {
	// 创建测试服务器
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		// 验证请求体
		var body MockResponse
		if err := json.NewDecoder(r.Body).Decode(&body); err != nil {
			t.Fatalf("Failed to decode request body: %v", err)
		}
		if body.Message != "test" {
			t.Errorf("Expected message 'test', got %s", body.Message)
		}

		// 返回成功响应
		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusCreated)
		json.NewEncoder(w).Encode(MockResponse{Message: "created", Code: 201})
	}))
	defer server.Close()

	// 创建客户端
	client := NewClient(nil, nil)

	// 执行POST请求并自动解析
	var response MockResponse
	requestData := MockResponse{Message: "test", Code: 1}
	err := client.PostJSON(server.URL+"/test", requestData, nil, &response)
	if err != nil {
		t.Fatalf("PostJSON request failed: %v", err)
	}

	// 验证响应
	if response.Message != "created" || response.Code != 201 {
		t.Errorf("Expected created response, got %+v", response)
	}
}

// TestRetry 测试重试机制
func TestRetry(t *testing.T) {
	// 计数器，记录请求次数
	var count int
	var mu sync.Mutex

	// 创建测试服务器，前两次返回500错误，第三次返回成功
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		mu.Lock()
		count++
		currentCount := count
		mu.Unlock()

		if currentCount < 3 {
			w.WriteHeader(http.StatusInternalServerError)
		} else {
			w.Header().Set("Content-Type", "application/json")
			w.WriteHeader(http.StatusOK)
			json.NewEncoder(w).Encode(MockResponse{Message: "success", Code: 200})
		}
	}))
	defer server.Close()

	// 创建客户端，设置重试次数为2
	client := NewClient(&Config{
		RetryCount: 2,
		RetryDelay: 10 * time.Millisecond, // 短延迟，加快测试
		Timeout:    1 * time.Second,
	}, nil)

	// 执行请求
	var response MockResponse
	err := client.GetJSON(server.URL+"/test", nil, nil, &response)
	if err != nil {
		t.Fatalf("GetJSON with retry failed: %v", err)
	}

	// 验证请求被重试了3次
	mu.Lock()
	defer mu.Unlock()
	if count != 3 {
		t.Errorf("Expected 3 requests, got %d", count)
	}
}

// TestTimeout 测试超时设置
func TestTimeout(t *testing.T) {
	// 创建测试服务器，永远不响应
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		time.Sleep(2 * time.Second) // 睡眠2秒，超过超时时间
	}))
	defer server.Close()

	// 创建客户端，设置超时时间为100毫秒
	client := NewClient(&Config{
		Timeout: 100 * time.Millisecond,
	}, nil)

	// 执行请求，应该超时
	startTime := time.Now()
	_, err := client.Get(server.URL+"/test", nil, nil)
	elapsed := time.Since(startTime)

	// 验证超时错误
	if err == nil {
		t.Fatal("Expected timeout error, got nil")
	}

	// 验证执行时间接近超时时间但不超过
	if elapsed < 90*time.Millisecond || elapsed > 200*time.Millisecond {
		t.Errorf("Expected execution time around 100ms, got %v", elapsed)
	}
}

// TestUploadFile 测试单个文件上传功能
func TestUploadFile(t *testing.T) {
	// 创建测试服务器
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		// 检查请求方法
		if r.Method != "POST" {
			t.Errorf("Expected POST method, got %s", r.Method)
		}

		// 解析multipart表单
		err := r.ParseMultipartForm(10 << 20) // 10MB
		if err != nil {
			t.Fatalf("Failed to parse multipart form: %v", err)
		}

		// 验证普通表单字段
		if r.FormValue("description") != "test file" {
			t.Errorf("Expected description 'test file', got %s", r.FormValue("description"))
		}

		// 获取上传的文件
		file, header, err := r.FormFile("file")
		if err != nil {
			t.Fatalf("Failed to get file: %v", err)
		}
		defer file.Close()

		// 验证文件名
		if header.Filename != "test.txt" {
			t.Errorf("Expected filename 'test.txt', got %s", header.Filename)
		}

		// 读取文件内容
		content, err := io.ReadAll(file)
		if err != nil {
			t.Fatalf("Failed to read file content: %v", err)
		}

		// 验证文件内容
		if string(content) != "test file content" {
			t.Errorf("Expected content 'test file content', got %s", string(content))
		}

		// 返回成功响应
		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusOK)
		json.NewEncoder(w).Encode(MockResponse{Message: "file uploaded", Code: 200})
	}))
	defer server.Close()

	// 创建客户端
	client := NewClient(nil, nil)

	// 创建测试文件内容
	fileContent := bytes.NewBufferString("test file content")

	// 创建文件信息
	file := FileInfo{
		FieldName: "file",
		FileName:  "test.txt",
		Reader:    fileContent,
	}

	// 表单数据
	formData := map[string]string{
		"description": "test file",
	}

	// 执行文件上传
	resp, err := client.UploadFile(server.URL+"/upload", file, formData, nil)
	if err != nil {
		t.Fatalf("UploadFile request failed: %v", err)
	}

	// 验证响应
	if resp.StatusCode != http.StatusOK {
		t.Errorf("Expected status code 200, got %d", resp.StatusCode)
	}

	// 解析响应体
	var mockResp MockResponse
	if err := json.Unmarshal(resp.Body, &mockResp); err != nil {
		t.Fatalf("Failed to unmarshal response: %v", err)
	}
	if mockResp.Message != "file uploaded" || mockResp.Code != 200 {
		t.Errorf("Expected file uploaded response, got %+v", mockResp)
	}
}

// TestUploadFileJSON 测试单个文件上传并自动解析JSON响应
func TestUploadFileJSON(t *testing.T) {
	// 创建测试服务器
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		// 解析multipart表单
		r.ParseMultipartForm(10 << 20)

		// 返回成功响应
		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusOK)
		json.NewEncoder(w).Encode(MockResponse{Message: "file uploaded", Code: 200})
	}))
	defer server.Close()

	// 创建客户端
	client := NewClient(nil, nil)

	// 创建测试文件内容
	fileContent := bytes.NewBufferString("test content")

	// 创建文件信息
	file := FileInfo{
		FieldName: "file",
		FileName:  "test.json",
		Reader:    fileContent,
	}

	// 执行文件上传并自动解析JSON
	var response MockResponse
	err := client.UploadFileJSON(server.URL+"/upload", file, nil, nil, &response)
	if err != nil {
		t.Fatalf("UploadFileJSON request failed: %v", err)
	}

	// 验证响应
	if response.Message != "file uploaded" || response.Code != 200 {
		t.Errorf("Expected file uploaded response, got %+v", response)
	}
}

// TestUploadFiles 测试多个文件上传功能
func TestUploadFiles(t *testing.T) {
	// 创建测试服务器
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		// 解析multipart表单
		err := r.ParseMultipartForm(10 << 20)
		if err != nil {
			t.Fatalf("Failed to parse multipart form: %v", err)
		}

		// 验证接收到的文件数量
		files := r.MultipartForm.File["files"]
		if len(files) != 2 {
			t.Errorf("Expected 2 files, got %d", len(files))
		}

		// 检查文件名
		expectedFiles := map[string]bool{
			"file1.txt": true,
			"file2.txt": true,
		}

		for _, fileHeader := range files {
			if !expectedFiles[fileHeader.Filename] {
				t.Errorf("Unexpected filename: %s", fileHeader.Filename)
			}
			delete(expectedFiles, fileHeader.Filename)
		}

		if len(expectedFiles) > 0 {
			t.Errorf("Missing expected files: %v", expectedFiles)
		}

		// 返回成功响应
		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusCreated)
		json.NewEncoder(w).Encode(MockResponse{Message: "multiple files uploaded", Code: 201})
	}))
	defer server.Close()

	// 创建客户端
	client := NewClient(nil, nil)

	// 创建多个测试文件
	files := []FileInfo{
		{
			FieldName: "files",
			FileName:  "file1.txt",
			Reader:    bytes.NewBufferString("content1"),
		},
		{
			FieldName: "files",
			FileName:  "file2.txt",
			Reader:    bytes.NewBufferString("content2"),
		},
	}

	// 执行多文件上传
	resp, err := client.UploadFiles(server.URL+"/upload", files, nil, nil)
	if err != nil {
		t.Fatalf("UploadFiles request failed: %v", err)
	}

	// 验证响应
	if resp.StatusCode != http.StatusCreated {
		t.Errorf("Expected status code 201, got %d", resp.StatusCode)
	}
}

// TestUploadFilesJSON 测试多个文件上传并自动解析JSON响应
func TestUploadFilesJSON(t *testing.T) {
	// 创建测试服务器
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		// 解析multipart表单
		r.ParseMultipartForm(10 << 20)

		// 返回成功响应
		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusCreated)
		json.NewEncoder(w).Encode(MockResponse{Message: "multiple files uploaded", Code: 201})
	}))
	defer server.Close()

	// 创建客户端
	client := NewClient(nil, nil)

	// 创建多个测试文件
	files := []FileInfo{
		{
			FieldName: "files",
			FileName:  "file1.txt",
			Reader:    bytes.NewBufferString("content1"),
		},
		{
			FieldName: "files",
			FileName:  "file2.txt",
			Reader:    bytes.NewBufferString("content2"),
		},
	}

	// 执行多文件上传并自动解析JSON
	var response MockResponse
	err := client.UploadFilesJSON(server.URL+"/upload", files, nil, nil, &response)
	if err != nil {
		t.Fatalf("UploadFilesJSON request failed: %v", err)
	}

	// 验证响应
	if response.Message != "multiple files uploaded" || response.Code != 201 {
		t.Errorf("Expected multiple files uploaded response, got %+v", response)
	}
}

// TestHTTPSRequest 测试HTTPS请求功能
func TestHTTPSRequest(t *testing.T) {
	// 创建HTTPS测试服务器
	server := httptest.NewTLSServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		// 验证请求方法
		if r.Method != "GET" {
			t.Errorf("Expected GET method, got %s", r.Method)
		}

		// 返回JSON响应
		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusOK)
		json.NewEncoder(w).Encode(MockResponse{Message: "https success", Code: 200})
	}))
	defer server.Close()

	// 使用服务器的客户端，它已经配置了正确的证书验证
	httpClient := server.Client()

	// 创建我们的客户端，复用服务器的Transport（包含正确的证书验证配置）
	client := NewClient(&Config{
		Timeout: 30 * time.Second,
	}, nil)
	// 使用服务器客户端的Transport来确保证书验证正确
	if transport, ok := httpClient.Transport.(*http.Transport); ok {
		client.httpClient.Transport = transport
	}

	// 执行HTTPS请求
	var response MockResponse
	err := client.GetJSON(server.URL+"/test", nil, nil, &response)
	if err != nil {
		t.Fatalf("HTTPS GetJSON request failed: %v", err)
	}

	// 验证响应
	if response.Message != "https success" || response.Code != 200 {
		t.Errorf("Expected https success response, got %+v", response)
	}
}

// TestProxyPoolRoundRobin 测试轮询策略的代理池功能
func TestProxyPoolRoundRobin(t *testing.T) {
	// 创建目标服务器，它会记录来源代理的信息
	var proxyIdentifiers []string
	var mu sync.Mutex

	targetServer := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		// 从请求头获取代理标识符（在代理中添加）
		proxyID := r.Header.Get("X-Proxy-ID")
		mu.Lock()
		proxyIdentifiers = append(proxyIdentifiers, proxyID)
		mu.Unlock()

		// 返回成功响应
		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusOK)
		json.NewEncoder(w).Encode(MockResponse{Message: "success", Code: 200})
	}))
	defer targetServer.Close()

	// 创建两个模拟代理服务器
	proxy1 := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		// 转发请求到目标服务器，并添加代理标识符
		newReq, _ := http.NewRequest(r.Method, targetServer.URL+r.URL.Path, r.Body)
		for k, v := range r.Header {
			newReq.Header[k] = v
		}
		newReq.Header.Set("X-Proxy-ID", "proxy-1")

		client := &http.Client{}
		resp, err := client.Do(newReq)
		if err != nil {
			w.WriteHeader(http.StatusInternalServerError)
			return
		}
		defer resp.Body.Close()

		// 复制响应头和响应体
		for k, v := range resp.Header {
			w.Header()[k] = v
		}
		w.WriteHeader(resp.StatusCode)
		io.Copy(w, resp.Body)
	}))
	defer proxy1.Close()

	proxy2 := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		// 转发请求到目标服务器，并添加代理标识符
		newReq, _ := http.NewRequest(r.Method, targetServer.URL+r.URL.Path, r.Body)
		for k, v := range r.Header {
			newReq.Header[k] = v
		}
		newReq.Header.Set("X-Proxy-ID", "proxy-2")

		client := &http.Client{}
		resp, err := client.Do(newReq)
		if err != nil {
			w.WriteHeader(http.StatusInternalServerError)
			return
		}
		defer resp.Body.Close()

		// 复制响应头和响应体
		for k, v := range resp.Header {
			w.Header()[k] = v
		}
		w.WriteHeader(resp.StatusCode)
		io.Copy(w, resp.Body)
	}))
	defer proxy2.Close()

	// 创建客户端，配置代理池和轮询策略
	client := NewClient(&Config{
		ProxyURLs:         []string{proxy1.URL, proxy2.URL},
		ProxyPoolStrategy: "round-robin",
		Timeout:           5 * time.Second,
	}, nil)

	// 执行多次请求，验证轮询策略
	expectedOrder := []string{"proxy-1", "proxy-2", "proxy-1", "proxy-2"}
	for i := 0; i < 4; i++ {
		var response MockResponse
		err := client.GetJSON(targetServer.URL+"/test", nil, nil, &response)
		if err != nil {
			t.Fatalf("Request with proxy pool failed: %v", err)
		}
	}

	// 验证代理使用顺序
	mu.Lock()
	defer mu.Unlock()
	if len(proxyIdentifiers) != 4 {
		t.Errorf("Expected 4 requests, got %d", len(proxyIdentifiers))
		return
	}

	for i, expected := range expectedOrder {
		if proxyIdentifiers[i] != expected {
			t.Errorf("Expected proxy %s at position %d, got %s", expected, i, proxyIdentifiers[i])
		}
	}
}

// TestProxyPoolRandom 测试随机策略的代理池功能
func TestProxyPoolRandom(t *testing.T) {
	// 创建目标服务器，记录使用的代理
	var proxyIdentifiers []string
	var mu sync.Mutex

	targetServer := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		proxyID := r.Header.Get("X-Proxy-ID")
		mu.Lock()
		proxyIdentifiers = append(proxyIdentifiers, proxyID)
		mu.Unlock()

		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusOK)
		json.NewEncoder(w).Encode(MockResponse{Message: "success", Code: 200})
	}))
	defer targetServer.Close()

	// 创建三个模拟代理服务器
	proxies := []struct {
		ID     string
		Server *httptest.Server
	}{}

	for i := 1; i <= 3; i++ {
		proxyID := fmt.Sprintf("proxy-%d", i)
		server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			// 转发请求并添加代理标识符
			newReq, _ := http.NewRequest(r.Method, targetServer.URL+r.URL.Path, r.Body)
			for k, v := range r.Header {
				newReq.Header[k] = v
			}
			newReq.Header.Set("X-Proxy-ID", proxyID)

			client := &http.Client{}
			resp, err := client.Do(newReq)
			if err != nil {
				w.WriteHeader(http.StatusInternalServerError)
				return
			}
			defer resp.Body.Close()

			// 复制响应
			for k, v := range resp.Header {
				w.Header()[k] = v
			}
			w.WriteHeader(resp.StatusCode)
			io.Copy(w, resp.Body)
		}))
		proxies = append(proxies, struct {
			ID     string
			Server *httptest.Server
		}{proxyID, server})
		defer server.Close()
	}

	// 提取代理URLs
	proxyURLs := make([]string, len(proxies))
	for i, p := range proxies {
		proxyURLs[i] = p.Server.URL
	}

	// 创建客户端，配置随机策略
	client := NewClient(&Config{
		ProxyURLs:         proxyURLs,
		ProxyPoolStrategy: "random",
		Timeout:           5 * time.Second,
	}, nil)

	// 执行多次请求
	for i := 0; i < 10; i++ {
		var response MockResponse
		err := client.GetJSON(targetServer.URL+"/test", nil, nil, &response)
		if err != nil {
			t.Fatalf("Request with random proxy failed: %v", err)
		}
	}

	// 验证所有代理至少被使用一次
	mu.Lock()
	defer mu.Unlock()

	proxyUsage := make(map[string]bool)
	for _, id := range proxyIdentifiers {
		proxyUsage[id] = true
	}

	for _, p := range proxies {
		if !proxyUsage[p.ID] {
			t.Errorf("Proxy %s was never used", p.ID)
		}
	}
}
