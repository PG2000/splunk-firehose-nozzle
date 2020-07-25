package nozzle_test

import (
	"time"

	"code.cloudfoundry.org/lager"

	. "github.com/cloudfoundry-community/splunk-firehose-nozzle/nozzle"
	"github.com/cloudfoundry/sonde-go/events"
	"github.com/gorilla/websocket"

	. "github.com/onsi/ginkgo"
	. "github.com/onsi/gomega"

	"github.com/cloudfoundry-community/splunk-firehose-nozzle/testing"
)

var _ = Describe("Nozzle", func() {
	var (
		eventSource *testing.MemoryEventSourceMock
		eventRouter *testing.EventRouterMock
		nozzle      *Nozzle
	)

	Context("When there are no errors from event source", func() {
		BeforeEach(func() {
			eventSource = testing.NewMemoryEventSourceMock(-1, int64(10), -1)
			eventRouter = testing.NewEventRouterMock()
			config := &Config{
				Logger: lager.NewLogger("test"),
			}
			nozzle = New(eventSource, eventRouter, config)
		})

		It("collects events from source and routes to sink", func() {
			go nozzle.Start()

			time.Sleep(time.Second)

			Eventually(func() []*events.Envelope {
				return eventRouter.Events()
			}).Should(HaveLen(10))
		})

		It("EventSource close", func() {
			go nozzle.Start()
			time.Sleep(time.Second)
			eventSource.Close()
			time.Sleep(time.Second)
			nozzle.Close()
		})
	})

	prepare := func(closeErr int) func() {
		return func() {
			eventSource = testing.NewMemoryEventSourceMock(-1, int64(10), closeErr)
			eventRouter = testing.NewEventRouterMock()
			config := &Config{
				Logger: lager.NewLogger("test"),
			}
			nozzle = New(eventSource, eventRouter, config)
		}
	}

	runAndAssert := func(closeErr int) func() {
		return func() {
			done := make(chan error, 1)
			go func() {
				err := nozzle.Start()
				done <- err
			}()

			time.Sleep(time.Second)
			nozzle.Close()

			err := <-done
			if ce, ok := err.(*websocket.CloseError); ok {
				Expect(ce.Code).To(Equal(closeErr))
			} else {
				Expect(err).To(Equal(testing.MockupErr))
			}
		}
	}

	Context("When there is websocket.CloseNormalClosure from event source", func() {
		BeforeEach(prepare(websocket.CloseNormalClosure))
		It("handles errors when collects events from source", runAndAssert(websocket.CloseNormalClosure))
	})

	Context("When there is websocket.ClosePolicyViolation from event source", func() {
		BeforeEach(prepare(websocket.ClosePolicyViolation))
		It("handles errors when collects events from source", runAndAssert(websocket.ClosePolicyViolation))
	})

	Context("When there is websocket.CloseGoingAway from event source", func() {
		BeforeEach(prepare(websocket.CloseGoingAway))
		It("handles errors when collects events from source", runAndAssert(websocket.CloseGoingAway))
	})

	Context("When there is other error from event source", func() {
		BeforeEach(prepare(0))
		It("handles errors when collects events from source", runAndAssert(0))
	})

})
