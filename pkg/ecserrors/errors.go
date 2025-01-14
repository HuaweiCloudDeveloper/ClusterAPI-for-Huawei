package ecserrors

import (
	"net/http"

	"github.com/huaweicloud/huaweicloud-sdk-go-v3/core/sdkerr"
)

const (
	ErrrorECSNotFound = "Ecs.0114"
)

// StatusCode returns the HTTP status for a particular error.
func StatusCode(err error) int {
	if t, ok := err.(*sdkerr.ServiceResponseError); ok {
		return t.StatusCode
	}

	return -1
}

// ErrorCode returns the error code for a particular error.
func ErrorCode(err error) string {
	if t, ok := err.(*sdkerr.ServiceResponseError); ok {
		return t.ErrorCode
	}

	return ""
}

func IsNotFound(err error) bool {
	if StatusCode(err) == http.StatusNotFound && ErrorCode(err) == ErrrorECSNotFound {
		return true
	}
	return false
}
