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

package client

/*

此包用于将 agent 中的 client.Client 转换为rest的api。
在service端，实现了另一个 client.Client 接口，其后端为agent的该rest api。

目的在于在service实现一个 "原生” 的资源操作client 以操作agent所在集群的资源。

这个 rest api 仅对内提供服务。
*/
