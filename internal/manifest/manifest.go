package manifest

import (
	"encoding/hex"
	"errors"
	"fmt"
	"net/url"
	"path/filepath"
	"regexp"
	"strings"
)

const (
	APIVersion = "hps.hermes/v1"
	Kind       = "ProvisioningManifest"
)

var namePattern = regexp.MustCompile(`^[a-z0-9](?:[a-z0-9-]{0,61}[a-z0-9])?$`)

// Manifest describes a versioned, non-executable provisioning plan.
type Manifest struct {
	APIVersion string   `json:"apiVersion"`
	Kind       string   `json:"kind"`
	Metadata   Metadata `json:"metadata"`
	Spec       Spec     `json:"spec"`
}

// Metadata identifies a manifest independently of its filename.
type Metadata struct {
	Name        string `json:"name"`
	Version     string `json:"version"`
	Description string `json:"description,omitempty"`
}

// Spec declares the target operating system and immutable assets.
type Spec struct {
	OS     OperatingSystem `json:"os"`
	Assets []Asset         `json:"assets,omitempty"`
}

// OperatingSystem identifies the intended installer family.
type OperatingSystem struct {
	Distribution string `json:"distribution"`
	Release      string `json:"release"`
	Architecture string `json:"architecture"`
	Variant      string `json:"variant,omitempty"`
}

// Asset declares one downloadable or locally staged immutable file.
type Asset struct {
	Name        string `json:"name"`
	Type        string `json:"type"`
	Source      string `json:"source"`
	Destination string `json:"destination"`
	SHA256      string `json:"sha256"`
	Required    bool   `json:"required"`
}

// Validate applies schema and safety checks without performing network or disk writes.
func (m Manifest) Validate() error {
	var problems []error

	if m.APIVersion != APIVersion {
		problems = append(problems, fmt.Errorf("apiVersion must be %q", APIVersion))
	}
	if m.Kind != Kind {
		problems = append(problems, fmt.Errorf("kind must be %q", Kind))
	}
	if !namePattern.MatchString(m.Metadata.Name) {
		problems = append(problems, errors.New("metadata.name must be a lowercase DNS-style name"))
	}
	if strings.TrimSpace(m.Metadata.Version) == "" {
		problems = append(problems, errors.New("metadata.version is required"))
	}
	if strings.TrimSpace(m.Spec.OS.Distribution) == "" {
		problems = append(problems, errors.New("spec.os.distribution is required"))
	}
	if strings.TrimSpace(m.Spec.OS.Release) == "" {
		problems = append(problems, errors.New("spec.os.release is required"))
	}
	if strings.TrimSpace(m.Spec.OS.Architecture) == "" {
		problems = append(problems, errors.New("spec.os.architecture is required"))
	}

	seenNames := make(map[string]struct{}, len(m.Spec.Assets))
	seenDestinations := make(map[string]struct{}, len(m.Spec.Assets))
	for i, asset := range m.Spec.Assets {
		prefix := fmt.Sprintf("spec.assets[%d]", i)
		if !namePattern.MatchString(asset.Name) {
			problems = append(problems, fmt.Errorf("%s.name must be a lowercase DNS-style name", prefix))
		}
		if _, exists := seenNames[asset.Name]; exists {
			problems = append(problems, fmt.Errorf("%s.name %q is duplicated", prefix, asset.Name))
		}
		seenNames[asset.Name] = struct{}{}

		if strings.TrimSpace(asset.Type) == "" {
			problems = append(problems, fmt.Errorf("%s.type is required", prefix))
		}
		if err := validateSource(asset.Source); err != nil {
			problems = append(problems, fmt.Errorf("%s.source: %w", prefix, err))
		}
		if err := validateDestination(asset.Destination); err != nil {
			problems = append(problems, fmt.Errorf("%s.destination: %w", prefix, err))
		} else {
			clean := filepath.Clean(asset.Destination)
			if _, exists := seenDestinations[clean]; exists {
				problems = append(problems, fmt.Errorf("%s.destination %q is duplicated", prefix, clean))
			}
			seenDestinations[clean] = struct{}{}
		}
		if err := validateSHA256(asset.SHA256); err != nil {
			problems = append(problems, fmt.Errorf("%s.sha256: %w", prefix, err))
		}
	}

	return errors.Join(problems...)
}

func validateSource(raw string) error {
	parsed, err := url.Parse(raw)
	if err != nil {
		return err
	}
	if parsed.Scheme != "http" && parsed.Scheme != "https" {
		return errors.New("scheme must be http or https")
	}
	if strings.TrimSpace(parsed.Host) == "" {
		return errors.New("host is required")
	}
	return nil
}

func validateDestination(destination string) error {
	if strings.TrimSpace(destination) == "" {
		return errors.New("path is required")
	}
	if filepath.IsAbs(destination) {
		return errors.New("path must be relative to the runtime repository")
	}
	clean := filepath.Clean(destination)
	if clean == "." || clean == ".." || strings.HasPrefix(clean, ".."+string(filepath.Separator)) {
		return errors.New("path escapes the runtime repository")
	}
	return nil
}

func validateSHA256(value string) error {
	if len(value) != 64 {
		return errors.New("digest must contain exactly 64 hexadecimal characters")
	}
	if _, err := hex.DecodeString(value); err != nil {
		return errors.New("digest must be hexadecimal")
	}
	return nil
}
