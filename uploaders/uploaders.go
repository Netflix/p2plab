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

package uploaders

import (
	"path/filepath"

	"github.com/Netflix/p2plab"
	"github.com/Netflix/p2plab/errdefs"
	"github.com/Netflix/p2plab/pkg/httputil"
	"github.com/Netflix/p2plab/uploaders/fileuploader"
	"github.com/Netflix/p2plab/uploaders/s3uploader"
	"github.com/pkg/errors"
	"github.com/rs/zerolog"
)

type UploaderSettings struct {
	Client *httputil.Client
	Logger *zerolog.Logger
	S3     s3uploader.S3UploaderSettings
	File   fileuploader.FileUploaderSettings
}

func GetUploader(root, uploaderType string, settings UploaderSettings) (p2plab.Uploader, error) {
	root = filepath.Join(root, uploaderType)
	switch uploaderType {
	case "file":
		return fileuploader.New(root, settings.Logger, settings.File)
	case "s3":
		return s3uploader.New(settings.Client.HTTPClient, settings.S3)
	default:
		return nil, errors.Wrapf(errdefs.ErrInvalidArgument, "unrecognized uploader type %q", uploaderType)
	}
}
