package utils

import "runtime"
import "log"

//PrintPanicStack print stack when panic
func PrintPanicStack(extras ...interface{}) {
	if x := recover(); x != nil {
		log.Println(x)
		log.Println(stack())
		// i := 0
		// funcName, file, line, ok := runtime.Caller(i)
		// for ok {
		// 	log.Printf("frame %v:[func:%v,file:%v,line:%v]\n", i, runtime.FuncForPC(funcName).Name(), file, line)
		// 	i++
		// 	funcName, file, line, ok = runtime.Caller(i)
		// }
		// for k := range extras {
		// 	log.Printf("EXRAS#%v DATA:%v\n", k, extras[k])
		// }
	}
}

func stack() string {
	var buf [2 << 10]byte
	return string(buf[:runtime.Stack(buf[:], false)])
}
