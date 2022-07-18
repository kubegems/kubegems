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

package workflow

/*

目前需要满足：

- [ ] 任务定义
	- [ ] 任务支持多阶段，多步骤，配置简，单模块化。
	- [ ] 异步任务各个阶段支持依赖关系。支持 串行，并行，分支。
	- [ ] 支持定时任务，周期任务。
- [ ] 任务控制
	- [ ] 支持运行任务终止/中断执行。 如果支持这个特性则需要 node 和server之间长连接以接受控制。
	- [ ] 支持从失败的任务阶段进行重试。
- [ ] 任务执行
	- [ ] 支持异步分布式worker模式。分散任务至多个worker处理。
	- [ ] 支持实时状态更新通知。用于**实时**展示任务状态。
  	- [ ] 支持历史任务查询，支持时间排序。
  	- [ ] 支持任务过期通知。此项目用于在任务过期时接受通知并持久化至数据库，便于后续分析。
*/

// server 作为控制端可以发布任务,多个 server 之间同级。
// server 可以控制任务执行，监控任务状态，汇总任务结果，任务分发
// node 可以接受 server 下发的任务并执行，实时反馈执行结果
// 任务生命周期可增加 hook，支持设置任务过期。

// 工作流程

// - 提交任务(task)，进入submit队列
// - 一个server消费该 task，初始化状态更新至kv
// -  该server拆分任务中的每个task，处理tasks之间依赖，发送至task队列
// - 一个worker消费该task，根据任务执行，在执行的各个阶段更新task状态发布至 callback队列
// - 一个 server 从callback队列接受到数据后，根据依赖决定下一阶段task，否则标记该任务成功/失败
