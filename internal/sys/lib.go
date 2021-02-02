package sys

import (
	"fmt"
	"reflect"
	"runtime"

	"github.com/x-research-team/bus"
)

var trace = false

// Trace Трассирока вызовов
func Trace(t bool) {
	trace = t
}

func funcName(i interface{}) string {
	return runtime.FuncForPC(reflect.ValueOf(i).Pointer()).Name()
}

// Call Функция системного вызова биллинга
func Call(f interface{}, args ...interface{}) []interface{} {
	var result []interface{}
	if f == nil {
		bus.Error <- fmt.Errorf("[Command] %v does not exits", funcName(f))
		return result
	}
	fn := reflect.ValueOf(f)
	in := make([]reflect.Value, len(args))
	for i := range args {
		in[i] = reflect.ValueOf(args[i])
	}
	out := fn.Call(in)
	for _, value := range out {
		result = append(result, value.Interface())
	}
	if trace {
		bus.Info <- fmt.Sprintf("[Command] %v(%v) %v", funcName(f), args, result)
	}
	return result
}
