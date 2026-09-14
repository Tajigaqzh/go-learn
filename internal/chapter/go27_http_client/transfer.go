package go27_http_client

import (
	"bytes"
	"context"
	"crypto/sha256"
	"encoding/hex"
	"fmt"
	"io"
	"mime/multipart"
	"net/http"
)

// UploadResult 是上传接口的响应。
type UploadResult struct {
	Name   string `json:"name"`
	Size   int    `json:"size"`
	SHA256 string `json:"sha256"`
}

// UploadFile 以 multipart/form-data 上传一段内容。
//
// multipart 的边界串由 multipart.Writer 生成，Content-Type 必须用
// writer.FormDataContentType()，手写 "multipart/form-data" 会缺 boundary。
func (a *API) UploadFile(ctx context.Context, field, name string, content []byte) (UploadResult, error) {
	var body bytes.Buffer
	writer := multipart.NewWriter(&body)

	part, err := writer.CreateFormFile(field, name)
	if err != nil {
		return UploadResult{}, fmt.Errorf("创建上传字段: %w", err)
	}
	if _, err := part.Write(content); err != nil {
		return UploadResult{}, fmt.Errorf("写入上传内容: %w", err)
	}
	if err := writer.Close(); err != nil {
		return UploadResult{}, fmt.Errorf("结束 multipart 编码: %w", err)
	}

	resp, err := a.do(ctx, http.MethodPost, "/api/upload", &body, writer.FormDataContentType(), nil)
	if err != nil {
		return UploadResult{}, err
	}
	if err := checkStatus(resp); err != nil {
		return UploadResult{}, err
	}

	var result UploadResult
	if err := decodeJSON(resp, &result); err != nil {
		return UploadResult{}, fmt.Errorf("解析上传响应: %w", err)
	}
	return result, nil
}

// DownloadTo 把响应体流式写入 w，返回写入字节数与内容的 sha256。
//
// io.Copy + io.MultiWriter 的组合让内存占用与文件大小无关：内容一边落盘
// 一边喂给 sha256，不需要先 ReadAll 到内存里。
func (a *API) DownloadTo(ctx context.Context, w io.Writer, path string) (int64, string, error) {
	resp, err := a.do(ctx, http.MethodGet, path, nil, "", nil)
	if err != nil {
		return 0, "", err
	}
	defer resp.Body.Close()
	if err := checkStatus(resp); err != nil {
		return 0, "", err
	}

	hasher := sha256.New()
	written, err := io.Copy(io.MultiWriter(w, hasher), resp.Body)
	if err != nil {
		return written, "", fmt.Errorf("下载中断: %w", err)
	}
	return written, hex.EncodeToString(hasher.Sum(nil)), nil
}
