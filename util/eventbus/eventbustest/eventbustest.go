// Copyright (c) Tailscale Inc & AUTHORS
// SPDX-License-Identifier: BSD-3-Clause

package eventbustest

import (
	"fmt"
	"reflect"
	"time"

	"tailscale.com/util/eventbus"
)

func NewTestWatcher(d *eventbus.Debugger) *TestWatcher {
	tw := &TestWatcher{
		mon:     d.WatchBus(),
		TimeOut: 5 * time.Second,
		done:    make(chan bool, 1),
		events:  make(chan any, 100),
	}
	go tw.watch()
	return tw
}

type TestWatcher struct {
	mon     *eventbus.Subscriber[eventbus.RoutedEvent]
	events  chan any
	done    chan bool
	TimeOut time.Duration
}

// Expect is a particular implementation of ExpectFunc that tests for the
// existence of an Event without caring about the contents of it.
//
// Example usage:
//
//	bus := eventbus.New()
//	defer bus.Close()
//
//	tw := bus.Debugger().NewTestWatcher()
//	defer tw.Done()
//
//	somethingThatEmitsSomeEvent()
//	if err := eventbustest.Expect[EventFoo](tw); err != nil {
//	  t.Error(err.Error())
//	}
func Expect[T any](tw *TestWatcher) error {
	return ExpectFunc(tw, func(event T) (bool, error) { return true, nil })
}

// ExpectFunc tests for a particular event but also a particular shape of said event.
// This allows for looking for a single event with a specific value, or a
// particular event in a pile of SomeEvent.
//
// Example usage of looking for one event with a specific value:
//
//	bus := eventbus.New()
//	defer bus.Close()
//
//	tw := bus.Debugger().NewTestWatcher()
//	defer tw.Done()
//
//	somethingThatEmitsSomeEvent()
//	expectedValue := 42
//	if err := eventbustest.ExpectFunc(tw, func(event SomeEvent) (bool, error) {
//		if event.Value != expectedValue {
//			return false, t.Errorf("expected %v, got %v", expected, event.External)
//		}
//		return true, nil
//	}); err != nil {
//	  t.Error(err.Error())
//	}
func ExpectFunc[T any](tw *TestWatcher, test func(event T) (bool, error)) error {
	eventCount := 0
	for {
		select {
		case event := <-tw.events:
			eventCount = eventCount + 1
			if ev, ok := event.(T); ok {
				if ok, err := test(ev); ok {
					return nil
				} else if err != nil {
					return err
				}
			}
		case <-time.After(tw.TimeOut):
			return fmt.Errorf("timed out waiting for event, saw %d events", eventCount)
		}
	}
}

// ExpectEvents tests for a particular set of events in an order but other events
// can happen at the same time. The contents of the events will not be tested,
// only the event type itself.
//
// Example usage:
//
//	bus := eventbus.New()
//	defer bus.Close()
//
//	tw := bus.Debugger().NewTestWatcher()
//	defer tw.Done()
//
//	somethingThatEmitsSomeEvent()
//	somethingThatEmitsNotRelevantEvent()
//	somethingThatEmitsAnotherEvent()
//	// This will return nil / no errors
//	if err := eventbustest.ExpectEvents(tw, SomeEvent{}, AnotherEvent{}); err != nil {
//	  t.Error(err.Error())
//	}
func ExpectEvents(tw *TestWatcher, events ...any) error {
	eventCount := 0
	head := 0
	for head < len(events) {
		select {
		case event := <-tw.events:
			eventCount = eventCount + 1
			typeEvent := reflect.TypeOf(event)
			typeHead := reflect.TypeOf(events[head])
			if typeEvent == typeHead {
				head = head + 1
			}
		case <-time.After(tw.TimeOut):
			return fmt.Errorf(
				"timed out waiting for event, saw %d events, %d was expected",
				eventCount, head)
		}
	}
	return nil
}

// ExpectOnlyEvents tests for a particular set of events in an order but other events
// are not allowed to happen at the same time. The contents of the events will not be tested,
// only the event type itself.
//
// Example usage:
//
//	bus := eventbus.New()
//	defer bus.Close()
//
//	tw := bus.Debugger().NewTestWatcher()
//	defer tw.Done()
//
//	somethingThatEmitsSomeEvent()
//	somethingThatEmitsNotRelevantEvent()
//	somethingThatEmitsAnotherEvent()
//	// This will return an error as the second event is not AnotherEvent{}
//	if err := eventbustest.ExpectEvents(tw, SomeEvent{}, AnotherEvent{}); err != nil {
//	  t.Error(err.Error())
//	}
func ExpectOnlyEvents(tw *TestWatcher, events ...any) error {
	eventCount := 0
	head := 0
	for head < len(events) {
		select {
		case event := <-tw.events:
			eventCount = eventCount + 1
			typeEvent := reflect.TypeOf(event)
			typeHead := reflect.TypeOf(events[head])
			if typeEvent != typeHead {
				return fmt.Errorf(
					"expected event type %s, saw %s, at index %d",
					typeHead, typeEvent, head)
			}
			head = head + 1
		case <-time.After(tw.TimeOut):
			return fmt.Errorf(
				"timed out waiting for event, saw %d events, %d was expected",
				eventCount, head)
		}
	}
	return nil
}

// EventFunc is a test helper type that holds an event an a function to test the
// validity of that event. F returns indicates:
//
//   - true, nil - Event matches, continue processing
//   - false, nil - Event does not match, skip and continue processing
//   - *, error - Stop processing, return error
//
// See ExpectEventsFunc for example.
type EventFunc struct {
	Event any
	F     func(event any) (bool, error)
}

// ExpectEventsFunc checks for some number of events showing up on the event bus
// in a given order, disregarding any events happening in between.
// If the list of events must match exactly with no extra events,
// use ExpectOnlyEventsFunc
//
// Example usage testing values non strictly:
//
//	bus := eventbus.New()
//	defer bus.Close()
//
//	tw := bus.Debugger().NewTestWatcher()
//	defer tw.Done()
//
//	somethingThatEmitsSomeEvent()
//	somethingThatEmitsNotRelevantEvent()
//	somethingThatEmitsAnotherEvent()
//
//	some := eventbustest.EventFunc{SomeEvent{}, func(event any) (bool, error){
//		ev := event.(SomeEvent)
//		if ev.Value == 42 {
//			return true, nil
//		}
//		return false, nil
//	}}
//	another := eventbustest.EventFunc{AnotherEvent{}, func(event any) (bool, error){
//		ev := event.(AnotherEvent)
//		if ev.Value == "42" {
//			return true, nil
//		}
//		return false, nil
//	}}
//	// This not return an error as the expected events are showing up.
//	if err := eventbustest.ExpectEventsFunc(tw, some, another); err != nil {
//	  t.Error(err.Error())
//	}
func ExpectEventsFunc(tw *TestWatcher, events ...EventFunc) error {
	eventCount := 0
	head := 0
	for head < len(events) {
		select {
		case event := <-tw.events:
			eventCount = eventCount + 1
			typeEvent := reflect.TypeOf(event)
			typeHead := reflect.TypeOf(events[head].Event)
			if typeEvent == typeHead {
				if ok, err := events[head].F(event); err != nil {
					return err
				} else if ok {
					head = head + 1
				}
			}
		case <-time.After(tw.TimeOut):
			return fmt.Errorf(
				"timed out waiting for event, saw %d events, %d was expected",
				eventCount, head)
		}
	}
	return nil
}

// ExpectOnlyEventsFunc checks for some number of events showing up on the event bus
// in a given order, returning an error if the events does not match the given list
// exactly. Use ExpectEventsFunc if other events are allowed.
//
// Example usage is similar to the one of ExpectEventsFunc, except ExpectOnlyEventsFunc
// would return an error in that given scenario a non listed event shows up.
func ExpectOnlyEventsFunc(tw *TestWatcher, events ...EventFunc) error {
	eventCount := 0
	head := 0
	for head < len(events) {
		select {
		case event := <-tw.events:
			eventCount = eventCount + 1
			typeEvent := reflect.TypeOf(event)
			typeHead := reflect.TypeOf(events[head].Event)
			if typeEvent != typeHead {
				return fmt.Errorf(
					"expected event type %s, saw %s, at index %d",
					typeHead, typeEvent, head)
			}
			if ok, err := events[head].F(event); err != nil {
				return err
			} else if !ok {
				return fmt.Errorf(
					"expected test ok for type %s, at index %d", typeHead, head)
			}
			head = head + 1
		case <-time.After(tw.TimeOut):
			return fmt.Errorf(
				"timed out waiting for event, saw %d events, %d was expected",
				eventCount, head)
		}
	}
	return nil
}

func (tw *TestWatcher) watch() {
	for {
		select {
		case event := <-tw.mon.Events():
			tw.events <- event.Event
		case <-tw.done:
			tw.mon.Close()
			return
		}
	}
}

func (tw *TestWatcher) Done() {
	tw.done <- true
}
