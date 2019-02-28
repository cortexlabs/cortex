package errors

import (
	"github.com/cortexlabs/cortex/pkg/lib/errors"
	"github.com/cortexlabs/cortex/pkg/operator/telemetry"
)

func Exit(items ...interface{}) {
	err := errors.ConsolidateErrItems(items...)
	telemetry.ReportError(err)
	errors.Exit(err)
}

func Panic(items ...interface{}) {
	err := errors.ConsolidateErrItems(items...)
	telemetry.ReportError(err)
	errors.Panic(err)
}
