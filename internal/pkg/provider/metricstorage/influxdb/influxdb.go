/*
 * Copyright (C) 2019 Nalej - All Rights Reserved
 */

package influxdb

// InfluxDB Metric Storage provider

import (
	"encoding/json"
	"fmt"
	"time"

	influx "github.com/influxdata/influxdb1-client/v2"

	"github.com/nalej/derrors"

	"github.com/nalej/edge-controller/internal/pkg/entities"
	"github.com/nalej/edge-controller/internal/pkg/provider/metricstorage"

	"github.com/rs/zerolog/log"
)

const InfluxDBProviderType metricstorage.ProviderType = "influxdb"

type InfluxDBProvider struct {
	config *influx.HTTPConfig
	client influx.Client

	database string
}

func init() {
	metricstorage.Register(InfluxDBProviderType, NewInfluxDBProvider)
}

func NewInfluxDBProvider(conf *metricstorage.ConnectionConfig) (metricstorage.Provider, derrors.Error) {
	influxConfig := &influx.HTTPConfig{
		Addr: conf.Address,
	}

	i := &InfluxDBProvider{
		config: influxConfig,
		database: conf.Database,
	}

	return i, nil
}

// Create a connection to the storage system. All relevant information
// should be passed when creating the provider instance
func (i *InfluxDBProvider) Connect() derrors.Error {
	log.Debug().Str("address", i.config.Addr).Msg("connecting to influxdb")
	client, err := influx.NewHTTPClient(*i.config)
	if err != nil {
		return derrors.NewUnavailableError("unable to connect to influxdb", err).WithParams(i.config.Addr)
	}

	i.client = client

	return nil
}

// Disconnect from the storage system
func (i *InfluxDBProvider) Disconnect() derrors.Error {
	err := i.client.Close()
	if err != nil {
		return derrors.NewInternalError("unable to disconnect from influxdb", err).WithParams(i.config.Addr)
	}
	i.client = nil

	return nil
}

// Check if there is a connection
func (i *InfluxDBProvider) Connected() bool {
	return i.client != nil
}

// Create the schema needed to store metrics data. Returns an error if
// any of the entities already exist, unless `ifNeeded` is set.
func (i *InfluxDBProvider) CreateSchema(ifNeeded bool) derrors.Error {
	// Check if exists
	response, err := i.query(queryShowDatabases)
	if err != nil {
		return derrors.NewUnavailableError("unable to get list of databases", err)
	}
	found := false
	for _, db := range(getFirstValues(response)) {
		if db[0] == i.database {
			found = true
			break
		}
	}

	if found {
		if !ifNeeded {
			return derrors.NewInvalidArgumentError("database already exists").WithParams(i.database)
		}
		return nil
	}

	response, err = i.query(fmt.Sprintf(queryCreateDatabase, i.database))
	if err != nil {
		return derrors.NewUnavailableError("unable to create database", err).WithParams(i.database)
	}

	// TODO: retention policies
	return nil
}

// Store metrics
func (i *InfluxDBProvider) StoreMetricsData(metrics *entities.MetricsData, extraTags map[string]string) derrors.Error {
	// Create structure to add received metrics in bulk
	bp, err := influx.NewBatchPoints(influx.BatchPointsConfig{
		// We don't retrieve metrics with more than a second precision
		Precision: "s",
		Database: i.database,
		// TODO: RetentionPolicy
	})
	if err != nil {
		return derrors.NewInternalError("error creating batchpoints", err)
	}

	for _, metric := range(metrics.Metrics) {
		fields := make(map[string]interface{}, len(metric.Fields))
		for k, v := range(metric.Fields) {
			fields[k] = int64(v)
		}
		for k, v := range(extraTags) {
			// I think it's ok to modify the metrics data, we don't
			// use it elsewhere
			metric.Tags[k] = v
		}
		point, err := influx.NewPoint(metric.Name, metric.Tags, fields, metrics.Timestamp)
		if err != nil {
			return derrors.NewInternalError("error creating point", err)
		}
		bp.AddPoint(point)
	}

	err = i.client.Write(bp)
	if err != nil {
		log.Error().Err(err).Msg("error writing to influxdb")
		return derrors.NewUnavailableError("error writing to influxdb", err)
	}

	return nil
}

// List available metrics. If tagSelector is empty, return all available,
// if tagSelector contains key-value pairs, return metrics available
// for the union of those tags
func (i *InfluxDBProvider) ListMetrics(tagSelector entities.TagSelector) ([]string, derrors.Error) {
	where := whereClause([]string{whereClauseFromTags(tagSelector)})
	response, err := i.query(fmt.Sprintf(queryListMetrics, where))
	if err != nil {
		return nil, derrors.NewUnavailableError("unable to list metrics", err)
	}

	return getMetrics(response), nil
}

// Query specific metric. If tagSelector is empty, return all values
// available, aggregated with aggr. If tagSelector is contains
// key-value pairs, return values for the union of those tags,
// aggregated with aggr. If tagSelector contains a single entry,
// values for that specific tag are returned and aggr is ignored.
func (i *InfluxDBProvider) QueryMetric(metric string, tagSelector entities.TagSelector, timeRange *entities.TimeRange, aggr entities.AggregationMethod) ([]entities.MetricValue, derrors.Error) {
	// TODO: Pre-process diskio_read, diskio_write

	query := generateQuery(metric, tagSelector, timeRange, aggr)
	log.Debug().Str("query", query).Msg("generated query")

	response, err := i.query(query)
	if err != nil {
		log.Error().Err(err).Msg("influxdb query error")
		return nil, derrors.NewInternalError("error executing influx query", err)
	}

	return metricValuesFromResponse(response)
}

func metricValuesFromResponse(response *influx.Response) ([]entities.MetricValue, derrors.Error) {
	values := getFirstValues(response)
	result := make([]entities.MetricValue, 0, len(values))
	for _, v := range(values) {
		timestamp, derr := timestampFromInterface(v[0])
		if derr != nil {
			return nil, derr
		}
		value, derr := valueFromInterface(v[1])
		if derr != nil {
			return nil, derr
		}
		result = append(result, entities.MetricValue{
			Timestamp: timestamp,
			Value: value,
		})
	}

	return result, nil
}

func timestampFromInterface(i interface{}) (time.Time, derrors.Error) {
	tsString, ok := i.(string)
	if !ok {
		return time.Time{}, derrors.NewInternalError("error retrieving value").WithParams(i)
	}
	timestamp, err := time.Parse(time.RFC3339, tsString)
	if err != nil {
		return time.Time{}, derrors.NewInternalError("error parsing timestamp", err).WithParams(tsString)
	}

	return timestamp, nil
}

func valueFromInterface(i interface{}) (int64, derrors.Error) {
	jsonValue, ok := i.(json.Number)
	if !ok {
		return 0, derrors.NewInternalError("error retrieving value").WithParams(i)
	}
	// First try to get int; if that fails, get float and convert to int
	value, err := jsonValue.Int64()
	if err != nil {
		fvalue, err := jsonValue.Float64()
		if err != nil {
			return 0, derrors.NewInternalError("error converting result to int", err).WithParams(jsonValue)
		}
		value = int64(fvalue)
	}

	return value, nil
}

func (i *InfluxDBProvider) query(q string) (*influx.Response, error) {
	query := influx.NewQuery(q, i.database, "")
	response, err := i.client.Query(query)
	if err == nil {
		err = response.Error()
	}
	return response, err
}

func getFirstValues(response *influx.Response) [][]interface{} {
	if response == nil || len(response.Results) == 0 {
		return nil
	}
	results := response.Results[0]

	if len(results.Series) == 0 {
		return nil
	}
	return results.Series[0].Values
}

// Some InfluxDB measurements are exposed as a separate read and write metric,
// each mapping to the same measurement but a different field.
var rwMetrics = map[string]bool{
	"diskio": true,
	"net": true,
}

func getMetrics(response *influx.Response) []string {
	values := getFirstValues(response)
	list := make([]string, 0, len(values))
	for _, v := range(values) {
		strVal := v[0].(string)
		if rwMetrics[strVal] {
			list = append(list, strVal + readSuffix)
			list = append(list, strVal + writeSuffix)
		} else {
			list = append(list, strVal)
		}
	}

	return list
}
