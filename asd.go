package main

import (
	"fmt"

	"github.com/Dynatrace/dynatrace-operator/controllers/csi/storage"
)

func main() {
	access := storage.NewAccess()
	v, e := access.GetLatestVersion("asd")
	if v == "" && e == nil {
		fmt.Print("AD")
	}
}
