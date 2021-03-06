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

package metrics

import (
	"time"

	"github.com/nalej/edge-controller/internal/pkg/entities"
	"github.com/nalej/edge-controller/internal/pkg/provider/metricstorage/test"

	"github.com/onsi/ginkgo"
	"github.com/onsi/gomega"
)



var _ = ginkgo.Describe("metrics", func() {
	ginkgo.It("should create and configure provider", func() {
		testConfig.Set("retention", "1d")
		testConfig.Set("test.database", "foo")
		testConfig.Set("test.address", "bar")

		p, derr := NewMetrics(testConfig)
		gomega.Expect(derr).To(gomega.Succeed())

		mp := p.(*Metrics)
		provider := mp.provider.(*test.TestProvider)
		gomega.Expect(provider.Database).To(gomega.Equal("foo"))
		gomega.Expect(provider.Address).To(gomega.Equal("bar"))
		gomega.Expect(provider.IsConnected).To(gomega.Equal(false))
		gomega.Expect(provider.SchemaCreated).To(gomega.Equal(false))
		gomega.Expect(provider.Retention).To(gomega.Equal(time.Duration(0)))

		gomega.Expect(p.StartPlugin()).To(gomega.Succeed())
		gomega.Expect(provider.IsConnected).To(gomega.Equal(true))
		gomega.Expect(provider.SchemaCreated).To(gomega.Equal(true))
		gomega.Expect(provider.Retention).To(gomega.Equal(time.Hour * 24))

	})
	ginkgo.It("should disconnect provider", func() {
		mp := testMetricsPlugin.(*Metrics)
		provider := mp.provider.(*test.TestProvider)
		gomega.Expect(provider.IsConnected).To(gomega.Equal(true))

		testMetricsPlugin.StopPlugin()
		gomega.Expect(provider.IsConnected).To(gomega.Equal(false))
	})
	ginkgo.It("should not store data when provider is not connected", func() {
		testMetricsPlugin.StopPlugin()
		gomega.Expect(testMetricsPlugin.HandleAgentData("test", testData)).To(gomega.HaveOccurred())
	})
	ginkgo.It("should store data", func() {
		gomega.Expect(testMetricsPlugin.HandleAgentData("test", testData)).To(gomega.Succeed())

		mp := testMetricsPlugin.(*Metrics)
		provider := mp.provider.(*test.TestProvider)

		expected := []entities.MetricValue{
			entities.MetricValue{
				Timestamp: time.Unix(1546300800, 0).UTC(),
				Value: 12345,
			},
			entities.MetricValue{
				Timestamp: time.Unix(1546300800, 0).UTC(),
				Value: 1,
			},
		}
		gomega.Expect(provider.QueryMetric("metric1", nil, nil, "")).To(gomega.ConsistOf(expected))
	})
})
