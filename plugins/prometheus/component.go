package prometheus

import (
	"context"
	"net/http"
	"time"

	grpcprometheus "github.com/grpc-ecosystem/go-grpc-prometheus"
	echoprometheus "github.com/labstack/echo-contrib/prometheus"
	"github.com/labstack/echo/v4"
	"github.com/labstack/echo/v4/middleware"
	"github.com/pkg/errors"
	"github.com/prometheus/client_golang/prometheus"
	"github.com/prometheus/client_golang/prometheus/collectors"
	"github.com/prometheus/client_golang/prometheus/promhttp"
	"go.uber.org/dig"

	"github.com/iotaledger/hive.go/core/app"
	"github.com/iotaledger/inx-api-core-v1/pkg/daemon"
)

func init() {
	Plugin = &app.Plugin{
		Component: &app.Component{
			Name:      "Prometheus",
			DepsFunc:  func(cDeps dependencies) { deps = cDeps },
			Params:    params,
			Provide:   provide,
			Configure: configure,
			Run:       run,
		},
		IsEnabled: func() bool {
			return ParamsPrometheus.Enabled
		},
	}
}

type dependencies struct {
	dig.In
	Echo           *echo.Echo
	PrometheusEcho *echo.Echo `name:"prometheusEcho"`
}

var (
	Plugin *app.Plugin
	deps   dependencies
)

func provide(c *dig.Container) error {

	type depsOut struct {
		dig.Out
		PrometheusEcho *echo.Echo `name:"prometheusEcho"`
	}

	return c.Provide(func() depsOut {
		e := echo.New()
		e.HideBanner = true
		e.Use(middleware.Recover())

		return depsOut{
			PrometheusEcho: e,
		}
	})
}

func configure() error {

	registry := registerMetrics()

	deps.PrometheusEcho.GET("/metrics", func(c echo.Context) error {

		handler := promhttp.HandlerFor(
			registry,
			promhttp.HandlerOpts{
				EnableOpenMetrics: true,
			},
		)

		if ParamsPrometheus.PromhttpMetrics {
			handler = promhttp.InstrumentMetricHandler(registry, handler)
		}

		handler.ServeHTTP(c.Response().Writer, c.Request())

		return nil
	})

	return nil
}

func run() error {
	return Plugin.Daemon().BackgroundWorker("Prometheus exporter", func(ctx context.Context) {
		Plugin.LogInfo("Starting Prometheus exporter ... done")

		go func() {
			Plugin.LogInfof("You can now access the Prometheus exporter using: http://%s/metrics", ParamsPrometheus.BindAddress)
			if err := deps.PrometheusEcho.Start(ParamsPrometheus.BindAddress); err != nil && !errors.Is(err, http.ErrServerClosed) {
				Plugin.LogWarnf("Stopped Prometheus exporter due to an error (%s)", err)
			}
		}()

		<-ctx.Done()
		Plugin.LogInfo("Stopping Prometheus exporter ...")

		shutdownCtx, shutdownCtxCancel := context.WithTimeout(context.Background(), 5*time.Second)
		defer shutdownCtxCancel()

		//nolint:contextcheck // false positive
		err := deps.PrometheusEcho.Shutdown(shutdownCtx)
		if err != nil {
			Plugin.LogWarn(err)
		}

		Plugin.LogInfo("Stopping Prometheus exporter ... done")
	}, daemon.PriorityStopPrometheus)
}

func registerMetrics() *prometheus.Registry {
	registry := prometheus.NewRegistry()

	if ParamsPrometheus.GoMetrics {
		registry.MustRegister(collectors.NewGoCollector())
	}

	if ParamsPrometheus.ProcessMetrics {
		registry.MustRegister(collectors.NewProcessCollector(collectors.ProcessCollectorOpts{}))
	}

	if ParamsPrometheus.INXMetrics {
		registry.MustRegister(grpcprometheus.DefaultClientMetrics)
	}

	if ParamsPrometheus.RestAPIMetrics {
		p := echoprometheus.NewPrometheus("iota_restapi", nil)
		for _, m := range p.MetricsList {
			registry.MustRegister(m.MetricCollector)
		}
		deps.Echo.Use(p.HandlerFunc)
	}

	return registry
}
