package main

// Reference:
// This code is based on the Talos StderrReporter originally licensed under the MPL.
// "github.com/siderolabs/talos/pkg/reporter"

import (
	"fmt"
	"strings"

	"github.com/siderolabs/talos/pkg/conditions"
)

// ConditionReporter is a reporter that reports conditions to a reporter.Reporter.
type ConditionReporter struct {
	//w *reporter.Reporter
	l        *UILogger
	lastLine string
}

// Update reports a condition to the reporter.
func (r *ConditionReporter) Update(condition conditions.Condition) {
	conditionToUpdate(r, condition)
}

// UILoggerReporter returns a reporter with Bubbletea logger output
func UILoggerReporter(logger *UILogger) *ConditionReporter {
	return &ConditionReporter{
		l: logger,
	}
}

func conditionToUpdate(r *ConditionReporter, condition conditions.Condition) {
	line := strings.TrimSpace(fmt.Sprintf("waiting for %s", condition.String()))
	if r.lastLine == line {
		return
	}

	switch {
	case strings.HasSuffix(line, "..."):
		r.l.Info(line)
	case strings.HasSuffix(line, conditions.OK):
		r.l.Successf("%s", line)
	case strings.HasSuffix(line, conditions.ErrSkipAssertion.Error()):
		r.l.Warnf("%s [SKIPPED]", line)
	default:
		r.l.Errorf("%s FAILED", line)
	}
	r.lastLine = line
}
