package util

import (
	"encoding/json"
	"fmt"
	"math/rand"
	"time"
)

// JSONUtil JSON Utility Class
type JSONUtil struct{}

// ToJSON Convert object to JSON string
func (j *JSONUtil) ToJSON(v interface{}) (string, error) {
	bytes, err := json.Marshal(v)
	if err != nil {
		return "", err
	}
	return string(bytes), nil
}

// FromJSON Convert JSON string to object
func (j *JSONUtil) FromJSON(jsonStr string, v interface{}) error {
	return json.Unmarshal([]byte(jsonStr), v)
}

// PrettyJSON Format JSON output
func (j *JSONUtil) PrettyJSON(v interface{}) (string, error) {
	bytes, err := json.MarshalIndent(v, "", "  ")
	if err != nil {
		return "", err
	}
	return string(bytes), nil
}

// StringUtil String Utility Class
type StringUtil struct{}

// IsEmpty Check if string is empty
func (s *StringUtil) IsEmpty(str string) bool {
	return len(str) == 0
}

// IsNotEmpty Check if string is not empty
func (s *StringUtil) IsNotEmpty(str string) bool {
	return len(str) > 0
}

// DefaultIfEmpty If string is empty, return default value
func (s *StringUtil) DefaultIfEmpty(str, defaultValue string) string {
	if s.IsEmpty(str) {
		return defaultValue
	}
	return str
}

// ValidateEmail Simple email format validation
func (s *StringUtil) ValidateEmail(email string) bool {
	if s.IsEmpty(email) {
		return false
	}

	// Simple email format check
	atIndex := -1
	for i, char := range email {
		if char == '@' {
			if atIndex != -1 {
				return false // Multiple @ symbols
			}
			atIndex = i
		}
	}

	if atIndex == -1 || atIndex == 0 || atIndex == len(email)-1 {
		return false
	}

	return true
}

// NumberUtil Number Utility Class
type NumberUtil struct{}

// InRange Check if number is within specified range
func (n *NumberUtil) InRange(value, min, max int) bool {
	return value >= min && value <= max
}

// RandomInt Generate random integer within specified range
func (n *NumberUtil) RandomInt(min, max int) int {
	if min >= max {
		return min
	}
	return rand.Intn(max-min+1) + min
}

// TimeUtil Time Utility Class
type TimeUtil struct{}

// FormatTime Format time
func (t *TimeUtil) FormatTime(tm time.Time, layout string) string {
	return tm.Format(layout)
}

// FormatDateTime Format date and time
func (t *TimeUtil) FormatDateTime(tm time.Time) string {
	return tm.Format("2006-01-02 15:04:05")
}

// FormatDate Format date
func (t *TimeUtil) FormatDate(tm time.Time) string {
	return tm.Format("2006-01-02")
}

// ParseTime Parse time string
func (t *TimeUtil) ParseTime(timeStr, layout string) (time.Time, error) {
	return time.Parse(layout, timeStr)
}

// IsExpired Check if expired
func (t *TimeUtil) IsExpired(expireTime time.Time) bool {
	return time.Now().After(expireTime)
}

// GetCurrentTime Get current time
func (t *TimeUtil) GetCurrentTime() time.Time {
	return time.Now()
}

// GetCurrentTimestamp Get current timestamp (seconds)
func (t *TimeUtil) GetCurrentTimestamp() int64 {
	return time.Now().Unix()
}

// GetCurrentTimestampMs Get current timestamp (milliseconds)
func (t *TimeUtil) GetCurrentTimestampMs() int64 {
	return time.Now().UnixMilli()
}

// ValidationUtil Validation Utility Class
type ValidationUtil struct{}

// ValidateStruct Validate struct
func (v *ValidationUtil) ValidateStruct(obj interface{}) error {
	if obj == nil {
		return fmt.Errorf("object cannot be nil")
	}

	// More complex validation logic can be added here
	// For example, using reflection to check required fields

	return nil
}

// ValidateID Validate ID format
func (v *ValidationUtil) ValidateID(id string) error {
	if len(id) == 0 {
		return fmt.Errorf("ID cannot be empty")
	}

	if len(id) < 3 {
		return fmt.Errorf("ID is too short")
	}

	return nil
}

// ValidatePageParams Validate pagination parameters
func (v *ValidationUtil) ValidatePageParams(page, pageSize int) error {
	if page < 1 {
		return fmt.Errorf("page must be greater than 0")
	}

	if pageSize < 1 || pageSize > 100 {
		return fmt.Errorf("pageSize must be between 1 and 100")
	}

	return nil
}

// PaginationUtil Pagination Utility Class
type PaginationUtil struct{}

// Pagination Pagination information
type Pagination struct {
	Page     int `json:"page"`
	PageSize int `json:"page_size"`
	Total    int `json:"total"`
	Pages    int `json:"pages"`
}

// CalculatePagination Calculate pagination information
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

// GetOffset Get offset
func (p *PaginationUtil) GetOffset(page, pageSize int) int {
	if page <= 0 {
		page = 1
	}
	if pageSize <= 0 {
		pageSize = 10
	}
	return (page - 1) * pageSize
}