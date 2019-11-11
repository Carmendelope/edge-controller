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

// Pre-defined queries, potentially with arguments

const (
	queryShowDatabases = "SHOW DATABASES"
	queryCreateDatabase = "CREATE DATABASE %s WITH DURATION %s REPLICATION 1 NAME %s" // database name, data retention, retention policy name

	// retention policy name, database name, retention policy duration, shard duration
	queryAlterRetentionPolicy = "ALTER RETENTION POLICY %s ON %s DURATION %s SHARD DURATION %s"

	queryListMetrics = "SHOW MEASUREMENTS %s" // tags where clause
)

