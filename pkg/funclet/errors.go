/*
 * Copyright (c) 2020 Baidu, Inc.
 *
 * Licensed under the Apache License, Version 2.0 (the "License");
 * you may not use this file except in compliance with the License.
 * You may obtain a copy of the License at
 *
 *     http://www.apache.org/licenses/LICENSE-2.0
 *
 * Unless required by applicable law or agreed to in writing, software
 * distributed under the License is distributed on an "AS IS" BASIS,
 * WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
 * See the License for the specific language governing permissions and
 * limitations under the License.
 */

package funclet

import (
	"fmt"

	"github.com/baidu/openless/pkg/api"
)

type ContainerNotExist struct {
	ID string
}

func (e ContainerNotExist) Error() string {
	return fmt.Sprintf("container %s not found", e.ID)
}

type ContainerNotRunning struct {
	ID string
}

func (e ContainerNotRunning) Error() string {
	return fmt.Sprintf("container %s not running", e.ID)
}

type ContainerIsBusy struct {
	ID           string
	CurrentEvent api.Event
	TriggerEvent api.Event
}

func (e ContainerIsBusy) Error() string {
	return fmt.Sprintf("container %s is busy: current event %s, trigger event %s", e.ID, e.CurrentEvent, e.TriggerEvent)
}
