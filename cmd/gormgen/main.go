// Code generator for GORM Gen query DAOs
// Generates type-safe database access code from domain models
package main

import (
	"fmt"

	"gorm.io/gen"

	"github.com/fulcrumproject/core/pkg/domain"
)

func main() {
	g := gen.NewGenerator(gen.Config{
		OutPath:      "./pkg/database",
		Mode:         gen.WithDefaultQuery | gen.WithQueryInterface,
		ModelPkgPath: "./pkg/domain",
	})

	g.ApplyBasic(
		domain.Participant{},
		domain.Token{},
		domain.AgentType{},
		domain.Agent{},
		domain.ServiceType{},
		domain.ServiceGroup{},
		domain.Service{},
		domain.ServiceOptionType{},
		domain.ServiceOption{},
		domain.ServicePoolSet{},
		domain.ServicePool{},
		domain.ServicePoolValue{},
		domain.Job{},
		domain.Event{},
		domain.EventSubscription{},
		domain.MetricType{},
		domain.MetricEntry{},
		domain.VaultSecret{},
	)

	g.Execute()
	fmt.Println("✓ Code generation complete")
}
