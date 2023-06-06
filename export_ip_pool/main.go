package main

import (
	"context"
	"fmt"
	"io/ioutil"
	"log"
	"net/http"
	"net/url"
	"runtime/debug"
	"strings"
	"sync"
	"time"

	"github.com/alecthomas/kong"
	"go.uber.org/zap"
	"go.uber.org/zap/zapcore"
	"golang.org/x/time/rate"
)

var cli struct {
	TargetUrl string `name:"url" help:"target url to request for export IP" default:"http://ifconfig.me/"`
	Qps       int    `name:"throughput" short:"t" help:"how many requests to send per second, if not set, then unlimited"`
	Proxy     struct {
		ProxyUser string   `name:"user" help:"the {user}:{password} for proxy" optional:"" env:"PROXY_USER" `
		Proxy     []string `name:"addr" short:"p" help:"request from a proxy, ie to get the full proxy export IP list" optional:""`
	} `embed:"" prefix:"proxy." optional:""`
	Test struct {
		Cases      int     `name:"cases" short:"c" default:"5" help:"how many cases to test"`
		RepeatRate float64 `name:"repeat-rate" short:"r" default:"1" help:"a repeat rate that will be considered should end, it's value between 0 to 1"`
	} `embed:"" prefix:"test." arg:""`
	Verbose int `name:"verbose" short:"v" type:"counter"`
}

func req(proxy string, auth string) (func() (string, error), error) {
	var c *http.Client

	if proxy != "" {
		u, err := createUrl("http", proxy, auth)
		if err != nil {
			return nil, err
		}

		c = &http.Client{
			Transport: &http.Transport{
				Proxy: func(r *http.Request) (*url.URL, error) {
					return u, nil
				},
			},
		}
	} else {
		c = http.DefaultClient
	}

	return func() (string, error) {
		resp, err := c.Get(cli.TargetUrl)
		if err != nil {
			return "", err
		}

		defer resp.Body.Close()

		allBytes, err := ioutil.ReadAll(resp.Body)
		if err != nil {
			return "", err
		}

		return string(allBytes), nil
	}, nil
}

func createUrl(schema, proxy, auth string) (*url.URL, error) {
	u, err := url.Parse(schema + "://" + proxy)
	if err != nil {
		return nil, err
	}
	if len(auth) != 0 {
		parts := strings.Split(auth, ":")
		if len(parts) != 2 {
			return nil, err
		}
		u.User = url.UserPassword(parts[0], parts[1])
	}
	return u, nil
}

func task(ctx context.Context, addr string, auth string, limiter *rate.Limiter) (m []string) {
	if addr != "" {
		ctx = CtxAddKvs(ctx, "proxy", addr)
	}
	f, err := req(addr, auth)
	if err != nil {
		LoggerOf(ctx).Info(fmt.Sprintf("create task for %s error: %v", addr, err))
		return nil
	}

	p := NewPool(cli.Test.Cases)

	defer func() {
		m = p.GetAll()
		LoggerOf(ctx).Info("task finished")
	}()

	for {
		err := limiter.Wait(context.Background())
		if err != nil {
			continue
		}

		result, err := f()
		if err != nil {
			LoggerOf(ctx).Info(fmt.Sprintf("task failed: %s, %v", addr, err))
			return
		}

		p.Put(result)
		r, c := p.FreshRate()
		repeat := 1 - r

		LoggerOf(ctx).Debug("task success", zap.Int("interval_count", c), zap.Float64("fresh_rate", r), zap.String("export", result))
		if c == cli.Test.Cases && repeat >= cli.Test.RepeatRate {
			break
		}
	}

	return
}

func main() {
	kong.Parse(&cli)
	prepareLogLevel()
	LoggerOf(context.Background()).Info("config", zap.Any("config", cli))

	limiter := prepareLimiter()

	i := 0
	concurrencyLimiter := make(chan struct{}, 1)

	m := make(map[string]struct{})
	resultList := make(chan []string, 10)
	cumulate := &sync.WaitGroup{}
	cumulate.Add(1)
	go WithRecover(func() {
		defer cumulate.Done()
		for r := range resultList {
			for _, k := range r {
				m[k] = struct{}{}
			}
		}
	})

	wg := &sync.WaitGroup{}
	var proxies []string
	if len(cli.Proxy.Proxy) == 0 {
		proxies = []string{""}
	} else {
		proxies = cli.Proxy.Proxy
	}
	for _, proxyAddr := range proxies {
		ctx := CtxAddKvs(context.Background(), "loop", i, "proxy", firstNotEmpty(proxyAddr, "no_proxy"))
		LoggerOf(ctx).Info("create task")
		concurrencyLimiter <- struct{}{}
		wg.Add(1)
		go WithRecover(func() {
			defer func() {
				LoggerOf(ctx).Info("task finished")
				<-concurrencyLimiter
			}()
			defer wg.Done()
			resultList <- task(ctx, proxyAddr, cli.Proxy.ProxyUser, limiter)
		})
	}
	wg.Wait()
	close(resultList)
	cumulate.Wait()

	LoggerOf(context.Background()).Info("task finished")

	for k := range m {
		fmt.Println(k)
	}
}

func firstNotEmpty(elem ...string) string {
	for _, s := range elem {
		if len(s) > 0 {
			return s
		}
	}
	return ""
}

func prepareLogLevel() {
	switch {
	case cli.Verbose >= 2:
		SetLevel(zapcore.DebugLevel)
	case cli.Verbose >= 1:
		SetLevel(zapcore.InfoLevel)
	default:
		SetLevel(zapcore.WarnLevel)
	}
}

func prepareLimiter() *rate.Limiter {
	var limit rate.Limit
	if cli.Qps == 0 {
		limit = rate.Inf
	} else {
		limit = rate.Every(time.Second / time.Duration(cli.Qps))
	}
	limiter := rate.NewLimiter(limit, 10)
	return limiter
}

func WithRecover(fn func()) {
	defer func() {
		if e := recover(); e != nil {
			log.Printf("panic: %v, goroutine: %s", e, debug.Stack())
		}
	}()

	fn()
}
