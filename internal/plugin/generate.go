package plugin

import (
	"fmt"
	"path"
	"path/filepath"
	"sort"

	"github.com/einride/protoc-gen-go-aip-test/internal/util"
	"github.com/einride/protoc-gen-go-aip-test/internal/xrange"
	"google.golang.org/genproto/googleapis/api/annotations"
	"google.golang.org/protobuf/compiler/protogen"
	"google.golang.org/protobuf/reflect/protoreflect"
)

const (
	fileSuffix           = "aiptest.pb.go"
	defaultPackageSuffix = "connect"
)

func Generate(plugin *protogen.Plugin) error {
	plugin.SupportedFeatures |= 1 // proto3 optional
	filesPerPackage, err := collectServices(plugin)
	if err != nil {
		return err
	}
	return generate(plugin, filesPerPackage)
}

type File struct {
	*protogen.File
	services []serviceGenerator
}

// collectServices collects valid services to generate AIP test code for.
func collectServices(
	plugin *protogen.Plugin,
) (map[protoreflect.FullName][]File, error) {
	pkgResources := findResourcesPerPackage(plugin)
	// TODO: Are these limits of "10" here (and below) arbitrary?
	result := make(map[protoreflect.FullName][]File, 10)
	for _, file := range plugin.Files {
		if len(file.Services) == 0 || !file.Generate {
			continue
		}
		f := File{
			File:     file,
			services: make([]serviceGenerator, 0, 10),
		}
		for _, service := range file.Services {
			resources := pkgResources[file.Desc.Package()]
			if len(resources) == 0 {
				// no resources in this package.
				continue
			}
			serviceResources := make([]resource, 0, len(resources))
			for _, r := range resources {
				if util.HasAnyStandardMethodFor(service.Desc, r.descriptor) {
					serviceResources = append(serviceResources, r)
				}
			}
			if len(serviceResources) == 0 {
				continue
			}
			ms := make([]*protogen.Message, 0, len(serviceResources))
			rs := make([]*annotations.ResourceDescriptor, 0, len(serviceResources))
			for _, serviceResource := range serviceResources {
				rs = append(rs, serviceResource.descriptor)
				m, err := protogenMessage(plugin, serviceResource.message.FullName())
				if err != nil {
					return nil, err
				}
				ms = append(ms, m)
			}
			generator := serviceGenerator{
				service:   service,
				resources: rs,
				messages:  ms,
			}
			f.services = append(f.services, generator)
		}
		result[file.Desc.Package()] = append(result[file.Desc.Package()], f)
	}
	return result, nil
}

func generate(plugin *protogen.Plugin, filesPerPackage map[protoreflect.FullName][]File) error {
	for _, files := range filesPerPackage {
		for _, file := range files {
			f := createServiceTestFile(plugin, file)
			f.Skip()
			for _, generator := range file.services {
				if err := generator.Generate(f); err != nil {
					return err
				}
				f.Unskip()
			}
		}
		generateForPackage(plugin, files)
	}
	return nil
}

func generateForPackage(plugin *protogen.Plugin, files []File) {
	directoryName := getDirectoryName(files[0])
	filename := filepath.Join(directoryName, fileSuffix)
	f := plugin.NewGeneratedFile(filename, files[0].GoImportPath)
	writeHeader(files[0].File, f)
	generateServicesConfigProvidersInterface(f, files)
	generateTestAllServices(f, files)
}

func generateServicesConfigProvidersInterface(f *protogen.GeneratedFile, files []File) {
	name := servicesTestSuiteConfigProvidersName()
	f.P("// ", name, " embeds providers for all services.")
	f.P("type ", name, " interface {")
	for _, file := range files {
		for _, service := range file.services {
			f.P(serviceTestConfigProviderName(service.service.Desc))
		}
	}
	f.P("}")
	f.P()
}

func generateTestAllServices(f *protogen.GeneratedFile, files []File) {
	t := f.QualifiedGoIdent(protogen.GoIdent{
		GoName:       "T",
		GoImportPath: "testing",
	})
	name := servicesTestSuiteConfigProvidersName()
	funcName := "TestServices"
	f.P("// ", funcName, " is the main entrypoint for starting the AIP tests for all services.")
	f.P("func ", funcName, "(t *", t, ",s ", name, ") {")
	for _, file := range files {
		for _, service := range file.services {
			name := "test" + string(service.service.Desc.Name())
			f.P(name, "(t, s)")
		}
	}
	f.P("}")
	f.P()
}

// Produces a filename like: "proto/gen/einride/example/freight/v1/examplefreightv1connect/freight_service_aiptest.pb.go"
func getFileName(file File) string {
	// FYI The logic here (and for getDirectoryName) is heavily derived from the connect-go code that generates it's own filenames.
	// Find it here: https://github.com/connectrpc/connect-go/blob/a34cd270cd7fee78df46cbd0afc4ab24d5eac2cf/cmd/protoc-gen-connect-go/main.go#L152

	// Here we get the directory name and then add our new file suffix to the original filename.
	generatedFilenamePrefixToSlash := filepath.ToSlash(file.GeneratedFilenamePrefix)
	filename := path.Join(
		getDirectoryName(file),
		path.Base(generatedFilenamePrefixToSlash+"_"+fileSuffix),
	)
	return filename
}

// Produces a directory name like: "proto/gen/einride/example/freight/v1/examplefreightv1connect"
func getDirectoryName(file File) string {
	// generatedFilenamePrefix	:	"proto/gen/einride/example/freight/v1/freight_service"
	// file.GoPackageName     	:	"examplefreightv1connect"

	// We basically just chop off the the last part of the generatedFilenamePrefix and add the file.GoPackageName.
	generatedFilenamePrefixToSlash := filepath.ToSlash(file.GeneratedFilenamePrefix)
	directoryName := path.Join(
		path.Dir(generatedFilenamePrefixToSlash),
		string(file.GoPackageName),
	)
	return directoryName
}

func addConnectSuffix(file File) {
	// TODO: This should technically be configurable, as this suffix is also configurable for connect-go.
	packageSuffix := defaultPackageSuffix
	file.GoPackageName = file.GoPackageName + protogen.GoPackageName(packageSuffix)
	file.GoImportPath = protogen.GoImportPath(getDirectoryName(file))
}

func createServiceTestFile(plugin *protogen.Plugin, file File) *protogen.GeneratedFile {
	addConnectSuffix(file)
	filename := getFileName(file)
	f := plugin.NewGeneratedFile(filename, file.GoImportPath)
	writeHeader(file.File, f)
	return f
}

func writeHeader(file *protogen.File, f *protogen.GeneratedFile) {
	f.P("// Code generated by protoc-gen-go-aip-test. DO NOT EDIT.")
	f.P()
	f.P("package ", file.GoPackageName)
	f.P()
}

func protogenMessage(plugin *protogen.Plugin, name protoreflect.FullName) (*protogen.Message, error) {
	for _, file := range plugin.Files {
		for _, message := range file.Messages {
			if message.Desc.FullName() == name {
				return message, nil
			}
		}
	}
	return nil, fmt.Errorf("no message named '%s' in plugin", name)
}

type resource struct {
	message    protoreflect.MessageDescriptor
	descriptor *annotations.ResourceDescriptor
}

func findResourcesPerPackage(plugin *protogen.Plugin) map[protoreflect.FullName][]resource {
	result := make(map[protoreflect.FullName][]resource)
	for _, file := range plugin.Files {
		pkg := file.Desc.Package()
		xrange.RangeResourceDescriptors(
			file.Desc,
			func(m protoreflect.MessageDescriptor, r *annotations.ResourceDescriptor) {
				// ignore forwarded resource descriptors
				if m == nil {
					return
				}
				result[pkg] = append(result[pkg], resource{
					message:    m,
					descriptor: r,
				})
			},
		)
	}
	// sort resources to ensure deterministic ordering
	for pkg, resources := range result {
		sort.Slice(resources, func(i, j int) bool {
			return resources[i].descriptor.GetType() < resources[j].descriptor.GetType()
		})
		result[pkg] = resources
	}
	return result
}
