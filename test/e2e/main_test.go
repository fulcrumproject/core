package e2e

import (
	"log"
	"testing"
)

func TestMain(t *testing.T) {
	adminToken, err := GetToken("admin1", "admin1")
	if err != nil {
		log.Fatalf("admin token: %v", err)
	}
	NewClient(adminToken)

}
