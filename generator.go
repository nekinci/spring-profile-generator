package main

import (
	"fmt"
	"strings"
)

type Generator struct {
	Trainer     *V1TrainerYaml
	environment string
}

type Url struct {
	Url string
}

func NewUrl(url interface{}) Url {
	switch url.(type) {
	case string:
		return Url{
			Url: url.(string),
		}
	case Environment:
		return Url{
			Url: fmt.Sprintf("%s://%s", url.(Environment).Scheme, url.(Environment).Value),
		}
	case *Environment:
		return Url{
			Url: fmt.Sprintf("%s://%s", url.(*Environment).Scheme, url.(*Environment).Value),
		}

	default:
		panic("unsupported url type")
	}
}

func (u Url) String() string {
	return u.Url
}

func (u Url) WithoutScheme() string {
	if u.Url == "" {
		return ""
	}

	httpContains := strings.Contains(u.Url, "http://")
	if httpContains {
		return strings.Replace(u.Url, "http://", "", 1)
	}

	httpsContains := strings.Contains(u.Url, "https://")
	if httpsContains {
		return strings.Replace(u.Url, "https://", "", 1)
	}

	return u.Url
}

func (u Url) Scheme() string {
	if u.Url == "" {
		return ""
	}

	httpContains := strings.Contains(u.Url, "http://")
	if httpContains {
		return "http"
	}

	httpsContains := strings.Contains(u.Url, "https://")
	if httpsContains {
		return "https"
	}

	// TODO: think about this, maybe we return empty string
	return "http"
}

func (u Url) Hostname() string {
	if u.Url == "" {
		return ""
	}

	withoutScheme := u.WithoutScheme()
	withoutScheme = strings.Split(withoutScheme, "/")[0]
	return withoutScheme
}

func (u Url) Path() string {
	if u.Url == "" {
		return ""
	}

	withoutScheme := u.WithoutScheme()
	index := strings.Index(withoutScheme, "/")
	if index == -1 {
		return ""
	}

	return withoutScheme[index:]
}

func NewGenerator(trainer *V1TrainerYaml, environment string) *Generator {
	checkEnvironment(trainer, environment)
	return &Generator{
		Trainer:     trainer,
		environment: environment,
	}
}

func (g *Generator) Generate(m map[string]interface{}) map[string]interface{} {
	m = g.GenerateForFields(m)
	m = g.GenerateForAbsoluteConfig("", m)
	return m
}

func (g *Generator) GenerateForAbsoluteConfig(key string, m map[string]interface{}) map[string]interface{} {

	for k, v := range m {
		kk := getKey(key, k)
		switch v.(type) {
		case map[string]interface{}:
			m[k] = g.GenerateForAbsoluteConfig(kk, v.(map[string]interface{}))
		case interface{}:
			m[k] = g.decideConfigValue(kk, v)
		default:
			panic("unsupported type")
		}
	}

	return m
}

func getKey(key string, k string) string {
	if key == "" {
		return k
	}
	return key + "." + k
}

func (g *Generator) decideConfigValue(k string, v interface{}) interface{} {
	config := g.getConfigValue(k)
	if config == nil {
		return v
	}
	return *config
}

func (g *Generator) getConfigValue(k interface{}) *interface{} {

	if k == "" {
		return nil
	}

	for _, config := range g.Trainer.Information.AbsoluteConfig {
		if config.Key == k {
			s := config.Environment[g.environment]
			return &s
		}
	}

	return nil
}

func (g *Generator) GenerateForFields(m map[string]interface{}) map[string]interface{} {

	for k, v := range m {
		switch v.(type) {
		case string:
			m[k] = g.generateString(k, v.(string))
		case map[string]interface{}:
			m[k] = g.GenerateForFields(v.(map[string]interface{}))
		case []interface{}:
			m[k] = v
		case int:
			m[k] = v
		case bool:
			m[k] = v
		default:
			panic(fmt.Sprintf("unsupported type %s %T", k, v))
		}
	}

	return m
}

func (g *Generator) generateString(k, v string) string {

	if v == "" {
		return v
	}

	if !strings.Contains(v, "http://") && !strings.Contains(v, "https://") {
		return v
	}

	environmentUrl := g.getEnvironmentUrl(v)
	currentUrl := NewUrl(v)
	return fmt.Sprintf("%s://%s%s", environmentUrl.Scheme(), environmentUrl.Hostname(), currentUrl.Path())
}

func (g *Generator) getEnvironmentUrl(currentUrl string) Url {
	field := g.findField(currentUrl)
	if field == nil {
		return NewUrl(currentUrl)
	}
	environment := field.GetEnvironment(g.environment)
	if environment == nil {
		return NewUrl(currentUrl)
	}

	return NewUrl(environment)
}

func (g *Generator) findField(url string) *Field {

	for _, field := range g.Trainer.Information.Fields {
		for _, environment := range field.Environment {
			if strings.Contains(url, environment.Value) {
				return &field
			}
		}
	}

	return nil
}

func checkEnvironment(trainer *V1TrainerYaml, environment string) {
	for _, field := range trainer.Information.Fields {
		isDefined := false
		for k, _ := range field.Environment {
			if k == environment {
				isDefined = true
			}
		}

		if !isDefined {
			panic("environment not defined")
		}
	}
}
