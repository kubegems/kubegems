// Copyright 2022 The kubegems.io Authors
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

package prometheus

import (
	gemlabels "kubegems.io/kubegems/pkg/apis/gems"
)

const (
	// 全局告警命名空间，非此命名空间强制加上namespace筛选
	GlobalAlertNamespace = gemlabels.NamespaceMonitor
	// namespace
	PromqlNamespaceKey = "namespace"
)
