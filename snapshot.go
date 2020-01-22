package eventhorizon

type Snapshot interface {
	RawDataI() interface{}
	Version() int
	AggregateType() AggregateType
	AggregateId() string
}
