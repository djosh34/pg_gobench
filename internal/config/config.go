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
	SSLMode  SSLMode
	Username string
	Password string
	DBName   string
	TLS      TLS
}

type SSLMode string

type TLS struct {
	CACert string
	Cert   string
	Key    string
}

type credentialSourceKind string

const (
	credentialSourceValue      credentialSourceKind = "value"
	credentialSourceEnvRef     credentialSourceKind = "env-ref"
	credentialSourceSecretFile credentialSourceKind = "secret-file"

	SSLModeDisable    SSLMode = "disable"
	SSLModeAllow      SSLMode = "allow"
	SSLModePrefer     SSLMode = "prefer"
	SSLModeRequire    SSLMode = "require"
	SSLModeVerifyCA   SSLMode = "verify-ca"
	SSLModeVerifyFull SSLMode = "verify-full"
)

type credentialSource struct {
	kind    credentialSourceKind
	payload string
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
	if err := rejectUnknownFields(fields, "source", "host", "port", "sslmode", "username", "password", "dbname", "tls"); err != nil {
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

	sslMode, err := requiredSSLModeField(fields, "source.sslmode", "sslmode")
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
	if err := validateSourceTLS(sslMode, tls); err != nil {
		return Source{}, err
	}

	return Source{
		Host:     host,
		Port:     port,
		SSLMode:  sslMode,
		Username: username,
		Password: password,
		DBName:   dbName,
		TLS:      tls,
	}, nil
}

func validateSourceTLS(sslMode SSLMode, tls TLS) error {
	if sslMode == SSLModeDisable {
		if tls.CACert != "" {
			return errors.New("source.tls.ca_cert must not be set when source.sslmode is disable")
		}
		if tls.Cert != "" {
			return errors.New("source.tls.cert must not be set when source.sslmode is disable")
		}
		if tls.Key != "" {
			return errors.New("source.tls.key must not be set when source.sslmode is disable")
		}
	}
	if tls.Cert == "" && tls.Key != "" {
		return errors.New("source.tls.cert is required when source.tls.key is set")
	}
	if tls.Key == "" && tls.Cert != "" {
		return errors.New("source.tls.key is required when source.tls.cert is set")
	}

	return nil
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
	source, err := parseCredentialSource(fieldPath, node)
	if err != nil {
		return "", err
	}
	return resolveCredentialSource(fieldPath, source)
}

func parseCredentialSource(field string, node *yaml.Node) (credentialSource, error) {
	fields, err := requiredMapping(node, field)
	if err != nil {
		return credentialSource{}, err
	}
	if err := rejectUnknownFields(fields, field, "value", "env-ref", "secret-file"); err != nil {
		return credentialSource{}, err
	}

	modeCount := 0
	var source credentialSource

	if valueNode, ok := fields["value"]; ok {
		modeCount++
		value, valueErr := requiredScalarString(valueNode, field+".value")
		if valueErr != nil {
			return credentialSource{}, valueErr
		}
		source = credentialSource{kind: credentialSourceValue, payload: value}
	}
	if envNode, ok := fields["env-ref"]; ok {
		modeCount++
		envName, envErr := requiredScalarString(envNode, field+".env-ref")
		if envErr != nil {
			return credentialSource{}, envErr
		}
		source = credentialSource{kind: credentialSourceEnvRef, payload: envName}
	}
	if secretNode, ok := fields["secret-file"]; ok {
		modeCount++
		secretPath, secretErr := requiredScalarString(secretNode, field+".secret-file")
		if secretErr != nil {
			return credentialSource{}, secretErr
		}
		source = credentialSource{kind: credentialSourceSecretFile, payload: secretPath}
	}

	if modeCount != 1 {
		return credentialSource{}, fmt.Errorf("%s must set exactly one of value, env-ref, or secret-file", field)
	}

	return source, nil
}

func resolveCredentialSource(field string, source credentialSource) (string, error) {
	switch source.kind {
	case credentialSourceValue:
		return source.payload, nil
	case credentialSourceEnvRef:
		value, ok := os.LookupEnv(source.payload)
		if !ok {
			return "", fmt.Errorf("%s env-ref %q is not set", field, source.payload)
		}
		if value == "" {
			return "", fmt.Errorf("%s env-ref %q resolved empty value", field, source.payload)
		}
		return value, nil
	case credentialSourceSecretFile:
		return readSecretFile(field, source.payload)
	default:
		return "", fmt.Errorf("%s has unsupported credential source %q", field, source.kind)
	}
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

func requiredSSLModeField(fields map[string]*yaml.Node, fieldPath, key string) (SSLMode, error) {
	node, ok := fields[key]
	if !ok {
		return "", fmt.Errorf("%s is required", fieldPath)
	}

	value, err := requiredScalarString(node, fieldPath)
	if err != nil {
		return "", err
	}

	switch SSLMode(value) {
	case SSLModeDisable, SSLModeAllow, SSLModePrefer, SSLModeRequire, SSLModeVerifyCA, SSLModeVerifyFull:
		return SSLMode(value), nil
	default:
		return "", fmt.Errorf("%s must be one of disable, allow, prefer, require, verify-ca, or verify-full", fieldPath)
	}
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
