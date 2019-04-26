/*
 * Copyright (C)  2019 Nalej - All Rights Reserved
 */

package agent

import (
	"github.com/onsi/ginkgo"
	"github.com/onsi/gomega"
	"testing"
)

func TestAgentPackage(t *testing.T) {
	gomega.RegisterFailHandler(ginkgo.Fail)
	ginkgo.RunSpecs(t, "Agent package suite")
}