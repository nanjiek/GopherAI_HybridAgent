package httpcontracts

// APIResponse 是统一响应结构。
type APIResponse struct {
	Code    int         `json:"code"`
	Message string      `json:"message"`
	Data    interface{} `json:"data,omitempty"`
}

// OK 返回成功响应。
func OK(data interface{}) APIResponse {
	return APIResponse{
		Code:    0,
		Message: "ok",
		Data:    data,
	}
}

// Err 返回失败响应。
func Err(code int, message string) APIResponse {
	return APIResponse{
		Code:    code,
		Message: message,
	}
}
