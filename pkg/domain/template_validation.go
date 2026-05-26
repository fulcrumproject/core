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

const (
	cmdTemplateExtraRef          = "configUrl"
	cmdTemplateExtraAuthTokenRef = "authToken"
)

var missingKeyRe = regexp.MustCompile(`map has no entry for key "([^"]+)"`)

type TemplateValidation struct {
	ConfigurationSchema schema.Schema `json:"configurationSchema" gorm:"type:jsonb;not null"`
	ConfigTemplate      string        `json:"configTemplate" gorm:"type:text"`
	CmdTemplate         string        `json:"cmdTemplate" gorm:"type:text"`
	ConfigContentType   string        `json:"configContentType" gorm:"type:text;not null;default:'text/plain'"`
}

func (tv *TemplateValidation) validateTemplates() error {
	configData := mockDataFromSchema(tv.ConfigurationSchema.Properties)
	cmdData := mockDataFromSchema(tv.ConfigurationSchema.Properties)
	cmdData[cmdTemplateExtraRef] = ""
	cmdData[cmdTemplateExtraAuthTokenRef] = ""

	if err := executeTemplate("configTemplate", tv.ConfigTemplate, configData, io.Discard); err != nil {
		return err
	}
	if err := executeTemplate("cmdTemplate", tv.CmdTemplate, cmdData, io.Discard); err != nil {
		return err
	}

	if (tv.ConfigTemplate == "") != (tv.CmdTemplate == "") {
		return fmt.Errorf("configTemplate and cmdTemplate must both be set or both be empty")
	}

	if tv.ConfigTemplate != "" && tv.CmdTemplate != "" {
		for _, required := range []string{cmdTemplateExtraRef, cmdTemplateExtraAuthTokenRef} {
			data := mockDataFromSchema(tv.ConfigurationSchema.Properties)
			// Populate every required key except the one under test so a missing
			// reference in cmdTemplate is the only way Execute can fail.
			for _, k := range []string{cmdTemplateExtraRef, cmdTemplateExtraAuthTokenRef} {
				if k != required {
					data[k] = ""
				}
			}
			err := executeTemplate("cmdTemplate", tv.CmdTemplate, data, io.Discard)
			if err == nil {
				return fmt.Errorf("cmdTemplate must reference {{.%s}} when configTemplate is set", required)
			}
			if !strings.Contains(err.Error(), required) {
				return err
			}
		}
	}

	if tv.ConfigContentType != "" {
		if _, _, err := mime.ParseMediaType(tv.ConfigContentType); err != nil {
			return fmt.Errorf("configContentType %q is not a valid media type: %v", tv.ConfigContentType, err)
		}
	}
	return nil
}

// executeTemplate parses body and executes it into out with missingkey=error.
// Empty body is a no-op. Missing-key execution errors are rewritten to the
// friendlier "%s references unknown property %q" form.
func executeTemplate(name, body string, data map[string]any, out io.Writer) error {
	if body == "" {
		return nil
	}
	tmpl, err := template.New(name).Option("missingkey=error").Parse(body)
	if err != nil {
		return fmt.Errorf("%s: %w", name, err)
	}
	if err := tmpl.Execute(out, data); err != nil {
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
