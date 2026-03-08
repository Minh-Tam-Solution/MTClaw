package main

import (
	_ "time/tzdata" // embed IANA timezone database for containers without tzdata

	"github.com/Minh-Tam-Solution/MTClaw/cmd"
)

func main() {
	cmd.Execute()
}
