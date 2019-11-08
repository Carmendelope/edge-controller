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
	"testing"
	"time"

	"github.com/nalej/edge-controller/internal/pkg/entities"

	"github.com/onsi/ginkgo"
	"github.com/onsi/gomega"
)

func TestHandlerPackage(t *testing.T) {
	gomega.RegisterFailHandler(ginkgo.Fail)
	ginkgo.RunSpecs(t, "internal/pkg/provider/metricstorage/influxdb package suite")
}

var (
	testMetricsData *entities.MetricsData
	testMetricsLine string
	testTags map[string][]string
	testExtraTags map[string]string
	testMetricsLineExtra string
)

var _ = ginkgo.BeforeSuite(func() {
	testMetricsData = &entities.MetricsData{
		Timestamp: time.Unix(1, 1),
		Metrics: []*entities.Metric{
			&entities.Metric{
				Name: "metric1",
				Tags: map[string]string{
					"a": "b",
					"c": "d",
				},
				Fields: map[string]uint64{
					"x": 12345,
					"y": 67890,
				},
			},
			&entities.Metric{
				Name: "metric2",
				Tags: map[string]string{
					"e": "f",
					"g": "h",
				},
				Fields: map[string]uint64{
					"z": 12345,
					"w": 67890,
				},
			},
		},
	}

	testMetricsLine = "metric1,a=b,c=d x=12345i,y=67890i 1\nmetric2,e=f,g=h w=67890i,z=12345i 1\n"

	testTags = map[string][]string{
		"asset_id": []string{"asset1", "asset2"},
	}

	testExtraTags = map[string]string{
		"tag1": "val1",
		"tag2": "val2",
	}

	testMetricsLineExtra = "metric1,a=b,c=d,tag1=val1,tag2=val2 x=12345i,y=67890i 1\nmetric2,e=f,g=h,tag1=val1,tag2=val2 w=67890i,z=12345i 1\n"
})
