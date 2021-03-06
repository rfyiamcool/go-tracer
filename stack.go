package tracer

import (
	"fmt"
	"runtime"
	"strings"
	"sync"
)

var (
	stackCache sync.Map
)

type stackEntry struct {
	frame runtime.Frame

	function string
	file     string
	line     int
}

func GetFuncSkip(skip int) string {
	return GetFunc(skip)
}

func GetFunc(ns ...int) string {
	var num = 0
	if len(ns) > 0 {
		num = ns[0]
	}

	_, _, fname := getCallerCache(1 + num)
	return fname
}

func GetCallerCache() string {
	file, line, fname := getCallerCache(1)
	return fmt.Sprintf("%s:%d:%s()", file, line, fname)
}

func getCallerCache(skip int) (file string, line int, funcName string) {
	// get stack pc offset
	rpc := [1]uintptr{}
	n := runtime.Callers(skip+2, rpc[:])
	if n < 1 {
		return
	}

	var (
		frame runtime.Frame
		pc    uintptr
	)

	pc = rpc[0]
	item, ok := stackCache.Load(pc)
	if ok {
		si := item.(stackEntry)
		return si.file, si.line, si.function
	}

	// get stack frame
	frame, _ = runtime.CallersFrames([]uintptr{pc}).Next()

	// get func name
	funcPC := runtime.FuncForPC(pc)
	if funcPC != nil {
		funcName = trimFuncname(funcPC.Name())
	}

	si := stackEntry{
		frame:    frame,
		function: funcName,
		file:     trimFilename(frame.File),
		line:     frame.Line,
	}
	stackCache.Store(pc, si)
	return si.file, si.line, si.function
}

func GetCaller() string {
	var (
		funcName = ""
		file     = ""
		line     = 0
		pc       uintptr
	)

	file, line, pc = getCaller(2)
	fullFuncName := runtime.FuncForPC(pc)
	if fullFuncName != nil {
		funcName = trimFuncname(fullFuncName.Name())
	}

	return fmt.Sprintf("%s:%d:%s()", file, line, funcName)
}

func getCaller(skip int) (string, int, uintptr) {
	pc, file, line, ok := runtime.Caller(skip)
	if !ok {
		return "", 0, pc
	}

	return trimFilename(file), line, pc
}

func trimFilename(file string) string {
	return trimString(file, '/', 2)
}

// trimClassFuncname git.github.com/ocean/internal/controller.(*obj).run2 => controller.(*obj).run2
func trimClassFuncname(name string) string {
	i := strings.LastIndex(name, "/")
	name = name[i+1:]
	return name
}

// trimFuncname git.github.com/ocean/internal/controller.(*obj).run2 => (*obj).run2
func trimFuncname(name string) string {
	i := strings.LastIndex(name, "/")
	name = name[i+1:]

	i = strings.Index(name, ".")
	return name[i+1:]
}

// trimString only retain shrot 2 level.
func trimString(str string, seq byte, level int) string {
	// get package name

	n := 0
	for i := len(str) - 1; i > 0; i-- {
		if str[i] == seq {
			n++
			if n >= 2 {
				return str[i+1:]
			}
		}
	}
	return str
}
