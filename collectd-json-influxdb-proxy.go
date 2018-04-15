package main

import (
	stdContext "context"
	stdExpvar "expvar"
	"flag"
	"math"
	"net/http"
	"os"
	"os/signal"
	"runtime"
	"syscall"
	"time"

	"github.com/coreos/go-systemd/daemon"

	"github.com/gin-contrib/expvar"
	"github.com/gin-gonic/gin"

	client "github.com/influxdata/influxdb/client/v2"

	"github.com/rs/zerolog"

	"golang.org/x/crypto/ssh/terminal"
)

const (
	appName          = "collectd-json-influxdb-proxy"
	influxDBdefault  = "collectd"
	influxURLdefault = "http://localhost:8086/"
	serverAddr       = ":5826"
)

// ValueLists export
type ValueLists []struct {
	Values         []float64 `json:"values"`
	DsTypes        []string  `json:"dstypes"`
	DsNames        []string  `json:"dsnames"`
	Time           float64   `json:"time"`
	Interval       float64   `json:"interval"`
	Host           string    `json:"host"`
	Plugin         string    `json:"plugin"`
	PluginInstance string    `json:"plugin_instance"`
	Type           string    `json:"type"`
	TypeInstance   string    `json:"type_instance"`
}

// Response export
type Response struct {
}

func createLogger() zerolog.Logger {
	host, _ := os.Hostname()

	if terminal.IsTerminal(int(os.Stdout.Fd())) {
		return zerolog.New(os.Stdout).With().
			Timestamp().
			Logger().
			Output(zerolog.ConsoleWriter{Out: os.Stderr})
	}

	return zerolog.New(os.Stdout).With().
		Timestamp().
		Str("app", appName).
		Str("host", host).
		Logger()
}

func logRequest(log zerolog.Logger) gin.HandlerFunc {
	return func(c *gin.Context) {
		t := time.Now()

		// process request
		c.Next()

		latency := time.Since(t)
		clientIP := c.ClientIP()
		method := c.Request.Method
		statusCode := c.Writer.Status()
		r := c.Request

		errors := c.Errors

		event := log.Info().
			Str("method", method).
			Str("url", r.URL.String()).
			Int("status", statusCode).
			Int64("size", r.ContentLength).
			Str("ip", clientIP).
			Dur("latency", latency)

		if errors != nil {
			event.Str("error", errors.String())
		}

		event.Msg("Request")
	}
}

func proxyData(ctx *gin.Context, log zerolog.Logger, c client.Client, influxDB string) (b client.BatchPoints, e error) {
	var valueLists ValueLists
	err := ctx.ShouldBindJSON(&valueLists)
	if err != nil {
		ctx.JSON(http.StatusBadRequest, gin.H{"Error": err.Error()})
		log.Error().
			Err(err).
			Msg(err.Error())
		return nil, err
	}

	bp, _ := client.NewBatchPoints(client.BatchPointsConfig{
		Database: influxDB,
	})

	for _, v := range valueLists {
		tags := map[string]string{}

		if v.Host != "" {
			tags["host"] = v.Host
		}
		if v.PluginInstance != "" {
			tags["plugin_instance"] = v.PluginInstance
		}
		if v.Type != "" {
			tags["type"] = v.Type
		}
		if v.TypeInstance != "" {
			tags["type_instance"] = v.TypeInstance
		}

		fields := map[string]interface{}{}

		for n, vv := range v.Values {
			name := v.DsNames[n]
			fields[name] = vv
		}

		s, ms := math.Modf(v.Time)
		t := time.Unix(int64(s), int64(ms*1e9))

		pt, err := client.NewPoint(
			v.Plugin,
			tags,
			fields,
			t,
		)
		if err != nil {
			log.Error().
				Err(err).
				Msg(err.Error())
			continue
		}

		bp.AddPoint(pt)
	}

	err = c.Write(bp)
	if err != nil {
		ctx.JSON(http.StatusInternalServerError, gin.H{"Error": err.Error()})
		log.Error().
			Err(err).
			Msg(err.Error())
		return bp, err
	}

	return bp, nil
}

func main() {
	addr := flag.String("address", serverAddr, "server address")
	debugVars := flag.Bool("debug-vars", false, "start server for exposer variables")
	logRequests := flag.Bool("log-requests", false, "logging every request to the server")
	influxURL := flag.String("influx-url", influxURLdefault, "Influx URL")
	influxDatabase := flag.String("influx-database", influxDBdefault, "Influx database")

	flag.Parse()

	log := createLogger()

	c, err := client.NewHTTPClient(client.HTTPConfig{
		Addr: *influxURL,
	})
	if err != nil {
		log.Fatal().Err(err).Msg("Error creating InfluxDB Client")
	} else {
		log.Info().Str("url", *influxURL).Msg("Connecting to InfluxDB")
	}
	defer c.Close()

	gin.SetMode(gin.ReleaseMode)
	router := gin.New()
	router.Use(gin.Recovery())
	if *logRequests {
		router.Use(logRequest(log))
	}

	requestsCounter := 1
	requestsCounterVar := stdExpvar.NewInt("requests.counter")
	router.POST("/", func(ctx *gin.Context) {
		requestsCounter++
		if *debugVars {
			requestsCounterVar.Set(int64(requestsCounter))
		}
		proxyData(ctx, log, c, *influxDatabase)
		response := Response{}
		ctx.JSON(http.StatusOK, response)
		return
	})

	if *debugVars {
		numGo := stdExpvar.NewInt("runtime.goroutines")
		router.GET("/debug/vars", expvar.Handler())
		go func() {
			tick := time.NewTicker(1 * time.Second)
			for {
				select {
				case <-tick.C:
					numGo.Set(int64(runtime.NumGoroutine()))
				}
			}
		}()
	}

	srv := &http.Server{
		Addr:         *addr,
		Handler:      router,
		ReadTimeout:  10 * time.Second,
		WriteTimeout: 10 * time.Second,
	}

	// On signal, gracefully shut down the server and wait 5
	// seconds for current connections to stop.
	done := make(chan struct{})
	quit := make(chan os.Signal, 1)
	signal.Notify(quit, os.Interrupt, os.Kill, syscall.SIGTERM)
	go func() {
		<-quit
		log.Info().Msg("Server is shutting down")
		ctx, cancel := stdContext.WithTimeout(stdContext.Background(), 5*time.Second)
		defer cancel()
		if err := srv.Shutdown(ctx); err != nil {
			log.Fatal().Err(err).Msg("Cannot gracefully shut down the server")
		}
		close(done)
	}()

	daemon.SdNotify(false, "READY=1")

	log.Info().Str("addr", *addr).Msg("Server is listening")

	if err := srv.ListenAndServe(); err != nil && err != http.ErrServerClosed {
		log.Fatal().Err(err).Msg("Start server listener")
	}

	// Wait for existing connections before exiting.
	<-done
}
