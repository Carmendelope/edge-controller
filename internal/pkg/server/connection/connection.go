/*
 * Copyright (C) 2018 Nalej - All Rights Reserved
 */

// This package provides connectivity to a node to execute commands and
// copy files through various methods.

package connection

import (
	"encoding/json"
	"fmt"
)

// Connection interface for the various methods of connecting to a node
type Connection interface {
	// TODO: Add connect / disconnect to be able to run multiple commands over a single connection

	// Execute a single command on a node capturing stdout
	Execute(command string) ([]byte, error)

	// Copy a single file to or from a node
	Copy(lpath string, rpath string, remoteSource bool) error

	// Get online status
	IsOnline() (bool, error)
}

// ConnectionType to define the set of supported protocols.
type ConnectionType string

// ConnectionNewFunc function type for a function that returns an empty Connection
type ConnectionNewFunc func() Connection

// All connection types and their NewFunc
// This is used in unmarshalling a connection to get an empty struct
// of the right implementation to unmarshal into
var connectionTypeMap = map[ConnectionType]ConnectionNewFunc{}

// AddConnectionType defines a function that registers a new connection type in the map.
func AddConnectionType(connType ConnectionType, newFunc ConnectionNewFunc) {
	connectionTypeMap[connType] = newFunc
}

// ConnectionJSON type is used to marshal and unmarshal the implementations correctly.
// Although we can marshal the interfaces directly, we need an actual type
// (not interface) for unmarshalling that detects the actual type and creates
// the proper implementation to the interface
type ConnectionJSON struct {
	Connection
}

// MarshalJSON is the function called by json.Marshal()
// We need this to not have the embedded Connection indirection
func (c *ConnectionJSON) MarshalJSON() ([]byte, error) {
	return json.Marshal(c.Connection)
}

// UnmarshalJSON is the function called by json.Unmarshal()
// Figure out the type and create the right Connection implementation
func (c *ConnectionJSON) UnmarshalJSON(b []byte) error {
	// Figure out type. Connection fields are ignored
	connType := &struct {
		Type ConnectionType `json:"type"`
	}{}

	err := json.Unmarshal(b, connType)
	if err != nil {
		return err
	}

	// Create right Connection.
	c.Connection, err = NewConnection(connType.Type, b)
	return err
}

// NewConnection creates a new connection of a given type.
func NewConnection(connType ConnectionType, b []byte) (Connection, error) {
	newFunc, found := connectionTypeMap[connType]
	if !found {
		err := fmt.Errorf("unknown connection type %v", connType)
		return nil, err
	}

	var connection Connection = newFunc()
	err := json.Unmarshal(b, connection)
	return connection, err
}
