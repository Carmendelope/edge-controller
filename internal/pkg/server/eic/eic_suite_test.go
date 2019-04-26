/*
 * Copyright (C)  2019 Nalej - All Rights Reserved
 */

package eic

import (
	"github.com/onsi/ginkgo"
	"github.com/onsi/gomega"
	"testing"
)

func TestEICPackage(t *testing.T) {
	gomega.RegisterFailHandler(ginkgo.Fail)
	ginkgo.RunSpecs(t, "EIC package suite")
}