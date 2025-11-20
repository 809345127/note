package util

import (
	"encoding/json"
	"fmt"
	"math/rand"
	"time"
)

// JSONUtil JSON工具类
type JSONUtil struct{}

// ToJSON 将对象转换为JSON字符串
func (j *JSONUtil) ToJSON(v interface{}) (string, error) {
	bytes, err := json.Marshal(v)
	if err != nil {
		return "", err
	}
	return string(bytes), nil
}

// FromJSON 将JSON字符串转换为对象
func (j *JSONUtil) FromJSON(jsonStr string, v interface{}) error {
	return json.Unmarshal([]byte(jsonStr), v)
}

// PrettyJSON 格式化JSON输出
func (j *JSONUtil) PrettyJSON(v interface{}) (string, error) {
	bytes, err := json.MarshalIndent(v, "", "  ")
	if err != nil {
		return "", err
	}
	return string(bytes), nil
}

// StringUtil 字符串工具类
type StringUtil struct{}

// IsEmpty 检查字符串是否为空
func (s *StringUtil) IsEmpty(str string) bool {
	return len(str) == 0
}

// IsNotEmpty 检查字符串是否不为空
func (s *StringUtil) IsNotEmpty(str string) bool {
	return len(str) > 0
}

// DefaultIfEmpty 如果字符串为空，返回默认值
func (s *StringUtil) DefaultIfEmpty(str, defaultValue string) string {
	if s.IsEmpty(str) {
		return defaultValue
	}
	return str
}

// ValidateEmail 简单的邮箱格式验证
func (s *StringUtil) ValidateEmail(email string) bool {
	if s.IsEmpty(email) {
		return false
	}
	
	// 简单的邮箱格式检查
	atIndex := -1
	for i, char := range email {
		if char == '@' {
			if atIndex != -1 {
				return false // 多个@符号
			}
			atIndex = i
		}
	}
	
	if atIndex == -1 || atIndex == 0 || atIndex == len(email)-1 {
		return false
	}
	
	return true
}

// NumberUtil 数字工具类
type NumberUtil struct{}

// InRange 检查数字是否在指定范围内
func (n *NumberUtil) InRange(value, min, max int) bool {
	return value >= min && value <= max
}

// RandomInt 生成指定范围内的随机整数
func (n *NumberUtil) RandomInt(min, max int) int {
	if min >= max {
		return min
	}
	return rand.Intn(max-min+1) + min
}

// TimeUtil 时间工具类
type TimeUtil struct{}

// FormatTime 格式化时间
func (t *TimeUtil) FormatTime(tm time.Time, layout string) string {
	return tm.Format(layout)
}

// FormatDateTime 格式化日期时间
func (t *TimeUtil) FormatDateTime(tm time.Time) string {
	return tm.Format("2006-01-02 15:04:05")
}

// FormatDate 格式化日期
func (t *TimeUtil) FormatDate(tm time.Time) string {
	return tm.Format("2006-01-02")
}

// ParseTime 解析时间字符串
func (t *TimeUtil) ParseTime(timeStr, layout string) (time.Time, error) {
	return time.Parse(layout, timeStr)
}

// IsExpired 检查是否过期
func (t *TimeUtil) IsExpired(expireTime time.Time) bool {
	return time.Now().After(expireTime)
}

// GetCurrentTime 获取当前时间
func (t *TimeUtil) GetCurrentTime() time.Time {
	return time.Now()
}

// GetCurrentTimestamp 获取当前时间戳（秒）
func (t *TimeUtil) GetCurrentTimestamp() int64 {
	return time.Now().Unix()
}

// GetCurrentTimestampMs 获取当前时间戳（毫秒）
func (t *TimeUtil) GetCurrentTimestampMs() int64 {
	return time.Now().UnixMilli()
}

// ValidationUtil 验证工具类
type ValidationUtil struct{}

// ValidateStruct 验证结构体
func (v *ValidationUtil) ValidateStruct(obj interface{}) error {
	if obj == nil {
		return fmt.Errorf("object cannot be nil")
	}
	
	// 这里可以添加更复杂的验证逻辑
	// 例如使用反射来检查必填字段等
	
	return nil
}

// ValidateID 验证ID格式
func (v *ValidationUtil) ValidateID(id string) error {
	if len(id) == 0 {
		return fmt.Errorf("ID cannot be empty")
	}
	
	if len(id) < 3 {
		return fmt.Errorf("ID is too short")
	}
	
	return nil
}

// ValidatePageParams 验证分页参数
func (v *ValidationUtil) ValidatePageParams(page, pageSize int) error {
	if page < 1 {
		return fmt.Errorf("page must be greater than 0")
	}
	
	if pageSize < 1 || pageSize > 100 {
		return fmt.Errorf("pageSize must be between 1 and 100")
	}
	
	return nil
}

// PaginationUtil 分页工具类
type PaginationUtil struct{}

// Pagination 分页信息
type Pagination struct {
	Page     int `json:"page"`
	PageSize int `json:"page_size"`
	Total    int `json:"total"`
	Pages    int `json:"pages"`
}

// CalculatePagination 计算分页信息
func (p *PaginationUtil) CalculatePagination(total, page, pageSize int) *Pagination {
	if pageSize <= 0 {
		pageSize = 10
	}
	
	if page <= 0 {
		page = 1
	}
	
	pages := (total + pageSize - 1) / pageSize
	
	return &Pagination{
		Page:     page,
		PageSize: pageSize,
		Total:    total,
		Pages:    pages,
	}
}

// GetOffset 获取偏移量
func (p *PaginationUtil) GetOffset(page, pageSize int) int {
	if page <= 0 {
		page = 1
	}
	if pageSize <= 0 {
		pageSize = 10
	}
	return (page - 1) * pageSize
}