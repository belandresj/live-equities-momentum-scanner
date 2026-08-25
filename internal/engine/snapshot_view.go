package engine

// SnapshotView is Component 10's detached projection of exactly one immutable
// engine publication. Publication, operational, and T/Q fields are derived
// from the same atomic cell; consumers must not join later Observe calls.
type SnapshotView struct {
	Publication PublicationView
	Operational OperationalView
	TQ          TQView
}

func (e *Engine) ObserveSnapshot() SnapshotView {
	publication := e.publication.Load()
	if publication == nil {
		return SnapshotView{}
	}
	return SnapshotView{
		Publication: livePublicationView(publication),
		Operational: operationalViewFromPublication(publication),
		TQ:          cloneTQView(publication.tq),
	}
}
