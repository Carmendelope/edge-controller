/*
 * Copyright 2019 Nalej
 *
 * Licensed under the Apache License, Version 2.0 (the "License");
 * you may not use this file except in compliance with the License.
 * You may obtain a copy of the License at
 *
 *     http://www.apache.org/licenses/LICENSE-2.0
 *
 * Unless required by applicable law or agreed to in writing, software
 * distributed under the License is distributed on an "AS IS" BASIS,
 * WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
 * See the License for the specific language governing permissions and
 * limitations under the License.
 */

package influxdb

// Generate queries

import (
	"fmt"
	"strings"
	"time"

	"github.com/nalej/derrors"

	"github.com/nalej/edge-controller/internal/pkg/entities"
)

const (
	readSuffix = "_read"
	writeSuffix = "_write"

	// Time window in seconds used for point-in-time queries
	// See comment in generateQuery()
	defaultMetricsWindow = 60
)

var fromOverrides = map[string]string{
	// Calculate millicores used as the ratio of difference in idle
	// ticks and differenc in total ticks
	"cpu": "(SELECT round((1-difference_time_idle/(difference_time_user+difference_time_system+difference_time_nice+difference_time_iowait+difference_time_irq+difference_time_softirq+difference_time_steal+difference_time_idle))*1000) AS usage FROM (SELECT difference(*) FROM cpu))",
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
// Also, I _just_ found out about Flux, which might be a much more suitable
// query language for our purpose...
func generateQuery(metric string, tagSelector entities.TagSelector, timeRange *entities.TimeRange, aggr entities.AggregationMethod) (string, derrors.Error) {
	// We at first use a time window of 60s to aggregate over. Once
	// we've done all our calculations, we apply the requested
	// window. We do this so the averages over a time range are
	// correct with respect to the time when assets where alive.
	// An example:
	// - Asset A is alive from t1-t2, average for metric M is x
	// - Asset B is alive from t2-t3, average for metric M is y
	// If we want to know the sum of a metric, averaged over t1-t3
	// and we use GROUP BY time(0), we would calculate the average
	// for A and for B, then add the two. We would think that
	// the sum of the average of metric M is (x+y) over t1-t3.
	// However, the sum of the average is x for t1-t2 (because
	// B was not alive), and y for t2-t3. So for t1-t3 it should
	// be ( (x+0) * (t1-t2) + (0+y) * (t2-t3) ) / (t1-t4).
	// We can get to this if we first group over a default window
	// and do our sum per time window, and in the end apply our
	// requested window (or 0 if we want the average over a time
	// period).
	resolution := time.Second * defaultMetricsWindow

	// Determine what to select from. Mostly just a measurement,
	// but sometimes (e.g., for CPU), we do some pre-processing
	from, found := fromOverrides[metric]
	if !found {
		from = metric
	}

	// Add restrictions in time and asset_id
	whereClause := whereClause([]string{
		whereClauseFromTime(timeRange),
		whereClauseFromTags(tagSelector),
	})

	// Determine what field our main metric is
	metricValue, found := metricFields[metric]
	if !found {
		return "", derrors.NewInvalidArgumentError("unsupported metric").WithParams(metric)
	}

	// For throughput metrics (x per sec) we need a derivative
	innerFunc := "mean"
	if derivativeMetric[metric] {
		metricValue = fmt.Sprintf("%s(%s),1s", innerFunc, metricValue)
		innerFunc = "derivative"
	}

	// First complete select with where clause
	selector := "metric"
	selectClause := fmt.Sprintf("%s FROM %s %s",
		selectFromFuncFieldAs(innerFunc, metricValue, selector),
		from,
		whereClause,
	)

	// Add inner summation if needed (e.g., all CPUs, all disks per asset)
	sumTag, found := sumTags[metric]
	if found {
		newSelector := "summed_metric"
		innerGroupBy := groupByClause(resolution, "asset_id", sumTag)
		selectClause = fmt.Sprintf("%s %s", selectClause, innerGroupBy)
		selectClause = fmt.Sprintf("%s FROM (%s)",
			selectFromFuncFieldAs("sum", selector, newSelector),
			selectClause,
		)
		selector = newSelector
	}

	// Add time and asset grouping. A resolution of 0 aggregates over
	// complete time range and returns a single value per asset
	selectClause = fmt.Sprintf("%s %s", selectClause, groupByClause(resolution, "asset_id"))

	// From this point onward we aggregate over assets, so we need
	// to count how many
	assetCount := ", count(asset_id) AS asset_count"

	// Aggregate over assets. If we have a single asset this is a no-op. We
	// execute anyway to make sure we have asset count
	if aggr == entities.AggregateNone {
		// We only have "none" if we select for at most a single asset
		aggr = entities.AggregateAvg
	}

	newSelector := "aggr_metric"
	selectClause = fmt.Sprintf("%s%s FROM (%s) %s",
		selectFromFuncFieldAs(aggr.String(), selector, newSelector),
		assetCount,
		selectClause,
		groupByClause(resolution),
	)
	selector = newSelector

	// Now that we've summed and aggregated by asset, we can limit the
	// result for a single point in time. We will always keep the resolution
	// at 60s. This means we get values for all assets in a 60s window.
	// This might not be the most precise, but if we make this window
	// smaller we might end up not aggregating over all assets.
	// Note that this also means that if an asset doesn't send metrics for
	// more than 60s, its values will not be included in the average. That
	// is probably reasonable, because at that moment the asset is probably
	// not available.

	// For this last aggregation we just need to propagate the asset count
	assetCount = ", last(asset_count) AS asset_count"
	if !timeRange.Timestamp.IsZero() {
		selectClause = fmt.Sprintf("SELECT last(%s)%s FROM (%s)", selector, assetCount, selectClause)
	} else if resolution != timeRange.Resolution {
		// If we used a different time window for aggregation than requested,
		// now is the time to aggregate over the requested window
		newSelector := "window_metric"
		selectClause = fmt.Sprintf("%s%s FROM (%s) %s",
			selectFromFuncFieldAs("mean", selector, newSelector),
			assetCount,
			selectClause,
			groupByClause(timeRange.Resolution),
		)
		selector = newSelector
	}

	return selectClause, nil
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

	// We always add a start time, even if it's 0. This is to work around
	// an apparent bug in InfluxDB's difference() that doesn't seem to
	// return any results without a "time >" clause.
	var startUnix int64 = 0
	if !start.IsZero() {
		// We only want to set a valid epoch-based timestamp to avoid
		// a real big negative number in the query to represent year
		// zero.
		startUnix = start.UnixNano()
	}
	clauses = append(clauses, fmt.Sprintf("time >= %d", startUnix))

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
