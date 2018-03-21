package main

import (
	stdContext "context"
	"log"
	"math"
	"os"
	"os/signal"
	"syscall"
	"time"

	"github.com/go-siris/middleware-logger"
	"github.com/go-siris/siris"
	"github.com/go-siris/siris/context"

	client "github.com/influxdata/influxdb/client/v2"

	"github.com/coreos/go-systemd/daemon"
)

const (
	influxDB   = "collectd"
	influxURL  = "http://localhost:8086/"
	serverAddr = ":5826"
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
	var addr string
	if len(os.Args) > 1 {
		addr = os.Args[1]
	} else {
		addr = serverAddr
	}

	c, err := client.NewHTTPClient(client.HTTPConfig{
		Addr: influxURL,
	})
	if err != nil {
		log.Fatalln("Error creating InfluxDB Client: ", err.Error())
	}
	defer c.Close()

	app := siris.New()
	siris.WithoutBanner(app)

	requestLogger := logger.New()
	app.Use(requestLogger)

	app.Post("/", func(ctx context.Context) {
		var valueLists ValueLists

		err := ctx.ReadJSON(&valueLists)

		if err != nil {
			ctx.Values().Set("error", err.Error())
			ctx.StatusCode(siris.StatusInternalServerError)
			return
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
				app.Logger().Error(err.Error())
				continue
			}

			bp.AddPoint(pt)
		}

		err = c.Write(bp)
		if err != nil {
			ctx.Values().Set("error", err.Error())
			ctx.StatusCode(siris.StatusInternalServerError)
			return
		}

		response := Response{}
		ctx.JSON(response)
	})

	// On signal, gracefully shut down the server and wait 5
	// seconds for current connections to stop.
	done := make(chan struct{})
	quit := make(chan os.Signal, 1)
	signal.Notify(quit, os.Interrupt, os.Kill, syscall.SIGTERM)
	go func() {
		<-quit
		log.Print("server is shutting down")
		ctx, cancel := stdContext.WithTimeout(stdContext.Background(), 5*time.Second)
		defer cancel()
		if err := app.Shutdown(ctx); err != nil {
			log.Fatalf("cannot gracefully shut down the server: %s", err)
		}
		close(done)
	}()

	daemon.SdNotify(false, "READY=1")
	app.Run(siris.Addr(addr), siris.WithCharset("UTF-8"))

	// Wait for existing connections before exiting.
	<-done
}
