package wrap

import (
	"bytes"
	"context"
	"errors"
	"log/slog"
	"os"
	"path"
	"strings"
	"text/template"

	"github.com/emicklei/proto"
)

const (
	filePerm                = 0644
	serverGetpFileSuffix    = "_grpc.go"
	serverFileSuffix        = "_server.go"
	serverWrapperFileSuffix = "_gofr.go"
	clientFileSuffix        = "_client.go"
	clientHealthFile        = "health_client.go"
	serverHealthFile        = "health_gofr.go"
	serverRequestFile       = "request_gofr.go"
)

var (
	ErrNoProtoFile        = errors.New("proto file path is required")
	ErrOpeningProtoFile   = errors.New("error opening the proto file")
	ErrFailedToParseProto = errors.New("failed to parse proto file")
	ErrGeneratingWrapper  = errors.New("error while generating the code using proto file")
	ErrWritingFile        = errors.New("error writing the generated code to the file")
)

// ServiceMethod represents a method in a proto service.
type ServiceMethod struct {
	Name            string
	Request         string
	Response        string
	StreamsRequest  bool
	StreamsResponse bool
}

// ProtoService represents a service in a proto file.
type ProtoService struct {
	Name    string
	Methods []ServiceMethod
}

// WrapperData is the template data structure.
type WrapperData struct {
	Package      string
	Service      string
	ServiceUpper string
	ServiceLower string
	Methods      []ServiceMethod
	Requests     []string
	Source       string
}

type FileType struct {
	FileSuffix    string
	CodeGenerator func(context.Context, *WrapperData) string
}

// BuildGRPCGoFrClient generates gRPC client wrapper code based on a proto definition.
func BuildGRPCGoFrClient(protoPath string) (any, error) {
	gRPCClient := []FileType{
		{FileSuffix: clientFileSuffix, CodeGenerator: generateGoFrClient},
		{FileSuffix: clientHealthFile, CodeGenerator: generateGoFrClientHealth},
	}

	return generateWrapper(context.Background(), protoPath, gRPCClient...)
}

// BuildGRPCGoFrServer generates gRPC client and server code based on a proto definition.
func BuildGRPCGoFrServer(protoPath string) (any, error) {
	gRPCServer := []FileType{
		//{FileSuffix: serverWrapperFileSuffix, CodeGenerator: generateGoFrServerWrapper},
		//{FileSuffix: serverHealthFile, CodeGenerator: generateGoFrServerHealthWrapper},
		//{FileSuffix: serverRequestFile, CodeGenerator: generateGoFrRequestWrapper},
		//{FileSuffix: serverFileSuffix, CodeGenerator: generateGoFrServer},
		{FileSuffix: serverGetpFileSuffix, CodeGenerator: generateGRPCTemplate},
	}

	return generateWrapper(context.Background(), protoPath, gRPCServer...)
}

// BuildGRPCServer generates gRPC client and server code based on a proto definition.
func BuildGRPCServer(protoPath string) (any, error) {
	gRPCServer := []FileType{
		{FileSuffix: serverGetpFileSuffix, CodeGenerator: generateGRPCTemplate},
	}

	return generateWrapper(context.Background(), protoPath, gRPCServer...)
}

// generateWrapper executes the function for specified FileType to create GoFr integrated
// gRPC server/client files with the required services in proto file and
// specified suffix for every service specified in the proto file.
func generateWrapper(ctx context.Context, protoPath string, options ...FileType) (any, error) {
	if protoPath == "" {
		slog.Error("No proto file", "err", ErrNoProtoFile)
		return nil, ErrNoProtoFile
	}

	definition, err := parseProtoFile(ctx, protoPath)
	if err != nil {
		slog.Error("Failed to parse proto file", "err", err)
		return nil, err
	}

	projectPath, packageName := getPackageAndProject(ctx, definition, protoPath)
	services := getServices(ctx, definition)
	requests := getRequests(ctx, services)

	for _, service := range services {
		wrapperData := WrapperData{
			Package:      packageName,
			Service:      service.Name,
			ServiceUpper: strings.ToUpper(service.Name),
			ServiceLower: strings.ToLower(service.Name),
			Methods:      service.Methods,
			Requests:     uniqueRequestTypes(ctx, service.Methods),
			Source:       path.Base(protoPath),
		}

		if err := generateFiles(ctx, projectPath, service.Name, &wrapperData, requests, options...); err != nil {
			return nil, err
		}
	}

	slog.Info("Successfully generated all files for GoFr integrated gRPC servers/clients")

	return "Successfully generated all files for GoFr integrated gRPC servers/clients", nil
}

// parseProtoFile opens and parses the proto file.
func parseProtoFile(_ context.Context, protoPath string) (*proto.Proto, error) {
	file, err := os.Open(protoPath)
	if err != nil {
		slog.Error("Failed to open proto file", "err", err)
		return nil, ErrOpeningProtoFile
	}
	defer file.Close()

	parser := proto.NewParser(file)

	definition, err := parser.Parse()
	if err != nil {
		slog.Error("Failed to parse proto file", "err", err)
		return nil, ErrFailedToParseProto
	}

	return definition, nil
}

// generateFiles generates files for a given service.
func generateFiles(ctx context.Context, projectPath, serviceName string, wrapperData *WrapperData,
	requests []string, options ...FileType) error {
	for _, option := range options {
		if option.FileSuffix == serverRequestFile {
			wrapperData.Requests = requests
		}

		generatedCode := option.CodeGenerator(ctx, wrapperData)
		if generatedCode == "" {
			slog.Error("Failed to generate code for service %s with file suffix", "service", serviceName, "suffix", option.FileSuffix)
			return ErrGeneratingWrapper
		}

		outputFilePath := getOutputFilePath(projectPath, serviceName, option.FileSuffix)
		if err := os.WriteFile(outputFilePath, []byte(generatedCode), filePerm); err != nil {
			slog.Error("Failed to write file", "path", outputFilePath, "err", err)
			return ErrWritingFile
		}

		slog.Info("Generated file for service %s at", "service", serviceName, "path", outputFilePath)
	}

	return nil
}

// getOutputFilePath generates the output file path based on the file suffix.
func getOutputFilePath(projectPath, serviceName, fileSuffix string) string {
	switch fileSuffix {
	case clientHealthFile:
		return path.Join(projectPath, clientHealthFile)
	case serverHealthFile:
		return path.Join(projectPath, serverHealthFile)
	case serverRequestFile:
		return path.Join(projectPath, serverRequestFile)
	default:
		return path.Join(projectPath, strings.ToLower(serviceName)+fileSuffix)
	}
}

// getRequests extracts all unique request types from the services.
func getRequests(_ context.Context, services []ProtoService) []string {
	requests := make(map[string]bool)

	for _, service := range services {
		for _, method := range service.Methods {
			requests[method.Request] = true
		}
	}

	slog.Debug("Extracted unique request types", "requests", requests)

	return mapKeysToSlice(requests)
}

// uniqueRequestTypes extracts unique request types from methods.
func uniqueRequestTypes(_ context.Context, methods []ServiceMethod) []string {
	requests := make(map[string]bool)

	for _, method := range methods {
		requests[method.Request] = true // Include all request types
	}

	slog.Debug("Extracted unique request types for methods", "requests", requests)

	return mapKeysToSlice(requests)
}

// mapKeysToSlice converts a map's keys to a slice.
func mapKeysToSlice(m map[string]bool) []string {
	keys := make([]string, 0, len(m))
	for key := range m {
		keys = append(keys, key)
	}

	return keys
}

// executeTemplate executes a template with the provided data.
func executeTemplate(_ context.Context, data *WrapperData, tmpl string) string {
	funcMap := template.FuncMap{
		"lowerFirst": func(s string) string {
			if s == "" {
				return ""
			}
			return strings.ToLower(s[:1]) + s[1:]
		},
	}

	tmplInstance := template.Must(template.New("template").Funcs(funcMap).Parse(tmpl))

	var buf bytes.Buffer

	if err := tmplInstance.Execute(&buf, data); err != nil {
		slog.Error("Template execution failed", "err", err)
		return ""
	}

	return buf.String()
}

// Template generators.
func generateGoFrServerWrapper(ctx context.Context, data *WrapperData) string {
	return executeTemplate(ctx, data, wrapperTemplate)
}

func generateGoFrRequestWrapper(ctx context.Context, data *WrapperData) string {
	return executeTemplate(ctx, data, messageTemplate)
}

func generateGoFrServerHealthWrapper(ctx context.Context, data *WrapperData) string {
	return executeTemplate(ctx, data, healthServerTemplate)
}

func generateGoFrClientHealth(ctx context.Context, data *WrapperData) string {
	return executeTemplate(ctx, data, clientHealthTemplate)
}
func generateGRPCTemplate(ctx context.Context, data *WrapperData) string {
	return executeTemplate(ctx, data, clientGRPCTemplate)
}

func generateGoFrServer(ctx context.Context, data *WrapperData) string {
	return executeTemplate(ctx, data, serverTemplate)
}

func generateGoFrClient(ctx context.Context, data *WrapperData) string {
	return executeTemplate(ctx, data, clientTemplate)
}

// getPackageAndProject extracts the package name and project path from the proto definition.
func getPackageAndProject(_ context.Context, definition *proto.Proto, protoPath string) (projectPath, packageName string) {
	proto.Walk(definition,
		proto.WithOption(func(opt *proto.Option) {
			if opt.Name == "go_package" {
				packageName = path.Base(opt.Constant.Source)
			}
		}),
	)

	projectPath = path.Dir(protoPath)
	slog.Debug("Extracted package name: %s, project path", "packageName", packageName, "projectPath", projectPath)
	packageNames := strings.Split(packageName, ";")
	if len(packageNames) >= 1 {
		packageName = packageNames[len(packageNames)-1]
	}
	return projectPath, packageName
}

// getServices extracts services from the proto definition.
func getServices(_ context.Context, definition *proto.Proto) []ProtoService {
	var services []ProtoService

	proto.Walk(definition,
		proto.WithService(func(s *proto.Service) {
			service := ProtoService{Name: s.Name}

			for _, element := range s.Elements {
				if rpc, ok := element.(*proto.RPC); ok {
					service.Methods = append(service.Methods, ServiceMethod{
						Name:            rpc.Name,
						Request:         rpc.RequestType,
						Response:        rpc.ReturnsType,
						StreamsRequest:  rpc.StreamsRequest,
						StreamsResponse: rpc.StreamsReturns,
					})
				}
			}

			services = append(services, service)
		}),
	)

	slog.Debug("Extracted services", "services", services)

	return services
}
