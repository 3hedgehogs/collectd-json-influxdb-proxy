package main

import (
	"log"
	"math"
	"os"
	"time"

	"github.com/go-siris/middleware-logger"
	"github.com/go-siris/siris"
	"github.com/go-siris/siris/context"

	client "github.com/influxdata/influxdb/client/v2"
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

	app.Run(siris.Addr(addr), siris.WithCharset("UTF-8"))
}
