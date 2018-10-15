package expreduce

import (
	"bytes"
	"github.com/op/go-logging"
	"os"
	"runtime/debug"
	"sync"
)

var format = logging.MustStringFormatter(
	`%{color}%{time:15:04:05.000} %{callpath} ▶ %{id:03x}%{color:reset} %{message}`,
)
var goLoggingMutex = &sync.Mutex{}

type CASLogger struct {
	_log		*logging.Logger
	leveled		logging.LeveledBackend
	debugState	bool
	isProfiling	bool
}

func (this *CASLogger) Debugf(fmt string, args ...interface{}) {
	if this.debugState {
		//this._log.Debugf(this.Pre() + fmt, args...)
		this._log.Debugf(fmt, args...)
	}
}

func (this *CASLogger) Infof(fmt string, args ...interface{}) {
	if this.debugState {
		//this._log.Infof(this.Pre() + fmt, args...)
		this._log.Infof(fmt, args...)
	}
}

func (this *CASLogger) Errorf(fmt string, args ...interface{}) {
	this._log.Errorf(fmt, args...)
}

func (this *CASLogger) DebugOn(level logging.Level) {
	this.leveled.SetLevel(level, "")
	this.debugState = true
	this.SetProfiling(true)
}

func (this *CASLogger) DebugOff() {
	this.leveled.SetLevel(logging.ERROR, "")
	this.debugState = false
	this.SetProfiling(false)
}

func (this *CASLogger) DebugState() bool {
	return this.debugState
}

func (this *CASLogger) SetProfiling(profiling bool) {
	this.isProfiling = profiling
}

func (this *CASLogger) Pre() string {
	toReturn := ""
	if this.leveled.GetLevel("") != logging.ERROR {
		depth := (bytes.Count(debug.Stack(), []byte{'\n'}) - 15) / 2
		for i := 0; i < depth; i++ {
			toReturn += " "
		}
	}
	return toReturn
}

func (this *CASLogger) SetUpLogging() {
	// go-logging appears to not be thread safe, so we have a mutex when
	// configuring the logging.
	goLoggingMutex.Lock()
	this._log = logging.MustGetLogger("example")
	backend := logging.NewLogBackend(os.Stderr, "", 0)
	formatter := logging.NewBackendFormatter(backend, format)
	this.leveled = logging.AddModuleLevel(formatter)
	logging.SetBackend(this.leveled)
	goLoggingMutex.Unlock()
}
