package massive

import "time"

// tryEnqueue preserves only the historical queue unit-test call shape while
// those tests are migrated to decoded-batch fixtures. Production has no raw
// admission method and always decodes before tryEnqueueDecoded.
func (q *liveFrameQueue) tryEnqueue(epoch uint64, kind socketMessageType, receivedAt time.Time, data []byte) (queuedLiveFrame, FrameAdmissionReason, LiveQueueAccounting) {
	sequence, reason, accounting := q.beginDecode(epoch, kind, receivedAt, len(data))
	if reason != FrameAdmitted {
		return queuedLiveFrame{}, reason, accounting
	}
	batch := DecodedBatch{BindingIdentity: q.binding.Identity(), ConnectionEpoch: epoch, FrameSequence: sequence, ReceivedAt: receivedAt, EncodedBytes: len(data), RetainedCharge: len(data), parsePasses: 1}
	if q.binding.Identity() != "" {
		batch = decodeLiveFrame(LiveFrame{Binding: q.binding, ConnectionEpoch: epoch, FrameSequence: sequence, ReceivedAt: receivedAt, Data: data}, nil, LiveNormalizationOptions{})
	}
	return q.tryEnqueueDecoded(epoch, sequence, receivedAt, len(data), batch)
}
