package asset

import (
	"github.com/onsi/ginkgo"
	"github.com/rs/zerolog/log"
	"io/ioutil"
)

var _ = ginkgo.Describe("Asset bbolt provider", func(){

	file, err := ioutil.TempFile("", "*.db")
	if err != nil {
		log.Panic().Msg("enable to create file")
	}
	b := NewBboltAssetProvider(file.Name())
	RunTest(b)

	ginkgo.AfterSuite(func() {
		b.Close()
	})

})
