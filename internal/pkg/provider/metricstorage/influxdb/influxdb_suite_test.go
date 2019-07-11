/*
 * Copyright (C) 2019 Nalej - All Rights Reserved
 */

package influxdb

import (
	"testing"

	"github.com/onsi/ginkgo"
	"github.com/onsi/gomega"
)

func TestHandlerPackage(t *testing.T) {
	gomega.RegisterFailHandler(ginkgo.Fail)
	ginkgo.RunSpecs(t, "internal/pkg/provider/metricstorage/influxdb package suite")
}

var (
)

var _ = ginkgo.BeforeSuite(func() {
})

var _ = ginkgo.BeforeEach(func() {
})

var _ = ginkgo.AfterEach(func() {
})
