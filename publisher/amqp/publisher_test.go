package amqp

import (
	"os"
	"testing"

	"github.com/lab-box/eventhorizon/publisher/testutil"
)

func TestEventPublisher(t *testing.T) {
	// Support Wercker testing with MongoDB.
	amqpURL := os.Getenv("AMQP_URL")
	exchangeName := "TestExchange"

	publisher1, err := NewEventPublisher(amqpURL, exchangeName)
	if err != nil {
		t.Fatal("there should be no error:", err)
	}
	defer publisher1.Close()

	// Another bus to test the observer.
	publisher2, err := NewEventPublisher(amqpURL, exchangeName)
	if err != nil {
		t.Fatal("there should be no error:", err)
	}
	defer publisher2.Close()

	// Wait for subscriptions to be ready.
	<-publisher1.ready
	<-publisher2.ready

	testutil.EventPublisherCommonTests(t, publisher1, publisher2)
}
