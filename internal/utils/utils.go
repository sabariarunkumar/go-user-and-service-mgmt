package utils

import (
	"reflect"
	"strings"
)

// GenericResponse..
type GenericResponse struct {
	Message string `json:"error"`
}

// ErrorResponse...
type ErrorResponse struct {
	Error string `json:"error"`
}

// TokenResponse...
type TokenResponse struct {
	AccessToken        string `json:"access_token"`
	PassChangeRequired bool   `json:"password_change_required"`
}

// TempPassResponse...
type TempPassResponse struct {
	TempPass string `json:"temporary_password"`
}

// FormatGenericResponse formats the data into message field
func FormatGenericResponse(data string) GenericResponse {
	return GenericResponse{Message: data}
}

// FormatGenericResponse formats the data into message field
func FormatErrorResponse(data string) ErrorResponse {
	return ErrorResponse{Error: data}
}

// FormatGenericResponse formats token and password state
func FormatTokenResponse(token string, passChangeRequired bool) TokenResponse {
	return TokenResponse{AccessToken: token, PassChangeRequired: passChangeRequired}
}

// FormatTempPassResponse formats temporary password
func FormatTempPassResponse(pass string) TempPassResponse {
	return TempPassResponse{TempPass: pass}
}

// FieldTypeBinder maps fields to its type
type FieldTypeBinder map[string]reflect.Type

// Reflect Type of string
var String = reflect.TypeOf("")

// EnsureFieldsStrictlyExists check if input have same set and equal fields mentioned in FieldTypeBinder
func EnsureFieldsStrictlyExists(input map[string]interface{}, fieldTypeMap FieldTypeBinder) bool {
	if len(input) != len(fieldTypeMap) {
		return false
	}
	for field, fieldType := range fieldTypeMap {
		value, exists := input[field]
		if !exists {
			return false
		}
		if reflect.TypeOf(value) != fieldType {
			return false
		}
	}
	return true
}

// ConvertFieldTypeToString convert FieldTypeBinder to string
func ConvertFieldTypeToString(fieldTypeMap FieldTypeBinder) string {
	var sb strings.Builder
	for field, dataType := range fieldTypeMap {
		sb.WriteString(field)
		sb.WriteByte('[')
		sb.WriteString(dataType.String())
		sb.WriteByte(']')
		sb.WriteByte(',')
	}
	return strings.TrimSuffix(sb.String(), ",")
}
