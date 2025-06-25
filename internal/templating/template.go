package templating

import (
	"bytes"
	"text/template"
)

type Template struct {
	Name               string   `yaml:"name"`
	Description        string   `yaml:"description"`
	SystemMessage      string   `yaml:"system_message,omitempty"`
	UserPromptTemplate string   `yaml:"user_prompt_template"`
	Model              string   `yaml:"model,omitempty"`
	Temperature        *float64 `yaml:"temperature,omitempty"`
}

type PromptShape struct {
	UserPrompt string
}

func (t *Template) ProcessUserPromptTemplate(rawPrompt string) (string, error) {
	templ, err := template.New("user_prompt").Parse(rawPrompt)
	if err != nil {
		return "", err
	}

	var buf bytes.Buffer
	err = templ.Execute(&buf, t)
	if err != nil {
		return "", err
	}

	return buf.String(), nil
}
