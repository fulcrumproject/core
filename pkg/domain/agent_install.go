package domain

import (
	"bytes"
)

// RenderCmdTemplate renders the AgentType's CmdTemplate with schemaData plus
// configUrl and authToken. The authToken is the plain value of the bootstrap
// bearer token and is expected to appear in the template's Authorization header.
// Returns "" when CmdTemplate is empty.
func RenderCmdTemplate(at *AgentType, schemaData map[string]any, configURL, authToken string) (string, error) {
	if at.CmdTemplate == "" {
		return "", nil
	}
	data := make(map[string]any, len(schemaData)+2)
	for k, v := range schemaData {
		data[k] = v
	}
	data[cmdTemplateExtraRef] = configURL
	data[cmdTemplateExtraAuthTokenRef] = authToken
	var buf bytes.Buffer
	if err := executeTemplate("cmdTemplate", at.CmdTemplate, data, &buf); err != nil {
		return "", err
	}
	return buf.String(), nil
}

// RenderConfigTemplate renders the AgentType's ConfigTemplate with schemaData.
// Returns "" when ConfigTemplate is empty.
func RenderConfigTemplate(at *AgentType, schemaData map[string]any) (string, error) {
	if at.ConfigTemplate == "" {
		return "", nil
	}
	var buf bytes.Buffer
	if err := executeTemplate("configTemplate", at.ConfigTemplate, schemaData, &buf); err != nil {
		return "", err
	}
	return buf.String(), nil
}
