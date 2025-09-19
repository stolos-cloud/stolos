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
	l *UILogger
}

// Update reports a condition to the reporter.
func (r *ConditionReporter) Update(condition conditions.Condition) {
	conditionToUpdate(r.l, condition)
}

// StderrReporter returns console reporter with stderr output.
func UILoggerReporter(logger *UILogger) *ConditionReporter {
	return &ConditionReporter{
		l: logger,
	}
}

func conditionToUpdate(l *UILogger, condition conditions.Condition) {
	line := strings.TrimSpace(fmt.Sprintf("waiting for %s", condition.String()))

	switch {
	case strings.HasSuffix(line, "..."):
		l.Info(line)
	case strings.HasSuffix(line, conditions.OK):
		l.Successf("%s", line)
	case strings.HasSuffix(line, conditions.ErrSkipAssertion.Error()):
		l.Warnf("%s [SKIPPED]", line)
	default:
		l.Errorf("%s FAILED", line)
	}
}
