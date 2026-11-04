package main

import (
	"fmt"
	"io"
	"sort"
	"strings"

	"github.com/pkg/errors"
	apiextv1 "k8s.io/apiextensions-apiserver/pkg/apis/apiextensions/v1"
	"k8s.io/apimachinery/pkg/runtime"
	"k8s.io/apimachinery/pkg/runtime/serializer"
	"k8s.io/apimachinery/pkg/util/sets"

	"github.com/openshift/installer/data"
)

const installConfigCRDFileName = "install.openshift.io_installconfigs.yaml"

type schemaField struct {
	Name     string
	Required bool
	Schema   *apiextv1.JSONSchemaProps
}

func loadInstallConfigSchema() (*apiextv1.JSONSchemaProps, error) {
	file, err := data.Assets.Open(installConfigCRDFileName)
	if err != nil {
		return nil, errors.Wrap(err, "failed to load InstallConfig CRD")
	}
	defer file.Close()

	raw, err := io.ReadAll(file)
	if err != nil {
		return nil, errors.Wrap(err, "failed to read InstallConfig CRD")
	}

	scheme := runtime.NewScheme()
	codecs := serializer.NewCodecFactory(scheme)
	apiextv1.AddToScheme(scheme)
	obj, err := runtime.Decode(codecs.UniversalDecoder(apiextv1.SchemeGroupVersion), raw)
	if err != nil {
		return nil, errors.Wrap(err, "failed to decode InstallConfig CRD")
	}
	crd, ok := obj.(*apiextv1.CustomResourceDefinition)
	if !ok {
		return nil, errors.Errorf("invalid object, should be *apiextv1.CustomResourceDefinition but found %T", obj)
	}
	if len(crd.Spec.Versions) != 1 {
		return nil, errors.New("missing versions in CRD")
	}
	return crd.Spec.Versions[0].Schema.OpenAPIV3Schema, nil
}

func lookupSchema(schema *apiextv1.JSONSchemaProps, path []string) (*apiextv1.JSONSchemaProps, error) {
	if len(path) == 0 {
		return schema, nil
	}

	properties := map[string]apiextv1.JSONSchemaProps{}
	if schema.Items != nil && schema.Items.Schema != nil {
		properties = schema.Items.Schema.Properties
	}
	if len(schema.Properties) > 0 {
		properties = schema.Properties
	}

	property, ok := properties[path[0]]
	if !ok {
		return nil, errors.Errorf("invalid field %s, no such property found", path[0])
	}
	return lookupSchema(&property, path[1:])
}

func schemaFields(schema *apiextv1.JSONSchemaProps) []schemaField {
	properties := map[string]apiextv1.JSONSchemaProps{}
	required := sets.NewString(schema.Required...)
	if schema.Items != nil && schema.Items.Schema != nil && len(schema.Items.Schema.Properties) > 0 {
		properties = schema.Items.Schema.Properties
		required.Insert(schema.Items.Schema.Required...)
	}
	if len(schema.Properties) > 0 {
		properties = schema.Properties
	}
	if len(properties) == 0 {
		return nil
	}

	keys := make([]string, 0, len(properties))
	for name := range properties {
		keys = append(keys, name)
	}
	sort.Strings(keys)

	fields := make([]schemaField, 0, len(keys))
	for _, name := range keys {
		fieldSchema := properties[name]
		fields = append(fields, schemaField{
			Name:     name,
			Required: required.Has(name),
			Schema:   &fieldSchema,
		})
	}
	return fields
}

func renderExplainPath(path string, w io.Writer) error {
	model := strings.TrimSuffix(path, ".")
	parts := strings.Split(model, ".")
	if len(parts) == 0 || parts[0] != "installconfig" {
		return errors.New("only installconfig resource is supported")
	}

	schema, err := loadInstallConfigSchema()
	if err != nil {
		return err
	}
	fieldSchema, err := lookupSchema(schema, parts[1:])
	if err != nil {
		return errors.Wrapf(err, "failed to load schema for the field %s", strings.Join(parts[1:], "."))
	}

	p := explainPrinter{Writer: w}
	p.printKindAndVersion()
	p.printResource(fieldSchema)
	p.printFields(fieldSchema)
	return nil
}

type explainPrinter struct {
	Writer io.Writer
}

func (p explainPrinter) printKindAndVersion() {
	io.WriteString(p.Writer, "KIND:     InstallConfig\n")
	io.WriteString(p.Writer, "VERSION:  v1\n\n")
}

func (p explainPrinter) printResource(schema *apiextv1.JSONSchemaProps) {
	resource := schema.Type
	if schema.Items != nil && schema.Items.Schema != nil {
		resource = fmt.Sprintf("[]%s", schema.Items.Schema.Type)
	}
	io.WriteString(p.Writer, fmt.Sprintf("RESOURCE: <%s>\n", resource))
	p.printLabels(2, schema)

	desc := schema.Description
	if len(desc) == 0 {
		desc = "<empty>"
	}
	explainWrite(2, p.Writer, desc)
	io.WriteString(p.Writer, "\n")
}

func (p explainPrinter) printFields(schema *apiextv1.JSONSchemaProps) {
	fields := schemaFields(schema)
	if len(fields) == 0 {
		return
	}

	io.WriteString(p.Writer, "FIELDS:\n")
	for _, field := range fields {
		p.printField(field)
	}
}

func (p explainPrinter) printField(field schemaField) {
	fieldType := field.Schema.Type
	if field.Schema.Items != nil && field.Schema.Items.Schema != nil {
		fieldType = fmt.Sprintf("[]%s", field.Schema.Items.Schema.Type)
	}
	title := fmt.Sprintf("%s <%s>", field.Name, fieldType)
	if field.Required {
		title = fmt.Sprintf("%s -required-", title)
	}
	explainWrite(4, p.Writer, title)
	p.printLabels(6, field.Schema)

	desc := field.Schema.Description
	if desc == "" {
		desc = "<empty>"
	}
	explainWrite(6, p.Writer, desc)
	if field.Schema.Items != nil && field.Schema.Items.Schema != nil && len(field.Schema.Items.Schema.Description) > 0 {
		explainWrite(6, p.Writer, field.Schema.Items.Schema.Description)
	}
	io.WriteString(p.Writer, "\n")
}

func (p explainPrinter) printLabels(indent int, schema *apiextv1.JSONSchemaProps) {
	if schema.Default != nil {
		explainWrite(indent, p.Writer, fmt.Sprintf("Default: %s", string(schema.Default.Raw)))
	}
	if len(schema.Format) > 0 {
		explainWrite(indent, p.Writer, fmt.Sprintf("Format: %s", schema.Format))
	}
	if len(schema.Enum) > 0 {
		values := make([]string, len(schema.Enum))
		for i, value := range schema.Enum {
			values[i] = string(value.Raw)
		}
		explainWrite(indent, p.Writer, fmt.Sprintf("Valid Values: %s", strings.Join(values, ",")))
	}
}

func explainWrite(indent int, w io.Writer, text string) {
	if strings.TrimSpace(text) == "" {
		io.WriteString(w, "\n")
	}
	io.WriteString(w, strings.Repeat(" ", indent)+text+"\n")
}
