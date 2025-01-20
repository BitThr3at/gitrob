package core

import (
	"io/ioutil"
	"regexp"

	"gopkg.in/yaml.v2"
)

// Pattern represents a single detection pattern
type Pattern struct {
	Name        string `yaml:"name"`
	Type        string `yaml:"type"`
	Pattern     string `yaml:"pattern"`
	Description string `yaml:"description"`
	Comment     string `yaml:"comment"`
}

// Config represents the root configuration
type Config struct {
	Patterns []Pattern `yaml:"patterns"`
}

// LoadConfig loads patterns from the config file
func LoadConfig(configPath string) (*Config, error) {
	data, err := ioutil.ReadFile(configPath)
	if err != nil {
		return nil, err
	}

	var config Config
	if err := yaml.Unmarshal(data, &config); err != nil {
		return nil, err
	}

	return &config, nil
}

// ConvertToSignatures converts config patterns to signatures
func (c *Config) ConvertToSignatures() []Signature {
	var signatures []Signature

	for _, pattern := range c.Patterns {
		switch pattern.Type {
		case "content":
			signatures = append(signatures, ContentSignature{
				part:        PartContent,
				match:       regexp.MustCompile(pattern.Pattern),
				description: pattern.Description,
				comment:     pattern.Comment,
			})
		case "extension":
			signatures = append(signatures, SimpleSignature{
				part:        PartExtension,
				match:       pattern.Pattern,
				description: pattern.Description,
				comment:     pattern.Comment,
			})
		case "filename":
			signatures = append(signatures, SimpleSignature{
				part:        PartFilename,
				match:       pattern.Pattern,
				description: pattern.Description,
				comment:     pattern.Comment,
			})
		case "path":
			signatures = append(signatures, PatternSignature{
				part:        PartPath,
				match:       regexp.MustCompile(pattern.Pattern),
				description: pattern.Description,
				comment:     pattern.Comment,
			})
		}
	}

	return signatures
}
