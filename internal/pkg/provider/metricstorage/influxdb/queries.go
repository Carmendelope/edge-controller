/*
 * Copyright (C) 2019 Nalej - All Rights Reserved
 */

package influxdb

// Pre-defined queries, potentially with arguments

const (
	queryShowDatabases = "SHOW DATABASES"
	queryCreateDatabase = "CREATE DATABASE %s WITH DURATION %s REPLICATION 1 NAME %s" // database name, data retention, retention policy name

	// retention policy name, database name, retention policy duration, shard duration
	queryAlterRetentionPolicy = "ALTER RETENTION POLICY %s ON %s DURATION %s SHARD DURATION %s"

	queryListMetrics = "SHOW MEASUREMENTS %s" // tags where clause
)

