package main

import "bytes"
import "fmt"
import "os"
import "reflect"
import "testing"
import "net/http/httptest"
import "net/http"
import "github.com/gin-gonic/gin"
import "github.com/rs/zerolog"
import "github.com/stretchr/testify/assert"
import client "github.com/influxdata/influxdb/client/v2"

func TestProxyData(t *testing.T) {
	gin.SetMode(gin.TestMode)

	//url := "http://localhost:5826/"
	url := "/dev/null"
	fmt.Println("URL: ", url)

	var jsonStr = []byte(`[{"values":[1901474177],"dstypes":["counter"],"dsnames":["value"],"time":1280959128,"interval":10,"host":"leeloo.octo.it","plugin":"cpu","plugin_instance":"0","type":"cpu","type_instance":"idle"}]`)

	rec := httptest.NewRecorder()
	c, _ := gin.CreateTestContext(rec)
	c.Request, _ = http.NewRequest("POST", url, bytes.NewBuffer(jsonStr))
	c.Request.Header.Set("Content-Type", "application/json")
	fmt.Println("recorder: ", rec.Code)
	res := rec.Result()
	fmt.Println("results: ", res.StatusCode)
	assert.Equal(t, 200, res.StatusCode)

	influxURL := "http://localhost:8086/"
	influxDB := "collectd"

	ci, _ := client.NewHTTPClient(client.HTTPConfig{
		Addr: influxURL,
	})

	log := zerolog.New(os.Stdout).With().
		Timestamp().
		Str("app", "collectd-json-influxdb-proxx_test").
		Logger()

	bp, _ := proxyData(c, log, ci, influxDB)
	assert.NotNil(t, bp)
	if bp == nil {
		return
	}

	fmt.Printf("BP: %+v\n", bp)
	fmt.Printf("BP.points: %+v\n", bp.Points())
	/* expect tags:  */
	tags := map[string]string{"host": "leeloo.octo.it", "plugin_instance": "0", "type": "cpu", "type_instance": "idle"}
	pt := bp.Points()[0]
	assert.Equal(t, true, reflect.DeepEqual(tags, pt.Tags()))
	/*
		if !reflect.DeepEqual(tags, pt.Tags()) {
			t.Errorf("Error, got %v, expected %v",
				pt.Tags(), tags)
		}
	*/
	/* expect fields */
	fields := map[string]interface{}{"value": 1.901474177e+09}
	pfields, _ := pt.Fields()
	assert.Equal(t, true, reflect.DeepEqual(fields, pfields))
	/*
			if !reflect.DeepEqual(fields, pfields) {
		   		t.Errorf("Error, got %v, expected %v",
		   			pfields, fields)
		   	}
	*/
	assert.Equal(t, "cpu", pt.Name())
	assert.Equal(t, "collectd", bp.Database())
}

func TestProxyDataNegativData1(t *testing.T) {
	gin.SetMode(gin.TestMode)

	//url := "http://localhost:5826/"
	url := "/dev/null"
	fmt.Println("URL: ", url)

	var jsonStr = []byte(`[{values":[1901474177],"dstypes":["counter"],"dsnames":["value"],"time":1280959128,"interval":10,"host":"leeloo.octo.it","plugin":"cpu","plugin_instance":"0","type":"cpu","type_instance":"idle"}]`)

	rec := httptest.NewRecorder()
	c, _ := gin.CreateTestContext(rec)
	c.Request, _ = http.NewRequest("POST", url, bytes.NewBuffer(jsonStr))
	c.Request.Header.Set("Content-Type", "application/json")
	res := rec.Result()
	assert.Equal(t, 200, res.StatusCode)

	influxURL := "http://localhost:8086/"
	influxDB := "collectd"

	ci, _ := client.NewHTTPClient(client.HTTPConfig{
		Addr: influxURL,
	})

	log := zerolog.New(os.Stdout).With().
		Timestamp().
		Str("app", "collectd-json-influxdb-proxx_test").
		Logger()

	bp, _ := proxyData(c, log, ci, influxDB)
	assert.Nil(t, bp)
}

func TestProxyDataNegativData2(t *testing.T) {
	gin.SetMode(gin.TestMode)

	//url := "http://localhost:5826/"
	url := "/dev/null"
	fmt.Println("URL: ", url)

	var jsonStr = []byte(`[{"values":["a1901474177"],"dstypes":["counter"],"dsnames":["value"],"time":1280959128,"interval":10,"host":"leeloo.octo.it","plugin":"cpu","plugin_instance":"0","type":"cpu","type_instance":"idle"}]`)

	rec := httptest.NewRecorder()
	c, _ := gin.CreateTestContext(rec)
	c.Request, _ = http.NewRequest("POST", url, bytes.NewBuffer(jsonStr))
	c.Request.Header.Set("Content-Type", "application/json")
	res := rec.Result()
	assert.Equal(t, 200, res.StatusCode)

	influxURL := "http://localhost:8086/"
	influxDB := "collectd"

	ci, _ := client.NewHTTPClient(client.HTTPConfig{
		Addr: influxURL,
	})

	log := zerolog.New(os.Stdout).With().
		Timestamp().
		Str("app", "collectd-json-influxdb-proxx_test").
		Logger()

	bp, _ := proxyData(c, log, ci, influxDB)
	assert.Nil(t, bp)
}

func TestProxyDataNegativData3(t *testing.T) {
	gin.SetMode(gin.TestMode)

	//url := "http://localhost:5826/"
	url := "/dev/null"
	fmt.Println("URL: ", url)

	var jsonStr = []byte(`[{"dstypes":[100]}]`)

	rec := httptest.NewRecorder()
	c, _ := gin.CreateTestContext(rec)
	c.Request, _ = http.NewRequest("POST", url, bytes.NewBuffer(jsonStr))
	c.Request.Header.Set("Content-Type", "application/json")
	res := rec.Result()
	assert.Equal(t, 200, res.StatusCode)

	influxURL := "http://localhost:8086/"
	influxDB := "collectd"

	ci, _ := client.NewHTTPClient(client.HTTPConfig{
		Addr: influxURL,
	})

	log := zerolog.New(os.Stdout).With().
		Timestamp().
		Str("app", "collectd-json-influxdb-proxx_test").
		Logger()

	bp, _ := proxyData(c, log, ci, influxDB)
	assert.Nil(t, bp)
}
