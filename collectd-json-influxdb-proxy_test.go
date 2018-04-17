package main

import (
	"bytes"
	"github.com/gin-gonic/gin"
	client "github.com/influxdata/influxdb/client/v2"
	"github.com/rs/zerolog"
	. "github.com/smartystreets/goconvey/convey"
	"net/http"
	"net/http/httptest"
	"os"
	"testing"
)

func TestJSONConvertion(t *testing.T) {
	Convey("Given a http request with GIN context with correct JSON data", t, func() {

		gin.SetMode(gin.TestMode)
		url := "http://localhost:5826/"

		var jsonStr = []byte(`[{"values":[1901474177],"dstypes":["counter"],"dsnames":["value"],"time":1280959128,"interval":10,"host":"leeloo.octo.it","plugin":"cpu","plugin_instance":"0","type":"cpu","type_instance":"idle"}]`)

		rec := httptest.NewRecorder()
		c, _ := gin.CreateTestContext(rec)
		c.Request, _ = http.NewRequest("POST", url, bytes.NewBuffer(jsonStr))
		c.Request.Header.Set("Content-Type", "application/json")
		res := rec.Result()
		So(res.StatusCode, ShouldEqual, 200)

		influxURL := "http://localhost:8086/"
		influxDB := "collectd"

		ci, _ := client.NewHTTPClient(client.HTTPConfig{
			Addr: influxURL,
		})

		zerolog.SetGlobalLevel(zerolog.Disabled)
		log := zerolog.New(os.Stdout).With().
			Timestamp().
			Str("app", "collectd-json-influxdb-proxx_test").
			Logger()

		bp, _ := proxyData(c, log, ci, influxDB)

		Convey("When JSON data is correct", func() {
			So(bp, ShouldNotBeNil)
			pt := bp.Points()[0]
			Convey("The tags are correct", func() {
				tags := map[string]string{"host": "leeloo.octo.it", "plugin_instance": "0", "type": "cpu", "type_instance": "idle"}
				So(tags, ShouldResemble, pt.Tags())
			})
			Convey("The rest fields are correct", func() {
				fields := map[string]interface{}{"value": 1.901474177e+09}
				pfields, _ := pt.Fields()
				So(fields, ShouldResemble, pfields)

				So("cpu", ShouldEqual, pt.Name())
				So("collectd", ShouldEqual, bp.Database())
			})
		})
	})

	Convey("Given a http request with GIN context but bad JSON data structure", t, func() {

		gin.SetMode(gin.TestMode)
		url := "http://localhost:5826/"

		var jsonStr = []byte(`[{values":[1901474177],"dstypes":["counter"],"dsnames":["value"],"time":1280959128,"interval":10,"host":"leeloo.octo.it","plugin":"cpu","plugin_instance":"0","type":"cpu","type_instance":"idle"}]`)

		rec := httptest.NewRecorder()
		c, _ := gin.CreateTestContext(rec)
		c.Request, _ = http.NewRequest("POST", url, bytes.NewBuffer(jsonStr))
		c.Request.Header.Set("Content-Type", "application/json")
		res := rec.Result()
		So(res.StatusCode, ShouldEqual, 200)

		influxURL := "http://localhost:8086/"
		influxDB := "collectd"

		ci, _ := client.NewHTTPClient(client.HTTPConfig{
			Addr: influxURL,
		})

		zerolog.SetGlobalLevel(zerolog.Disabled)
		log := zerolog.New(os.Stdout).With().
			Timestamp().
			Str("app", "collectd-json-influxdb-proxx_test").
			Logger()

		bp, _ := proxyData(c, log, ci, influxDB)
		So(bp, ShouldBeNil)
	})

	Convey("Given a http request with GIN context but bad JSON data, cannot unmarshal to go structure", t, func() {

		gin.SetMode(gin.TestMode)
		url := "http://localhost:5826/"

		var jsonStr = []byte(`[{"values":["a1901474177"],"dstypes":[1],"dsnames":"value","time":"a1280959128","interval":"10a","host":0,"plugin":[2],"plugin_instance":0,"type":["cpu"],"type_instance":1}]`)

		rec := httptest.NewRecorder()
		c, _ := gin.CreateTestContext(rec)
		c.Request, _ = http.NewRequest("POST", url, bytes.NewBuffer(jsonStr))
		c.Request.Header.Set("Content-Type", "application/json")
		res := rec.Result()
		So(res.StatusCode, ShouldEqual, 200)

		influxURL := "http://localhost:8086/"
		influxDB := "collectd"

		ci, _ := client.NewHTTPClient(client.HTTPConfig{
			Addr: influxURL,
		})

		//zerolog.SetGlobalLevel(zerolog.InfoLevel)
		log := zerolog.New(os.Stdout).With().
			Timestamp().
			Str("app", "collectd-json-influxdb-proxx_test").
			Logger()

		bp, _ := proxyData(c, log, ci, influxDB)
		So(bp, ShouldBeNil)
	})
}
