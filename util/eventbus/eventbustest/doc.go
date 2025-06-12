// Copyright (c) Tailscale Inc & AUTHORS
// SPDX-License-Identifier: BSD-3-Clause

// Package eventbustest provides helper methods for testing the
// package eventbus.
//
// An event bus connects publishers of typed events with subscribers
// interested in those events. Typically, there is one global event
// bus per process.
//
// # Usage
//
// The test helper presents a set of generic ideas for testing and a
// few specific helpers.
//
// Generally, tests can be run as:
//
//	bus := eventbus.New()
//	defer bus.Close()
//
//	tw := bus.Debugger().NewTestWatcher()
//	defer tw.Done()
//
//	somethingThatEmitsSomeEvent()
//	if err := eventbus.Expect[EventFoo](tw); err != nil {
//	  t.Error(err.Error())
//	}
//
// This example is also the given example for the simples helper that
// expects a particular event to have happened on the bus.
//
// The helpers are:
//
// Expect: Expect a given event of type
//
//	eventbustest.Expect[T any](tw *eventbustest.TestWatcher) error
//
// ExpectFunc: Expect a given even of type, with func to check event values.
// Expect is implemented using ExpectFunc and a func of `{ return true }`.
//
//	eventbustest.ExpectFunc[T any](tw *eventbustest.TestWatcher, func(event T) (bool, error)) error
//
// ExpectEvents: Expect a given set of events allowing other events interspersed
//
//	eventbustest.ExpectEvents(tw *eventbustest.TestWatcher, events ...any) error
//
// ExpectOnlyEvents: Expect a given set of events but now allowing other events interspersed
//
//	eventbustest.ExpectOnlyEvents(tw *eventbustest.TestWatcher, events ...any) error
//
// The last two helpers uses the following struct as a tuple:
//
//	type EventFunc struct {
//		Event any
//		F     func(event any) (bool, error)
//	}
//
// ExpectEventsFunc: Similar to ExpectEvents but allowing checking of the event values.
//
//	eventbustest.ExpectEventsFunc(tw *eventbustest.TestWatcher, events ...eventbustest.EventFunc) error {
//
// ExpectOnlyEventsFunc: Similar to ExpectOnlyEvents but allowing checking of the event values.
//
//	eventbustest.ExpectOnlyEventsFunc(tw *eventbustest.TestWatcher, events ...eventbustest.EventFunc) error {

package eventbustest
