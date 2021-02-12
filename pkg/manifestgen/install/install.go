/*
Copyright 2020 The Flux authors

Licensed under the Apache License, Version 2.0 (the "License");
you may not use this file except in compliance with the License.
You may obtain a copy of the License at

    http://www.apache.org/licenses/LICENSE-2.0

Unless required by applicable law or agreed to in writing, software
distributed under the License is distributed on an "AS IS" BASIS,
WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
See the License for the specific language governing permissions and
limitations under the License.
*/

package install

import (
	"context"
	"encoding/json"
	"fmt"
	"io/ioutil"
	"net/http"
	"os"
	"path"
	"strings"
	"time"

	securejoin "github.com/cyphar/filepath-securejoin"

	"github.com/fluxcd/flux2/pkg/manifestgen"
)

// Generate returns the install manifests as a multi-doc YAML.
// The manifests are built from a GitHub release or from a
// Kustomize overlay if the supplied Options.BaseURL is a local path.
func Generate(options Options) (*manifestgen.Manifest, error) {
	ctx, cancel := context.WithTimeout(context.Background(), options.Timeout)
	defer cancel()

	tmpDir, err := ioutil.TempDir("", options.Namespace)
	if err != nil {
		return nil, fmt.Errorf("temp dir error: %w", err)
	}
	defer os.RemoveAll(tmpDir)

	output, err := securejoin.SecureJoin(tmpDir, options.ManifestFile)
	if err != nil {
		return nil, err
	}

	if !strings.HasPrefix(options.BaseURL, "http") {
		if err := build(options.BaseURL, output); err != nil {
			return nil, err
		}
	} else {
		if err := fetch(ctx, options.BaseURL, options.Version, tmpDir); err != nil {
			return nil, err
		}

		if err := generate(tmpDir, options); err != nil {
			return nil, err
		}

		if err := build(tmpDir, output); err != nil {
			return nil, err
		}
	}

	content, err := ioutil.ReadFile(output)
	if err != nil {
		return nil, err
	}

	return &manifestgen.Manifest{
		Path:    path.Join(options.TargetPath, options.Namespace, options.ManifestFile),
		Content: string(content),
	}, nil
}

// GetLatestVersion calls the GitHub API and returns the latest released version.
func GetLatestVersion() (string, error) {
	ghURL := "https://api.github.com/repos/fluxcd/flux2/releases/latest"
	c := http.DefaultClient
	c.Timeout = 15 * time.Second

	res, err := c.Get(ghURL)
	if err != nil {
		return "", fmt.Errorf("calling GitHub API failed: %w", err)
	}

	if res.Body != nil {
		defer res.Body.Close()
	}

	type meta struct {
		Tag string `json:"tag_name"`
	}
	var m meta
	if err := json.NewDecoder(res.Body).Decode(&m); err != nil {
		return "", fmt.Errorf("decoding GitHub API response failed: %w", err)
	}

	return m.Tag, err
}
