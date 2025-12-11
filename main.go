package main

import (
	_ "encoding/json"
	_ "fmt"
	"kps-migration-api/api"
	_ "kps-migration-api/api"
	_ "kps-migration-api/docs"
	_ "net/http"

	_ "github.com/rclone/rclone/librclone/librclone"

	_ "github.com/rclone/rclone/backend/all"

	// import all backends
	_ "github.com/rclone/rclone/cmd/cmount"

	// import cmount
	_ "github.com/rclone/rclone/cmd/mount"

	// import mount
	_ "github.com/rclone/rclone/cmd/mount2"

	// import mount2
	_ "github.com/rclone/rclone/fs/operations"

	// import operations/* rc commands
	_ "github.com/rclone/rclone/fs/sync"

	// import sync/*
	_ "github.com/rclone/rclone/lib/plugin"
)

func main() {
	println("Let's start to kps migration api")

	//go
	api.ProcessREST()
}
