package utils

import (
	"path/filepath"
	"reflect"
	"runtime"

	"github.com/google/uuid"

	"github.com/longhorn/go-common-libs/types"
)

// Contains checks if the given slice contains the given value.
func Contains[T comparable](slice []T, value T) bool {
	for _, s := range slice {
		if s == value {
			return true
		}
	}
	return false
}

// GetFunctionName returns the <package>.<function name> of the given function.
func GetFunctionName(f interface{}) string {
	value := reflect.ValueOf(f)
	if value.Kind() != reflect.Func {
		return ""
	}
	return filepath.Base(runtime.FuncForPC(value.Pointer()).Name())
}

// GetFunctionPath returns the full path of the given function.
func GetFunctionPath(f interface{}) string {
	getFn := func() string {
		return runtime.FuncForPC(reflect.ValueOf(f).Pointer()).Name()
	}
	return GetFunctionInfo(f, getFn)
}

// GetFunctionInfo take a function as interface{} and a function for getting the info.
func GetFunctionInfo(f interface{}, getInfoFn func() string) string {
	value := reflect.ValueOf(f)
	if value.Kind() != reflect.Func {
		return ""
	}
	return getInfoFn()
}

// IsStringInSlice checks if the given string 'item' is present in the 'list' of strings.
func IsStringInSlice(list []string, item string) bool {
	for _, str := range list {
		if str == item {
			return true
		}
	}
	return false
}

// RandomID returns a random string with the specified length.
// If the specified length is less than or equal to 0, the default length will
// be used.
func RandomID(randomIDLenth int) string {
	if randomIDLenth <= 0 {
		randomIDLenth = types.RandomIDDefaultLength
	}

	uuid := UUID()

	if len(uuid) > randomIDLenth {
		uuid = uuid[:randomIDLenth]
	}
	return uuid
}

// UUID returns a random UUID string.
func UUID() string {
	return uuid.New().String()
}
