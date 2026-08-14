package massive

import (
	"testing"
	"time"
)

func TestPTQRPressureBurstLeavesHardCapacityHeadroom(t *testing.T) {
	config := LiveQueueConfig{FrameSlots: MaximumLiveFrameSlots, MaxFrameBytes: 1024, TotalFrameBytes: MaximumLiveFrameSlots * 1024}
	queue := newLiveFrameQueue(config)
	start := time.Date(2026, 8, 14, 16, 0, 0, 0, time.UTC)
	earlyShedBoundary := MaximumLiveFrameSlots / 4
	for index := 0; index < earlyShedBoundary; index++ {
		if _, reason, _ := queue.tryEnqueue(1, socketMessageText, start.Add(time.Duration(index)*time.Nanosecond), []byte(`[{"ev":"T","sym":"AAA"}]`)); reason != FrameAdmitted {
			t.Fatalf("burst frame %d admission=%s", index, reason)
		}
	}
	accounting := queue.snapshot()
	if accounting.FramesQueued != uint64(earlyShedBoundary) || accounting.FramesRejectedCapacity != 0 ||
		accounting.CapacityFrames != MaximumLiveFrameSlots || !accounting.Reconciles() ||
		int(accounting.FramesQueued) >= accounting.CapacityFrames {
		t.Fatalf("early-shed burst consumed hard capacity: %+v", accounting)
	}
}
