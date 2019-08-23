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

package logutil

import (
	"bufio"
	"bytes"
	"context"
	"encoding/json"
	"io"

	"github.com/pkg/errors"
	"github.com/rs/zerolog"
)

func WriteRemoteLogs(ctx context.Context, remote io.Reader, writer io.Writer) error {
	scanner := bufio.NewScanner(remote)
	for scanner.Scan() {
		decoder := json.NewDecoder(bytes.NewReader([]byte(scanner.Text())))
		decoder.UseNumber()

		var evt map[string]interface{}
		err := decoder.Decode(&evt)
		if err != nil {
			zerolog.Ctx(ctx).Debug().Msg(scanner.Text())
			for scanner.Scan() {
				zerolog.Ctx(ctx).Debug().Msg(scanner.Text())
			}
			return errors.New("unexpected non-json response")
		}

		levelRaw, ok := evt[zerolog.LevelFieldName]
		if ok {
			levelStr, ok := levelRaw.(string)
			if ok {
				level, _ := zerolog.ParseLevel(levelStr)
				event := zerolog.Ctx(ctx).WithLevel(level)
				if event == nil {
					// If event is nil, then the log event was filtered out by the current
					// logger.
					continue
				}
			}
		}

		_, err = writer.Write(append(scanner.Bytes(), byte('\n')))
		if err != nil {
			return err
		}
	}

	return scanner.Err()
}
