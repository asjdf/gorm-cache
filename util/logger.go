package util

import (
	"context"
	"fmt"
	"time"
)

type LoggerInterface interface {
	SetIsDebug(debug bool)
	CtxInfo(ctx context.Context, format string, v ...interface{})
	CtxError(ctx context.Context, format string, v ...interface{})
}

type DefaultLogger struct {
	isDebug bool
}

func (l *DefaultLogger) SetIsDebug(d bool) {
	l.isDebug = d
}

func (l *DefaultLogger) CtxInfo(ctx context.Context, format string, v ...interface{}) {
	if l.isDebug {
		timePrefix := time.Now().Format("2006-01-02 15:04:05.999")
		fmt.Printf(timePrefix+" [INFO] "+format+"\n", v...)
	}
}

func (l *DefaultLogger) CtxError(ctx context.Context, format string, v ...interface{}) {
	if l.isDebug {
		timePrefix := time.Now().Format("2006-01-02 15:04:05.999")
		fmt.Printf(timePrefix+" [ERROR] "+format+"\n", v...)
	}
}
