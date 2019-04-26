/*
 * Copyright (C)  2019 Nalej - All Rights Reserved
 */

package asset

import (
	"github.com/onsi/ginkgo"
	"github.com/onsi/gomega"
	"testing"
)

func TestAssetProviderPackage(t *testing.T){
	gomega.RegisterFailHandler(ginkgo.Fail)
	ginkgo.RunSpecs(t, "Asset Providers package suite")
}
