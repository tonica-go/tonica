package project

import (
	"bytes"
	"fmt"
	"log/slog"
	"os"
	"os/exec"
	"strings"
	"text/template"
)

func CreateProject(name string) error {
	err := os.Mkdir("proto", 0755)
	if err != nil {
		return err
	}
	err = os.Mkdir("services", 0755)
	if err != nil {
		return err
	}
	err = os.Mkdir("openapi", 0755)
	if err != nil {
		return err
	}
	return createFiles(name)
}

func createFiles(name string) error {
	templates := map[string]string{}

	templates["main.go"] = executeTemplate(mainGo, name)
	templates["buf.gen.yaml"] = executeTemplate(bufgenYaml, name)
	templates["buf.work.yaml"] = executeTemplate(bufworkYaml, name)
	templates[".golangci.yml"] = executeTemplate(golangciYaml, name)
	templates["proto/buf.yaml"] = executeTemplate(bugYaml, name)
	templates["proto/buf.lock"] = executeTemplate(bufLock, name)
	templates[".gitignore"] = executeTemplate(gitignore, name)
	templates[".env.example"] = executeTemplate(dotEnv, name)

	for filename, contents := range templates {
		err := os.WriteFile(filename, []byte(contents), 0644)
		if err != nil {
			return err
		}
	}

	return nil
}

// executeTemplate executes a template with the provided data.
func executeTemplate(tmpl string, name string) string {
	tmplInstance := template.Must(template.New("template").Parse(tmpl))

	var buf bytes.Buffer
	type Data struct {
		ProjectName string
		Name        string
	}
	data := Data{
		ProjectName: name,
		Name:        strings.ToLower(name),
	}
	if err := tmplInstance.Execute(&buf, data); err != nil {
		slog.Error("Template execution failed", "err", err)
		return ""
	}

	return buf.String()
}

func InstallGoDeps() error {
	deps := []string{
		"google.golang.org/protobuf/cmd/protoc-gen-go@latest",
		"google.golang.org/grpc/cmd/protoc-gen-go-grpc@latest",
		"github.com/grpc-ecosystem/grpc-gateway/v2/protoc-gen-grpc-gateway@latest",
		"github.com/grpc-ecosystem/grpc-gateway/v2/protoc-gen-openapiv2@latest",
	}

	for _, dep := range deps {
		cmdArgs := []string{"install", dep}
		cmd := exec.Command("go", cmdArgs...)
		stdout, err := cmd.Output()

		if err != nil {
			fmt.Println(err.Error())
			return err
		}

		// Print the output
		fmt.Println(string(stdout))
	}

	return nil
}
