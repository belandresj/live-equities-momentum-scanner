package operations

import (
	"testing"
	"time"
)

func TestPLBRC2DefaultCombinedCadenceIsOneSecond(t *testing.T) {
	if cadence := DefaultConfig().SampleCadence; cadence != time.Second {
		t.Fatalf("combined publication/membership cadence = %s, want 1s", cadence)
	}
}
