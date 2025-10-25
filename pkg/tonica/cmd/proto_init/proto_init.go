package proto_init

import (
	"bytes"
	"fmt"
	"log/slog"
	"os"
	"os/exec"
	"strings"
	"text/template"

	"github.com/tonica-go/tonica/pkg/tonica/cmd/wrap"
)

func CreateProto(name string) error {
	protoFolder := fmt.Sprintf("proto/%s/v1", name)
	err := os.Mkdir(fmt.Sprintf("proto/%s", name), 0755)
	if err != nil {
		return err
	}

	err = os.Mkdir(protoFolder, 0755)
	if err != nil {
		return err
	}

	err = os.WriteFile(fmt.Sprintf("%s/service.proto", protoFolder), []byte(executeTemplate(protoTpl, name)), 0644)
	if err != nil {
		return err
	}

	genProto()
	serviceFolder := fmt.Sprintf("services/%s", name)
	err = os.Mkdir(serviceFolder, 0755)
	if err != nil {
		return err
	}

	err = os.WriteFile(fmt.Sprintf("%s/%s.go", serviceFolder, name), []byte(executeTemplate(serviceTpl, name)), 0644)
	if err != nil {
		return err
	}

	_, err = wrap.BuildGRPCServer(fmt.Sprintf("%s/service.proto", protoFolder))
	if err != nil {
		return err
	}

	return nil
}

func genProto() {
	app := "buf"

	arg0 := "generate"

	cmd := exec.Command(app, arg0)
	stdout, err := cmd.Output()

	if err != nil {
		fmt.Println(err.Error())
		return
	}

	// Print the output
	fmt.Println(string(stdout))
}

// executeTemplate executes a template with the provided data.
func executeTemplate(tmpl string, name string) string {
	tmplInstance := template.Must(template.New("template").Parse(tmpl))

	var buf bytes.Buffer
	type Data struct {
		ProjectName    string
		Name           string
		NameFirstUpper string
	}

	data := Data{
		ProjectName:    name,
		Name:           strings.ToLower(name),
		NameFirstUpper: strings.Title(name),
	}
	if err := tmplInstance.Execute(&buf, data); err != nil {
		slog.Error("Template execution failed", "err", err)
		return ""
	}

	return buf.String()
}
