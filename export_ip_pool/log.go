package main

import (
	"context"
	"fmt"
	"os"

	"go.uber.org/zap"
	"go.uber.org/zap/zapcore"
)

type kvLists struct {
	values   []zap.Field
	previous *kvLists
}

func (list *kvLists) appendTo(t []zap.Field) []zap.Field {
	if list.previous != nil {
		t = list.previous.appendTo(t)
	}
	t = append(t, list.values...)
	return t
}

var logger *zap.Logger

func init() {
	logger, _ = zap.NewDevelopment()
}

func SetLevel(l zapcore.Level) {
	writeSyncer := zapcore.AddSync(os.Stderr)
	encoder := zapcore.NewConsoleEncoder(zap.NewDevelopmentEncoderConfig())
	core := zapcore.NewCore(encoder, writeSyncer, l)
	logger = zap.New(core)
}

type kvKey struct{}

func CtxAddKvs(ctx context.Context, kvs ...interface{}) context.Context {
	if len(kvs) == 0 {
		return ctx
	}

	var fields = make([]zap.Field, 0, len(kvs)/2+1)

	for i := 0; i < len(kvs); i += 2 {
		key := fmt.Sprint(kvs[i])
		val := fmt.Sprint(kvs[i+1])
		fields = append(fields, zap.String(key, val))
	}

	value := ctx.Value(kvKey{})
	previous, _ := value.(*kvLists)
	newList := &kvLists{
		values:   fields,
		previous: previous,
	}

	return context.WithValue(ctx, kvKey{}, newList)
}

func LoggerOf(ctx context.Context) *zap.Logger {
	return logger.With(getKvList(ctx)...)
}

func getKvList(ctx context.Context) []zap.Field {
	list, _ := ctx.Value(kvKey{}).(*kvLists)
	if list == nil {
		return nil
	}

	return list.appendTo(nil)
}
