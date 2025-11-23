package v1

import (
	"fmt"
	"testing"
	"time"

	"github.com/hyperledger-labs/fabric-smart-client/platform/common/utils"
)

func TestA(t *testing.T) {
	fmt.Println(utils.ExponentialBucketTimeRange(0, 1*time.Second, 10))
}
