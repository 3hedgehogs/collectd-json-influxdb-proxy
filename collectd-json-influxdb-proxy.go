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
	"github.com/rs/zerolog"

	client "github.com/influxdata/influxdb/client/v2"
)

const (
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

func reqLogger(log zerolog.Logger) gin.HandlerFunc {
	return func(c *gin.Context) {
		t := time.Now()

		// process request
		c.Next()

		latency := time.Since(t)
		clientIP := c.ClientIP()
		method := c.Request.Method
		statusCode := c.Writer.Status()
		r := c.Request

		log.Info().
			Str("method", method).
			Str("url", r.URL.String()).
			Int("status", statusCode).
			Int64("size", r.ContentLength).
			Str("ip", clientIP).
			Dur("latency", latency).
			Msgf("%s", c.Errors.String())
	}
}

func proxyData(ctx *gin.Context, log zerolog.Logger, c client.Client, influxDB string) (b client.BatchPoints, e error) {

	var valueLists ValueLists
	err := ctx.ShouldBindJSON(&valueLists)
	if err != nil {
		ctx.JSON(http.StatusBadRequest, gin.H{"Error": err.Error()})
		log.Info().
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
			log.Info().
				Err(err).
				Msg(err.Error())
			continue
		}

		bp.AddPoint(pt)
	}

	err = c.Write(bp)
	if err != nil {
		ctx.JSON(http.StatusInternalServerError, gin.H{"Error": err.Error()})
		log.Info().
			Err(err).
			Msg(err.Error())
		return bp, err
	}

	return bp, nil
}

func main() {

	addr := flag.String("address", serverAddr,
		"server address")
	debugserver := flag.Bool("expvar_server", false,
		"start server for exposer variables")
	logRequests := flag.Bool("logrequests", false,
		"logging every request to the server")
	influxURL := flag.String("influxurl", influxURLdefault,
		"Influx URL")
	influxDB := flag.String("influxdb", influxDBdefault,
		"Influx DB")

	flag.Parse()

	host, _ := os.Hostname()
	log := zerolog.New(os.Stdout).With().
		Timestamp().
		Str("app", "collectd-json-influxdb-proxy").
		Str("host", host).
		Logger()

	c, err := client.NewHTTPClient(client.HTTPConfig{
		Addr: *influxURL,
	})
	if err != nil {
		log.Fatal().Err(err).Msg("Error creating InfluxDB Client")
	}
	defer c.Close()

	gin.SetMode(gin.ReleaseMode)
	router := gin.New()
	router.Use(gin.Recovery())
	if *logRequests {
		router.Use(reqLogger(log))
	}

	router.POST("/", func(ctx *gin.Context) {
		proxyData(ctx, log, c, *influxDB)
		response := Response{}
		ctx.JSON(http.StatusOK, response)
		return
	})

	if *debugserver {
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
		log.Info().
			Msg("Server is shutting down.")
		ctx, cancel := stdContext.WithTimeout(stdContext.Background(), 5*time.Second)
		defer cancel()
		if err := srv.Shutdown(ctx); err != nil {
			log.Fatal().
				Err(err).
				Msg("Cannot gracefully shut down the server")
		}
		close(done)
	}()

	daemon.SdNotify(false, "READY=1")
	if err := srv.ListenAndServe(); err != nil && err != http.ErrServerClosed {
		log.Fatal().
			Err(err).
			Msg("Start server listener")
	}

	// Wait for existing connections before exiting.
	<-done
}
