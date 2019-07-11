/*
 * Copyright (C) 2019 Nalej - All Rights Reserved
 */

package influxdb

import (
	"net/http"

	"github.com/nalej/edge-controller/internal/pkg/provider/metricstorage"

	"github.com/influxdata/influxdb1-client/v2"
	"github.com/influxdata/influxdb1-client/models"

	"github.com/onsi/ginkgo"
	"github.com/onsi/gomega"
	"github.com/onsi/gomega/ghttp"

	"github.com/spf13/viper"
)



var _ = ginkgo.Describe("influxdb", func() {
	var server *ghttp.Server
	var provider *InfluxDBProvider

	ginkgo.BeforeEach(func() {
		server = ghttp.NewServer()

		conf := viper.New()
		conf.Set("influxdb.address", server.URL())
		conf.Set("influxdb.database", "testdb")
		connConf, derr := metricstorage.NewConnectionConfig(conf)
		gomega.Expect(derr).To(gomega.Succeed())

		p, derr := NewInfluxDBProvider(connConf)
		gomega.Expect(derr).To(gomega.Succeed())
		provider = p.(*InfluxDBProvider)
	})

	ginkgo.AfterEach(func() {
		server.Close()
		provider = nil
	})

	ginkgo.Context("connection", func() {
		ginkgo.It("should connect and disconnect", func() {
			gomega.Expect(provider.Connect()).To(gomega.Succeed())
			gomega.Expect(provider.Connected()).To(gomega.BeTrue())
			gomega.Expect(provider.Disconnect()).To(gomega.Succeed())
			gomega.Expect(provider.Connected()).To(gomega.BeFalse())
		})
	})

	ginkgo.Context("CreateSchema", func() {
		ginkgo.It("should create a schema when database does not exist", func() {
			server.AppendHandlers(
				ghttp.CombineHandlers(
					ghttp.VerifyRequest("POST", "/query"),
					ghttp.VerifyFormKV("q", "SHOW DATABASES"),
					ghttp.RespondWithJSONEncoded(http.StatusOK, client.Response{}),
				),
				ghttp.CombineHandlers(
					ghttp.VerifyRequest("POST", "/query"),
					ghttp.VerifyFormKV("q", "CREATE DATABASE testdb WITH DURATION inf REPLICATION 1 NAME testdb"),
					ghttp.RespondWithJSONEncoded(http.StatusOK, client.Response{}),
				),
			)
			gomega.Expect(provider.Connect()).To(gomega.Succeed())
			gomega.Expect(provider.CreateSchema(false)).To(gomega.Succeed())
		})
		ginkgo.It("should create a schema when database does not exist and ifNeeded is set", func() {
			server.AppendHandlers(
				ghttp.CombineHandlers(
					ghttp.VerifyRequest("POST", "/query"),
					ghttp.VerifyFormKV("q", "SHOW DATABASES"),
					ghttp.RespondWithJSONEncoded(http.StatusOK, client.Response{}),
				),
				ghttp.CombineHandlers(
					ghttp.VerifyRequest("POST", "/query"),
					ghttp.VerifyFormKV("q", "CREATE DATABASE testdb WITH DURATION inf REPLICATION 1 NAME testdb"),
					ghttp.RespondWithJSONEncoded(http.StatusOK, client.Response{}),
				),
			)
			gomega.Expect(provider.Connect()).To(gomega.Succeed())
			gomega.Expect(provider.CreateSchema(true)).To(gomega.Succeed())
		})
		ginkgo.It("should fail when database exists", func() {
			server.AppendHandlers(
				ghttp.CombineHandlers(
					ghttp.VerifyRequest("POST", "/query"),
					ghttp.VerifyFormKV("q", "SHOW DATABASES"),
					ghttp.RespondWithJSONEncoded(http.StatusOK, client.Response{
						Results: []client.Result{
							client.Result{
								Series: []models.Row{
									models.Row{
										Values: [][]interface{}{
											[]interface{}{
												"testdb1",
											},
											[]interface{}{
												"testdb",
											},
										},
									},
								},
							},
						},
					}),
				),
			)
			gomega.Expect(provider.Connect()).To(gomega.Succeed())
			gomega.Expect(provider.CreateSchema(false)).To(gomega.HaveOccurred())
		})
		ginkgo.It("should not fail when database exists and ifNeeded is set", func() {
			server.AppendHandlers(
				ghttp.CombineHandlers(
					ghttp.VerifyRequest("POST", "/query"),
					ghttp.VerifyFormKV("q", "SHOW DATABASES"),
					ghttp.RespondWithJSONEncoded(http.StatusOK, client.Response{
						Results: []client.Result{
							client.Result{
								Series: []models.Row{
									models.Row{
										Values: [][]interface{}{
											[]interface{}{
												"testdb1",
											},
											[]interface{}{
												"testdb",
											},
										},
									},
								},
							},
						},
					}),
				),
			)
			gomega.Expect(provider.Connect()).To(gomega.Succeed())
			gomega.Expect(provider.CreateSchema(true)).To(gomega.Succeed())
		})
	})

	ginkgo.Context("StoreMetricsData", func() {
		ginkgo.It("should not fail on empty metrics", func() {

		})
		ginkgo.It("should store multiple metrics", func() {

		})
		ginkgo.It("should store multiple fields", func() {

		})
		ginkgo.It("should store extra tags", func() {

		})
	})

	ginkgo.Context("ListMetrics", func() {
		ginkgo.It("should return empty list when no metrics available", func() {

		})
		ginkgo.It("should return metrics list", func() {

		})
	})

	// Note - generate queries are tested separately
	ginkgo.Context("QueryMetric", func() {
		ginkgo.It("should return empty response when no data is available", func() {

		})
		ginkgo.It("should return valid data", func() {

		})
		ginkgo.It("should handle errors", func() {

		})
	})

	ginkgo.Context("SetRetention", func() {
		ginkgo.It("should set infinite retention", func() {

		})
		ginkgo.It("should set short retention with short shard", func() {

		})
		ginkgo.It("should set medium retention with medium shard", func() {

		})
		ginkgo.It("should set long retention with long shard", func() {

		})
	})
})

/*
type testQuery struct {
	Query string
	Response client.Response
}

func testServerProvider(queries []testQuery) (*InfluxDBProvider, *httptest.Server) {
	var queryCount int = 0

	handler := http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		var data client.Response

		params := r.URL.Query()
		query := params.Get("q")
		if query != "" {
			gomega.Expect(query).To(gomega.Equal(queries[queryCount].Query))
			data = queries[queryCount].Response
			queryCount++
		}

		fmt.Printf("%+v\n", params)
		w.Header().Set("Content-Type", "application/json")
		w.Header().Set("X-Influxdb-Version", "1.3.1")
		w.WriteHeader(http.StatusOK)
		_ = json.NewEncoder(w).Encode(data)
	})

	ts := httptest.NewServer(handler)

	conf := viper.New()
	conf.Set("influxdb.address", ts.URL)
	conf.Set("influxdb.database", "testdb")
	connConf, derr := metricstorage.NewConnectionConfig(conf)
	gomega.Expect(derr).To(gomega.Succeed())

	p, derr := NewInfluxDBProvider(connConf)
	gomega.Expect(derr).To(gomega.Succeed())

	return p.(*InfluxDBProvider), ts
}
*/
