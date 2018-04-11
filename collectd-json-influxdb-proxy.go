package main

import (
	stdContext "context"
	"expvar"
	"flag"
	"log"
	"math"
	"net/http"
	"os"
	"os/signal"
	"runtime"
	"syscall"
	"time"

	"github.com/coreos/go-systemd/daemon"

	"github.com/gin-gonic/gin"
	"github.com/golang/glog"
	"github.com/szuecs/gin-glog"

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

func main() {

	addr := flag.String("address", serverAddr,
		"server address")
	debugserver := flag.String("expvar_server", "",
		"server for exposer variables (empty to disable)")
	influxURL := flag.String("influxurl", influxURLdefault,
		"Influx URL")
	influxDB := flag.String("influxdb", influxDBdefault,
		"Influx DB")

	flag.Parse()

	c, err := client.NewHTTPClient(client.HTTPConfig{
		Addr: *influxURL,
	})
	if err != nil {
		log.Fatalln("Error creating InfluxDB Client: ", err.Error())
	}
	defer c.Close()

	gin.SetMode(gin.ReleaseMode)
	router := gin.Default()
	router.Use(ginglog.Logger(3 * time.Second))
	router.Use(gin.Recovery())

	router.POST("/", func(ctx *gin.Context) {
		var valueLists ValueLists

		err := ctx.ShouldBindJSON(&valueLists)
		if err != nil {
			ctx.JSON(http.StatusBadRequest, gin.H{"Error": err.Error()})
			//ctx.Abort()
			return
		}

		bp, _ := client.NewBatchPoints(client.BatchPointsConfig{
			Database: *influxDB,
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
				glog.Error(err.Error())
				continue
			}

			bp.AddPoint(pt)
		}

		err = c.Write(bp)
		if err != nil {
			ctx.JSON(http.StatusInternalServerError, gin.H{"Error": err.Error()})
			//ctx.Abort()
			return
		}

		response := Response{}
		ctx.JSON(http.StatusOK, response)
		return
	})

	if *debugserver != "" {
		numGo := expvar.NewInt("runtime.goroutines")
		go func() {
			glog.Fatalf("Start debug server: %s", http.ListenAndServe(*debugserver, nil))
		}()
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
		log.Print("Server is shutting down.")
		ctx, cancel := stdContext.WithTimeout(stdContext.Background(), 5*time.Second)
		defer cancel()
		if err := srv.Shutdown(ctx); err != nil {
			log.Fatalf("cannot gracefully shut down the server: %s", err)
		}
		close(done)
	}()

	daemon.SdNotify(false, "READY=1")
	if err := srv.ListenAndServe(); err != nil && err != http.ErrServerClosed {
		log.Fatalf("listen: %s\n", err)
	}

	// Wait for existing connections before exiting.
	<-done
}
