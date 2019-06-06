/*
 * Copyright (C) 2019 Nalej - All Rights Reserved
 */

package influxdb

// InfluxDB Metric Storage provider

import (
	"fmt"

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
func (i *InfluxDBProvider) ListMetrics(tagSelector map[string]string) ([]string, derrors.Error) {
	where := whereClause([]string{whereClauseFromTags(tagSelector)})
	response, err := i.query(fmt.Sprintf(queryListMetrics, where))
	if err != nil {
		return nil, derrors.NewUnavailableError("unable to list metrics", err)
	}

	return getFirstValueStrings(response), nil
}

// Query specific metric. If tagSelector is empty, return all values
// available, aggregated with aggr. If tagSelector is contains
// key-value pairs, return values for the union of those tags,
// aggregated with aggr. If tagSelector contains a single entry,
// values for that specific tag are returned and aggr is ignored.
func (i *InfluxDBProvider) QueryMetric(metric string, tagSelector map[string]string, timeRange metricstorage.TimeRange, aggr metricstorage.AggregationMethod) ([]metricstorage.Value, derrors.Error) {
	return nil, nil
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

func getFirstValueStrings(response *influx.Response) []string {
	values := getFirstValues(response)
	list := make([]string, 0, len(values))
	for _, v := range(values) {
		list = append(list, v[0].(string))
	}

	return list
}
