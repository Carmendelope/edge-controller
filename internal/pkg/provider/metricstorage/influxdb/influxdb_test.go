/*
 * Copyright 2019 Nalej
 *
 * Licensed under the Apache License, Version 2.0 (the "License");
 * you may not use this file except in compliance with the License.
 * You may obtain a copy of the License at
 *
 *     http://www.apache.org/licenses/LICENSE-2.0
 *
 * Unless required by applicable law or agreed to in writing, software
 * distributed under the License is distributed on an "AS IS" BASIS,
 * WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
 * See the License for the specific language governing permissions and
 * limitations under the License.
 */

package influxdb

import (
	"net/http"
	"time"

	"github.com/nalej/edge-controller/internal/pkg/entities"
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
			expectQueries(server,
				testQuery{Type: regularQuery, Query: "SHOW DATABASES"},
				testQuery{Type: regularQuery, Query: "CREATE DATABASE testdb WITH DURATION inf REPLICATION 1 NAME testdb"},
			)
			gomega.Expect(provider.Connect()).To(gomega.Succeed())
			gomega.Expect(provider.CreateSchema(false)).To(gomega.Succeed())
		})
		ginkgo.It("should create a schema when database does not exist and ifNeeded is set", func() {
			expectQueries(server,
				testQuery{Type: regularQuery, Query: "SHOW DATABASES"},
				testQuery{Type: regularQuery, Query: "CREATE DATABASE testdb WITH DURATION inf REPLICATION 1 NAME testdb"},
			)
			gomega.Expect(provider.Connect()).To(gomega.Succeed())
			gomega.Expect(provider.CreateSchema(true)).To(gomega.Succeed())
		})
		ginkgo.It("should fail when database exists", func() {
			expectQueries(server,
				testQuery{
					Type: regularQuery,
					Query: "SHOW DATABASES",
					Response: []interface{}{"testdb1", "testdb"},
				},
				testQuery{Type: regularQuery, Query: "CREATE DATABASE testdb WITH DURATION inf REPLICATION 1 NAME testdb"},
			)
			gomega.Expect(provider.Connect()).To(gomega.Succeed())
			gomega.Expect(provider.CreateSchema(false)).To(gomega.HaveOccurred())
		})
		ginkgo.It("should not fail when database exists and ifNeeded is set", func() {
			expectQueries(server,
				testQuery{
					Type: regularQuery,
					Query: "SHOW DATABASES",
					Response: []interface{}{"testdb1", "testdb"},
				},
				testQuery{Type: regularQuery, Query: "CREATE DATABASE testdb WITH DURATION inf REPLICATION 1 NAME testdb"},
			)
			gomega.Expect(provider.Connect()).To(gomega.Succeed())
			gomega.Expect(provider.CreateSchema(true)).To(gomega.Succeed())
		})
	})

	ginkgo.Context("StoreMetricsData", func() {
		ginkgo.It("should not fail on empty metrics", func() {
			expectQueries(server, testQuery{Type: batchQuery})
			gomega.Expect(provider.Connect()).To(gomega.Succeed())
			gomega.Expect(provider.StoreMetricsData(&entities.MetricsData{}, nil)).To(gomega.Succeed())

		})
		ginkgo.It("should store multiple metrics", func() {
			expectQueries(server, testQuery{
				Type: batchQuery,
				Query: testMetricsLine,
			})
			gomega.Expect(provider.Connect()).To(gomega.Succeed())
			gomega.Expect(provider.StoreMetricsData(testMetricsData, nil)).To(gomega.Succeed())
		})
		ginkgo.It("should store extra tags", func() {
			expectQueries(server, testQuery{
				Type: batchQuery,
				Query: testMetricsLineExtra,
			})

			gomega.Expect(provider.Connect()).To(gomega.Succeed())
			gomega.Expect(provider.StoreMetricsData(testMetricsData, testExtraTags)).To(gomega.Succeed())

		})
	})

	ginkgo.Context("ListMetrics", func() {
		ginkgo.It("should return empty list when no metrics available", func() {
			expectQueries(server,
				testQuery{Type: regularQuery, Query: "SHOW MEASUREMENTS WHERE (\"asset_id\"='asset1' OR \"asset_id\"='asset2')"},
			)
			gomega.Expect(provider.Connect()).To(gomega.Succeed())
			gomega.Expect(provider.ListMetrics(testTags)).To(gomega.BeEmpty())
		})
		ginkgo.It("should return metrics list", func() {
			expectQueries(server,
				testQuery{
					Type: regularQuery,
					Query: "SHOW MEASUREMENTS ",
					Response: []interface{}{"metric1", "metric2"},
				},
			)
			gomega.Expect(provider.Connect()).To(gomega.Succeed())
			gomega.Expect(provider.ListMetrics(nil)).To(gomega.ConsistOf([]string{"metric1", "metric2"}))

		})
	})

	// Note - generate queries are tested separately
	ginkgo.Context("QueryMetric", func() {
		ginkgo.It("should return empty response when no data is available", func() {
			expectQueries(server, testQuery{Type: regularQuery})
			gomega.Expect(provider.Connect()).To(gomega.Succeed())
			gomega.Expect(provider.QueryMetric("cpu", nil, &entities.TimeRange{Timestamp: time.Unix(1,1)}, entities.AggregateAvg)).To(gomega.BeEmpty())
		})
		ginkgo.It("should return valid data", func() {
			expectQueries(server, testQuery{
				Type: regularQuery,
				Response: []interface{}{[]interface{}{"2019-07-11T10:32:00Z",689,2}},
			})
			timestamp, _ := time.Parse(time.RFC3339, "2019-07-11T10:32:00Z")
			response := []entities.MetricValue{
				entities.MetricValue{
					Timestamp: timestamp,
					Value: 689,
					AssetCount: 2,
				},
			}
			gomega.Expect(provider.Connect()).To(gomega.Succeed())
			gomega.Expect(provider.QueryMetric("cpu", nil, &entities.TimeRange{Timestamp: time.Unix(1,1)}, entities.AggregateAvg)).To(gomega.Equal(response))

		})
		ginkgo.It("should handle errors", func() {
			expectQueries(server, testQuery{
				Type: regularQuery,
				Error: "this is an error",
			})
			gomega.Expect(provider.Connect()).To(gomega.Succeed())
			_, err := provider.QueryMetric("cpu", nil, &entities.TimeRange{Timestamp: time.Unix(1,1)}, entities.AggregateAvg)
			gomega.Expect(err).To(gomega.HaveOccurred())
		})
	})

	ginkgo.Context("SetRetention", func() {
		ginkgo.It("should set infinite retention", func() {
			expectQueries(server, testQuery{
				Type: regularQuery,
				Query: "ALTER RETENTION POLICY testdb ON testdb DURATION inf SHARD DURATION 1w",
			})
			gomega.Expect(provider.Connect()).To(gomega.Succeed())
			gomega.Expect(provider.SetRetention(time.Duration(0))).To(gomega.Succeed())
		})
		ginkgo.It("should set short retention with short shard", func() {
			expectQueries(server, testQuery{
				Type: regularQuery,
				Query: "ALTER RETENTION POLICY testdb ON testdb DURATION 1h0m0s SHARD DURATION 1h",
			})
			gomega.Expect(provider.Connect()).To(gomega.Succeed())
			gomega.Expect(provider.SetRetention(time.Hour)).To(gomega.Succeed())
		})
		ginkgo.It("should set medium retention with medium shard", func() {
			expectQueries(server, testQuery{
				Type: regularQuery,
				Query: "ALTER RETENTION POLICY testdb ON testdb DURATION 72h0m0s SHARD DURATION 1d",
			})
			gomega.Expect(provider.Connect()).To(gomega.Succeed())
			gomega.Expect(provider.SetRetention(time.Hour * 72)).To(gomega.Succeed())
		})
		ginkgo.It("should set long retention with long shard", func() {
			expectQueries(server, testQuery{
				Type: regularQuery,
				Query: "ALTER RETENTION POLICY testdb ON testdb DURATION 4800h0m0s SHARD DURATION 1w",
			})
			gomega.Expect(provider.Connect()).To(gomega.Succeed())
			gomega.Expect(provider.SetRetention(time.Hour * 24 * 200)).To(gomega.Succeed())
		})
	})
})

type queryType string
const (
	regularQuery queryType = "query"
	batchQuery queryType = "write"
)

type testQuery struct {
	Type queryType
	Query string
	Response []interface{}
	Error string
}

func expectQueries(server *ghttp.Server, queries ...testQuery) {
	for _, query := range(queries) {
		values := [][]interface{}{}
		for _, value := range(query.Response) {
			vList, ok := value.([]interface{})
			if !ok {
				vList = []interface{}{value}
			}
			values = append(values, vList)
		}

		response := client.Response{
			Results: []client.Result{
				client.Result{
					Series: []models.Row{
						models.Row{
							Values: values,
						},
					},
				},
			},
		}

		if query.Error != "" {
			response.Err = query.Error
		}

		var f http.HandlerFunc
		switch query.Type {
		case regularQuery:
			if query.Query != "" {
				f = ghttp.VerifyFormKV("q", query.Query)
			} else {
				f = ghttp.VerifyForm(nil)
			}
		case batchQuery:
			f = ghttp.VerifyBody([]byte(query.Query))
		}

		server.AppendHandlers(
			ghttp.CombineHandlers(
				ghttp.VerifyRequest("POST", "/" + string(query.Type)),
				f,
				ghttp.RespondWithJSONEncoded(http.StatusOK, response),
			),
		)
	}
}
