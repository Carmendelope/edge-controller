package asset

import (
	"github.com/onsi/ginkgo"
)

const databasePath = "/tmp/nalej.db"
var _ = ginkgo.Describe("Asset bbolt provider", func(){

	b := NewBboltAssetProvider(databasePath)
	RunTest(b)

})
