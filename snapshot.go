package eventhorizon

type Snapshot interface {
	RawDataI() interface{}
}
