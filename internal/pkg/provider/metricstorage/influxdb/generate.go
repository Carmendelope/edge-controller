/*
 * Copyright (C) 2019 Nalej - All Rights Reserved
 */

package influxdb

// Generate queries

import (
	"fmt"
	"strings"
	"time"

	"github.com/nalej/edge-controller/internal/pkg/entities"
)

const (
	readSuffix = "_read"
	writeSuffix = "_write"
)

var fromOverrides = map[string]string{
	// Calculate millicores used
	"cpu": "(SELECT round((1-time_idle/(time_user+time_system+time_nice+time_iowait+time_irq+time_softirq+time_steal+time_idle))*1000) AS usage, asset_id, cpu FROM cpu)",
	"diskio_read": "diskio",
	"diskio_write": "diskio",
	"net_read": "net",
	"net_write": "net",
}

var metricFields = map[string]string{
	"cpu": "usage",
	"mem": "used",
	"disk": "used",
	"diskio_read": "read_bytes",
	"diskio_write": "write_bytes",
	"net_read": "bytes_recv",
	"net_write": "bytes_sent",
}

var sumTags = map[string]string{
	"cpu": "cpu",
	"disk": "device",
	"diskio_read": "name",
	"diskio_write": "name",
	"net_read": "interface",
	"net_write": "interface",
}

var derivativeMetric = map[string]bool{
	"diskio_read": true,
	"diskio_write": true,
	"net_read": true,
	"net_write": true,
}

// If we need more flexibility than the queries this function can generate,
// we probably want to create something similar to a query tree
func generateQuery(metric string, tagSelector entities.TagSelector, timeRange *entities.TimeRange, aggr entities.AggregationMethod) string {
	// Determine what to select from. Mostly just a measurement,
	// but sometimes (e.g., for CPU), we do some pre-processing
	from, found := fromOverrides[metric]
	if !found {
		from = metric
	}
	fromClause := fmt.Sprintf("FROM %s", from)

	// Add restrictions in time and asset_id
	whereClause := whereClause([]string{
		whereClauseFromTime(timeRange),
		whereClauseFromTags(tagSelector),
	})

	// Determine what field our main metric is
	metricValue, found := metricFields[metric]
	if !found {
		// TBD
	}

	// First iteration of complete select
	selector := "metric"
	selectClause := fmt.Sprintf("%s %s %s",
		// As we either interpolate or aggregate over time, we add a
		// "mean" here.
		selectFromFuncFieldAs("mean", metricValue, selector),
		fromClause,
		whereClause,
	)

	// If we want a single point in time, we set resolution to 1s to not
	// miss anything, and later limit to a single value
	if !timeRange.Timestamp.IsZero() {
		timeRange.Resolution = time.Second
	}

	// Add inner summation if needed (e.g., all CPUs, all disks per asset)
	sumTag, found := sumTags[metric]
	if found {
		innerGroupBy := groupByClause(timeRange.Resolution, []string{"asset_id", sumTag}...)
		newSelector := "summed_metric"
		outerSelect := selectFromFuncFieldAs("sum", selector, newSelector)
		selector = newSelector
		selectClause = fmt.Sprintf("%s FROM (%s %s)", outerSelect, selectClause, innerGroupBy)
	}

	// Add time and asset grouping. A resolution of 0 aggregates over
	// complete time range and returns a single value per asset
	selectClause = fmt.Sprintf("%s %s", selectClause, groupByClause(timeRange.Resolution, "asset_id"))

	// Aggregate over assets. If we have a single asset this is a no-op
	if aggr != entities.AggregateNone {
		newSelector := "aggr_metric"
		selectClause = fmt.Sprintf("%s FROM (%s) %s",
			selectFromFuncFieldAs(aggr.String(), selector, newSelector),
			selectClause,
			groupByClause(timeRange.Resolution),
		)
		selector = newSelector
	}

	// For throughput metrics (x per sec) we need a derivative
	if derivativeMetric[metric] {
		newSelector := "derv_metric"
		selectClause = fmt.Sprintf("%s FROM (%s)",
			selectFromFuncFieldAs("derivative", selector, newSelector),
			selectClause,
		)
		selector = newSelector
	}

	// Now that we've summed and aggregated by asset, we can limit the
	// result for a single point in time
	if !timeRange.Timestamp.IsZero() {
		selectClause = fmt.Sprintf("SELECT last(%s) FROM (%s)", selector, selectClause)
	}

	return selectClause
}

func whereClauseFromTags(tags map[string][]string) string {
	clauses := make([]string, 0, len(tags))
	for tag, values := range(tags) {
		for _, value := range(values) {
			clauses = append(clauses, fmt.Sprintf("\"%s\"='%s'", tag, value))
		}
	}

	if len(clauses) == 0 {
		return ""
	}

	return fmt.Sprintf("(%s)", strings.Join(clauses, " OR "))
}

func whereClause(subclauses []string) string {
	trimmed := make([]string, 0, len(subclauses))
	for _, s := range(subclauses) {
		if s == "" {
			continue
		}
		trimmed = append(trimmed, s)
	}

	clause := strings.Join(trimmed, " AND ")
	if len(clause) == 0 {
		return ""
	}

	return fmt.Sprintf("WHERE %s", clause)
}

func whereClauseFromTime(timeRange *entities.TimeRange) string {
	start := timeRange.Start
	end := timeRange.End

	// single point in time actually will be average over a
	// range to avoid having no data during that time
	if !timeRange.Timestamp.IsZero() {
		end = timeRange.Timestamp
	}

	clauses := make([]string, 0, 2)
	if !start.IsZero() {
		clauses = append(clauses, fmt.Sprintf("time >= %d", start.UnixNano()))
	}
	if !end.IsZero() {
		clauses = append(clauses, fmt.Sprintf("time <= %d", end.UnixNano()))
	}

	return fmt.Sprintf("(%s)", strings.Join(clauses, " AND "))
}

func selectFromFieldAs(f string, as string) string {
	return fmt.Sprintf("SELECT %s AS %s", f, as)
}

func selectFromFuncFieldAs(fn string, field string, as string) string {
	if fn != "" {
		field = fmt.Sprintf("%s(%s)", fn, field)
	}
	return selectFromFieldAs(field, as)
}

func groupByClause(resolution time.Duration, extraTags ...string) string {
	tags := []string{
		fmt.Sprintf("time(%s)", resolution.String()),
	}
	// Tags need to be in quotes in case of reserved keywords
	for _, tag := range(extraTags) {
		tags = append(tags, fmt.Sprintf("\"%s\"", tag))
	}

	return fmt.Sprintf("GROUP BY %s fill(none)", strings.Join(tags, ","))
}
