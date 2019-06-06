/*
 * Copyright (C) 2019 Nalej - All Rights Reserved
 */

package influxdb

// Create partial query strings for tags

import (
	"fmt"
	"strings"
)

func whereClauseFromTags(tags map[string][]string) string {
	clauses := make([]string, 0, len(tags))
	for tag, values := range(tags) {
		for _, value := range(values) {
			clauses = append(clauses, fmt.Sprintf("\"%s\"='%s'", tag, value))
		}
	}

	return strings.Join(clauses, " OR ")
}

func whereClause(subclauses []string) string {
	clause := strings.Join(subclauses, " OR ")
	if len(clause) == 0 {
		return ""
	}

	return fmt.Sprintf("WHERE %s", clause)
}
