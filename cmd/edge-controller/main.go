/*
 * Copyright (C) 2019 Nalej - All Rights Reserved
 */

package main

import (
	"github.com/nalej/edge-controller/cmd/edge-controller/commands"
	"github.com/nalej/edge-controller/version"
)

// MainVersion with the application version.
var MainVersion string
// MainCommit with the commit id.
var MainCommit string

func main() {
	version.AppVersion = MainVersion
	version.Commit = MainCommit
	commands.Execute()
}
