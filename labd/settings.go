// Copyright 2019 Netflix, Inc.
//
// Licensed under the Apache License, Version 2.0 (the "License");
// you may not use this file except in compliance with the License.
// You may obtain a copy of the License at
//
//     http://www.apache.org/licenses/LICENSE-2.0
//
// Unless required by applicable law or agreed to in writing, software
// distributed under the License is distributed on an "AS IS" BASIS,
// WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
// See the License for the specific language governing permissions and
// limitations under the License.

package labd

import (
	"github.com/Netflix/p2plab/providers"
	"github.com/Netflix/p2plab/uploaders"
)

type LabdOption func(*LabdSettings) error

type LabdSettings struct {
	Libp2pPort    int
	Provider         string
	ProviderSettings providers.ProviderSettings
	Uploader         string
	UploaderSettings uploaders.UploaderSettings
}

func WithLibp2pPort(port int) LabdOption {
	return func(s *LabdSettings) error {
		s.Libp2pPort = port
		return nil
	}
}

func WithProvider(provider string) LabdOption {
	return func(s *LabdSettings) error {
		s.Provider = provider
		return nil
	}
}

func WithProviderSettings(settings providers.ProviderSettings) LabdOption {
	return func(s *LabdSettings) error {
		s.ProviderSettings = settings
		return nil
	}
}

func WithUploader(uploader string) LabdOption {
	return func(s *LabdSettings) error {
		s.Uploader = uploader
		return nil
	}
}

func WithUploaderSettings(settings uploaders.UploaderSettings) LabdOption {
	return func(s *LabdSettings) error {
		s.UploaderSettings = settings
		return nil
	}
}
