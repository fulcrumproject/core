package domain

import (
	"fmt"
	"io"
	"mime"
	"regexp"
	"strings"
	"text/template"

	"github.com/fulcrumproject/core/pkg/schema"
)

const cmdTemplateExtraRef = "configUrl"

var missingKeyRe = regexp.MustCompile(`map has no entry for key "([^"]+)"`)

func (at *AgentType) validateTemplates() error {
	configData := mockDataFromSchema(at.ConfigurationSchema.Properties)
	cmdData := mockDataFromSchema(at.ConfigurationSchema.Properties)
	cmdData[cmdTemplateExtraRef] = ""

	if err := parseAndRender("configTemplate", at.ConfigTemplate, configData); err != nil {
		return err
	}
	if err := parseAndRender("cmdTemplate", at.CmdTemplate, cmdData); err != nil {
		return err
	}

	if (at.ConfigTemplate == "") != (at.CmdTemplate == "") {
		return fmt.Errorf("configTemplate and cmdTemplate must both be set or both be empty")
	}

	if at.ConfigTemplate != "" && at.CmdTemplate != "" {
		cmdDataNoURL := mockDataFromSchema(at.ConfigurationSchema.Properties)
		err := parseAndRender("cmdTemplate", at.CmdTemplate, cmdDataNoURL)
		if err == nil {
			return fmt.Errorf("cmdTemplate must reference {{.configUrl}} when configTemplate is set")
		}
		if !strings.Contains(err.Error(), cmdTemplateExtraRef) {
			return err
		}
	}

	if at.ConfigContentType != "" {
		if _, _, err := mime.ParseMediaType(at.ConfigContentType); err != nil {
			return fmt.Errorf("configContentType %q is not a valid media type: %v", at.ConfigContentType, err)
		}
	}
	return nil
}

func parseAndRender(name, body string, data map[string]any) error {
	if body == "" {
		return nil
	}
	tmpl, err := template.New(name).Option("missingkey=error").Parse(body)
	if err != nil {
		return fmt.Errorf("%s: %w", name, err)
	}
	if err := tmpl.Execute(io.Discard, data); err != nil {
		if m := missingKeyRe.FindStringSubmatch(err.Error()); len(m) == 2 {
			return fmt.Errorf("%s references unknown property %q", name, m[1])
		}
		return fmt.Errorf("%s: %w", name, err)
	}
	return nil
}

func mockDataFromSchema(props map[string]schema.PropertyDefinition) map[string]any {
	data := make(map[string]any, len(props))
	for name, def := range props {
		data[name] = mockForDef(def)
	}
	return data
}

func mockForDef(def schema.PropertyDefinition) any {
	switch def.Type {
	case "integer", "number":
		return 0
	case "boolean":
		return false
	case "object":
		return mockDataFromSchema(def.Properties)
	case "array":
		if def.Items != nil {
			return []any{mockForDef(*def.Items)}
		}
		return []any{}
	default:
		return ""
	}
}
