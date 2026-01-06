package driver

import (
	"testing"

	"go.uber.org/zap"
)

func TestMakeCSIPanicRecovery_Recovers(t *testing.T) {
	logger, _ := zap.NewDevelopment()
	log := logger.Sugar()

	func() {
		defer MakeCSIPanicRecovery(log, nil, "UnitTestOp", map[string]string{"test": "true"})()
		panic("boom")
		//code after panic doesn't run, but this as we are recovering from panic, test suite will not crash
		//which will validate this functionality
	}()

}
