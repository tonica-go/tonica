package docker_compose

import (
	"bufio"
	"fmt"
	"os"
	"strings"
	"text/template"
)

type ServiceConfig struct {
	AppDB       bool
	Temporal    bool
	Redpanda    bool
	Dragonfly   bool
	Mailhog     bool
	Jaeger      bool
}

type TemplateData struct {
	AppDB       bool
	Temporal    bool
	Redpanda    bool
	Dragonfly   bool
	Mailhog     bool
	Jaeger      bool
	HasVolumes  bool
}

func GenerateDockerCompose() error {
	config := ServiceConfig{}
	reader := bufio.NewReader(os.Stdin)

	// Ask about each service
	fmt.Println("Docker Compose Generator")
	fmt.Println("=========================")
	fmt.Println()

	config.AppDB = askYesNo(reader, "Enable App Database (PostgreSQL)?")
	config.Temporal = askYesNo(reader, "Enable Temporal (workflow engine)?")
	config.Redpanda = askYesNo(reader, "Enable Redpanda (Kafka)?")
	config.Dragonfly = askYesNo(reader, "Enable Dragonfly (Redis)?")
	config.Mailhog = askYesNo(reader, "Enable Mailhog (email testing)?")
	config.Jaeger = askYesNo(reader, "Enable Jaeger (tracing)?")

	// Generate docker-compose.yml
	err := generateComposeFile(config)
	if err != nil {
		return fmt.Errorf("failed to generate docker-compose.yml: %w", err)
	}

	fmt.Println()
	fmt.Println("✓ docker-compose.yml generated successfully!")
	fmt.Println()
	fmt.Println("To start services run:")
	fmt.Println("  docker-compose up -d")
	fmt.Println()

	return nil
}

func askYesNo(reader *bufio.Reader, question string) bool {
	fmt.Printf("%s [y/N]: ", question)
	response, err := reader.ReadString('\n')
	if err != nil {
		return false
	}
	response = strings.TrimSpace(strings.ToLower(response))
	return response == "y" || response == "yes"
}

func generateComposeFile(config ServiceConfig) error {
	data := TemplateData{
		AppDB:      config.AppDB,
		Temporal:   config.Temporal,
		Redpanda:   config.Redpanda,
		Dragonfly:  config.Dragonfly,
		Mailhog:    config.Mailhog,
		Jaeger:     config.Jaeger,
		HasVolumes: config.AppDB || config.Temporal || config.Dragonfly,
	}

	tmpl := template.Must(template.New("docker-compose").Parse(dockerComposeTemplate))

	file, err := os.Create("docker-compose.yml")
	if err != nil {
		return err
	}
	defer file.Close()

	return tmpl.Execute(file, data)
}
