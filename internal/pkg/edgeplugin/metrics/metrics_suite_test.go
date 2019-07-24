/*
 * Copyright (C) 2019 Nalej - All Rights Reserved
 */

package metrics

import (
	"testing"

	"github.com/nalej/grpc-edge-controller-go"


	"github.com/nalej/edge-controller/internal/pkg/edgeplugin"
	_ "github.com/nalej/edge-controller/internal/pkg/provider/metricstorage/test"

	"github.com/onsi/ginkgo"
	"github.com/onsi/gomega"

	"github.com/spf13/viper"
)

func TestHandlerPackage(t *testing.T) {
	gomega.RegisterFailHandler(ginkgo.Fail)
	ginkgo.RunSpecs(t, "internal/pkg/edgeplugin/metrics package suite")
}

var (
	testConfig *viper.Viper
	testMetricsPlugin edgeplugin.EdgePlugin

	testData *grpc_edge_controller_go.PluginData
)

var _ = ginkgo.BeforeSuite(func() {
	testTags := map[string]string{
		"testtag1": "testval1",
		"testtag2": "testval2",
	}

	testFields := map[string]uint64{
		"field1": 12345,
		"field2": 1,
	}

	testData = &grpc_edge_controller_go.PluginData{
		Plugin: 0,
		Data: &grpc_edge_controller_go.PluginData_MetricsData{
			MetricsData: &grpc_edge_controller_go.MetricsPluginData{
				Timestamp: 1546300800, // 1/1/2019 12:00AM GMT
				Metrics: []*grpc_edge_controller_go.MetricsPluginData_Metric{
					&grpc_edge_controller_go.MetricsPluginData_Metric{
						Name: "metric1",
						Tags: testTags,
						Fields: testFields,
					},
					&grpc_edge_controller_go.MetricsPluginData_Metric{
						Name: "metric2",
						Tags: testTags,
						Fields: testFields,
					},
				},
			},
		},
	}
})

var _ = ginkgo.BeforeEach(func() {
	testConfig = viper.New()
	testConfig.Set("provider", "test")

	p, derr := NewMetrics(testConfig)
	gomega.Expect(derr).To(gomega.Succeed())

	testMetricsPlugin = p.(edgeplugin.EdgePlugin)
	gomega.Expect(testMetricsPlugin.StartPlugin()).To(gomega.Succeed())
})

var _ = ginkgo.AfterEach(func() {
	testMetricsPlugin.StopPlugin()
})
