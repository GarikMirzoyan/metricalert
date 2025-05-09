package main

import (
	"github.com/GarikMirzoyan/metricalert/internal/agent"
	"github.com/GarikMirzoyan/metricalert/internal/agent/config"
)

func main() {
	config := config.InitConfig()

	agent := agent.NewAgent(config)
	agent.Run()
}
