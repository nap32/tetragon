// SPDX-License-Identifier: Apache-2.0
// Copyright Authors of Tetragon

package test

import (
	"context"
	"errors"
	"os"
	"sync"
	"testing"

	ec "github.com/cilium/tetragon/api/v1/tetragon/codegen/eventchecker"
	"github.com/cilium/tetragon/pkg/observer"
	"github.com/cilium/tetragon/pkg/testutils"
	"github.com/sirupsen/logrus"
)

// TestTestChecker tests the test checker
func TestTestChecker(t *testing.T) {
	if _, err := os.Stat("/sys/kernel/debug/tracing/events/syscalls"); os.IsNotExist(err) {
		t.Skip("cannot use syscall tracepoints (consider enabling CONFIG_FTRACE_SYSCALLS)")
	}

	var doneWG, readyWG sync.WaitGroup
	defer doneWG.Wait()

	ctx, cancel := context.WithTimeout(context.Background(), cmdWaitTime)
	defer cancel()

	dummyErr := errors.New("dummy error")
	dummyChecker := ec.FnEventChecker{
		NextCheckFn: func(ev ec.Event, log *logrus.Logger) (bool, error) {
			return false, nil
		},
		FinalCheckFn: func(log *logrus.Logger) error {
			return dummyErr
		},
	}
	errorChecker := NewTestChecker(&dummyChecker)

	obs, err := observer.GetDefaultObserver(t, tetragonLib)
	if err != nil {
		t.Fatalf("GetDefaultObserver error: %s", err)
	}
	sensor := GetTestSensor()
	testutils.LoadSensor(ctx, t, sensor)

	observer.LoopEvents(ctx, t, &doneWG, &readyWG, obs)
	readyWG.Wait()

	TestCheckerMarkEnd(t)

	err = observer.JsonTestCheck(t, errorChecker)
	t.Logf("got error: %v", err)
	if !errors.Is(err, dummyErr) {
		t.Fatalf("unexpected error: %v", err)
	}
	// NB: we expect a failure, so mark the file to be removed.
	testutils.DontKeepExportFile(t)
}
