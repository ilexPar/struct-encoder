package pkg

import (
	"encoding/base64"
	"fmt"
	"reflect"
	"strconv"
	"strings"
	"time"
)

func randString() string {
	now := strconv.Itoa(int(time.Now().UnixNano()))
	return base64.StdEncoding.EncodeToString([]byte(now))
}

func randStringIfEmpty(val string) string {
	if val == "" {
		return randString()
	}
	return val
}

func getNativeRepresentationKey(native, alien reflect.Type, field string) string {
	nativePkg := randStringIfEmpty(native.PkgPath())
	nativeType := randStringIfEmpty(native.Name())
	alienPkg := randStringIfEmpty(alien.PkgPath())
	alienType := randStringIfEmpty(alien.Name())
	return fmt.Sprintf("%v:%v:%v~%v:%v", nativePkg, nativeType, field, alienPkg, alienType)
}

func getForeignTargetKey(alien reflect.Type, field string, path []string) string {
	alienPkg := randStringIfEmpty(alien.PkgPath())
	alienType := randStringIfEmpty(alien.Name())
	pathName := strings.Join(path, ".")
	return fmt.Sprintf("%v:%v:%v:%v", alienPkg, alienType, pathName, field)
}
