package config

import (
	"bytes"
	"errors"
	"fmt"
	"os"
	"strconv"
	"strings"

	"gopkg.in/yaml.v3"
)

type Config struct {
	Source Source
}

type Source struct {
	Host     string
	Port     int
	Username string
	Password string
	DBName   string
	TLS      TLS
}

type TLS struct {
	CACert string
	Cert   string
	Key    string
}

func Load(path string) (Config, error) {
	data, err := os.ReadFile(path)
	if err != nil {
		return Config{}, fmt.Errorf("read config file %q: %w", path, err)
	}

	decoder := yaml.NewDecoder(bytes.NewReader(data))
	decoder.KnownFields(true)

	var document yaml.Node
	if err := decoder.Decode(&document); err != nil {
		return Config{}, fmt.Errorf("decode config file %q: %w", path, err)
	}

	cfg, err := parseConfigDocument(&document)
	if err != nil {
		return Config{}, fmt.Errorf("validate config file %q: %w", path, err)
	}

	return cfg, nil
}

func parseConfigDocument(document *yaml.Node) (Config, error) {
	if document.Kind != yaml.DocumentNode || len(document.Content) != 1 {
		return Config{}, errors.New("config must contain exactly one YAML document")
	}

	root := document.Content[0]
	rootFields, err := requiredMapping(root, "config")
	if err != nil {
		return Config{}, err
	}
	if err := rejectUnknownFields(rootFields, "config", "source"); err != nil {
		return Config{}, err
	}

	sourceNode, ok := rootFields["source"]
	if !ok {
		return Config{}, errors.New("config.source is required")
	}

	source, err := parseSource(sourceNode)
	if err != nil {
		return Config{}, err
	}

	return Config{Source: source}, nil
}

func parseSource(node *yaml.Node) (Source, error) {
	fields, err := requiredMapping(node, "source")
	if err != nil {
		return Source{}, err
	}
	if err := rejectUnknownFields(fields, "source", "host", "port", "username", "password", "dbname", "tls"); err != nil {
		return Source{}, err
	}

	host, err := requiredStringField(fields, "source.host", "host")
	if err != nil {
		return Source{}, err
	}

	port, err := requiredPortField(fields, "source.port", "port")
	if err != nil {
		return Source{}, err
	}

	username, err := requiredCredentialField(fields, "source.username", "username")
	if err != nil {
		return Source{}, err
	}

	password, err := requiredCredentialField(fields, "source.password", "password")
	if err != nil {
		return Source{}, err
	}

	dbName, err := requiredStringField(fields, "source.dbname", "dbname")
	if err != nil {
		return Source{}, err
	}

	tls, err := optionalTLSField(fields["tls"])
	if err != nil {
		return Source{}, err
	}

	return Source{
		Host:     host,
		Port:     port,
		Username: username,
		Password: password,
		DBName:   dbName,
		TLS:      tls,
	}, nil
}

func optionalTLSField(node *yaml.Node) (TLS, error) {
	if node == nil {
		return TLS{}, nil
	}

	fields, err := requiredMapping(node, "source.tls")
	if err != nil {
		return TLS{}, err
	}
	if err := rejectUnknownFields(fields, "source.tls", "ca_cert", "cert", "key"); err != nil {
		return TLS{}, err
	}

	caCert, err := optionalPathStringField(fields["ca_cert"], "source.tls.ca_cert")
	if err != nil {
		return TLS{}, err
	}
	cert, err := optionalPathStringField(fields["cert"], "source.tls.cert")
	if err != nil {
		return TLS{}, err
	}
	key, err := optionalPathStringField(fields["key"], "source.tls.key")
	if err != nil {
		return TLS{}, err
	}

	return TLS{
		CACert: caCert,
		Cert:   cert,
		Key:    key,
	}, nil
}

func requiredCredentialField(fields map[string]*yaml.Node, fieldPath, key string) (string, error) {
	node, ok := fields[key]
	if !ok {
		return "", fmt.Errorf("%s is required", fieldPath)
	}
	return resolveCredential(fieldPath, node)
}

func resolveCredential(field string, node *yaml.Node) (string, error) {
	fields, err := requiredMapping(node, field)
	if err != nil {
		return "", err
	}
	if err := rejectUnknownFields(fields, field, "value", "env-ref", "secret-file"); err != nil {
		return "", err
	}

	modeCount := 0
	var resolved string

	if valueNode, ok := fields["value"]; ok {
		modeCount++
		resolved, err = requiredScalarString(valueNode, field+".value")
		if err != nil {
			return "", err
		}
	}
	if envNode, ok := fields["env-ref"]; ok {
		modeCount++
		envName, envErr := requiredScalarString(envNode, field+".env-ref")
		if envErr != nil {
			return "", envErr
		}
		value, ok := os.LookupEnv(envName)
		if !ok {
			return "", fmt.Errorf("%s env-ref %q is not set", field, envName)
		}
		if value == "" {
			return "", fmt.Errorf("%s env-ref %q resolved empty value", field, envName)
		}
		resolved = value
	}
	if secretNode, ok := fields["secret-file"]; ok {
		modeCount++
		secretPath, secretErr := requiredScalarString(secretNode, field+".secret-file")
		if secretErr != nil {
			return "", secretErr
		}
		value, secretErr := readSecretFile(field, secretPath)
		if secretErr != nil {
			return "", secretErr
		}
		resolved = value
	}

	if modeCount != 1 {
		return "", fmt.Errorf("%s must set exactly one of value, env-ref, or secret-file", field)
	}
	if resolved == "" {
		return "", fmt.Errorf("%s resolved empty value", field)
	}

	return resolved, nil
}

func readSecretFile(field, path string) (string, error) {
	data, err := os.ReadFile(path)
	if err != nil {
		return "", fmt.Errorf("%s read secret-file %q: %w", field, path, err)
	}

	value := strings.TrimRight(string(data), "\r\n")
	if value == "" {
		return "", fmt.Errorf("%s secret-file %q resolved empty value", field, path)
	}

	return value, nil
}

func requiredPortField(fields map[string]*yaml.Node, fieldPath, key string) (int, error) {
	node, ok := fields[key]
	if !ok {
		return 0, fmt.Errorf("%s is required", fieldPath)
	}
	if node.Kind != yaml.ScalarNode {
		return 0, fmt.Errorf("%s must be an integer", fieldPath)
	}

	port, err := strconv.Atoi(node.Value)
	if err != nil {
		return 0, fmt.Errorf("%s must be an integer: %w", fieldPath, err)
	}
	if port < 1 || port > 65535 {
		return 0, fmt.Errorf("%s must be between 1 and 65535", fieldPath)
	}

	return port, nil
}

func requiredStringField(fields map[string]*yaml.Node, fieldPath, key string) (string, error) {
	node, ok := fields[key]
	if !ok {
		return "", fmt.Errorf("%s is required", fieldPath)
	}
	return requiredScalarString(node, fieldPath)
}

func optionalPathStringField(node *yaml.Node, fieldPath string) (string, error) {
	if node == nil {
		return "", nil
	}

	value, err := requiredScalarString(node, fieldPath)
	if err != nil {
		return "", err
	}
	if strings.Contains(value, "\n") || strings.Contains(value, "\r") {
		return "", fmt.Errorf("%s must be a file path string, not inline content", fieldPath)
	}

	return value, nil
}

func requiredScalarString(node *yaml.Node, fieldPath string) (string, error) {
	if node.Kind != yaml.ScalarNode {
		return "", fmt.Errorf("%s must be a string", fieldPath)
	}
	if node.Value == "" {
		return "", fmt.Errorf("%s must not be empty", fieldPath)
	}
	return node.Value, nil
}

func requiredMapping(node *yaml.Node, fieldPath string) (map[string]*yaml.Node, error) {
	if node.Kind != yaml.MappingNode {
		return nil, fmt.Errorf("%s must be a mapping object", fieldPath)
	}

	fields := make(map[string]*yaml.Node, len(node.Content)/2)
	for i := 0; i < len(node.Content); i += 2 {
		fields[node.Content[i].Value] = node.Content[i+1]
	}

	return fields, nil
}

func rejectUnknownFields(fields map[string]*yaml.Node, fieldPath string, allowedKeys ...string) error {
	allowed := make(map[string]struct{}, len(allowedKeys))
	for _, key := range allowedKeys {
		allowed[key] = struct{}{}
	}

	for key := range fields {
		if _, ok := allowed[key]; !ok {
			return fmt.Errorf("%s has unknown field %q", fieldPath, key)
		}
	}

	return nil
}
