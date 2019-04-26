/*
 * Copyright (C)  2019 Nalej - All Rights Reserved
 */

package asset

import "github.com/onsi/ginkgo"

var _ = ginkgo.Describe("Asset provider", func(){

	sp := NewMockupAssetProvider()
	RunTest(sp)

})