/*
 * Copyright 2023 The RuleGo Authors.
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

package kafka

import (
	"github.com/IBM/sarama"
	"github.com/rulego/rulego"
	"github.com/rulego/rulego/api/types"
	"github.com/rulego/rulego/components/base"
	"github.com/rulego/rulego/utils/maps"
	"github.com/rulego/rulego/utils/str"
	"strconv"
)

// 注册节点
func init() {
	_ = rulego.Registry.Register(&ProducerNode{})
}

// NodeConfiguration 节点配置
type NodeConfiguration struct {
	// Brokers kafka服务器地址列表
	Brokers []string
	// Topic 发布主题，可以使用 ${metadata.key} 读取元数据中的变量或者使用 ${msg.key} 读取消息负荷中的变量进行替换
	Topic string
	// Key 分区键，可以使用 ${metadata.key} 读取元数据中的变量或者使用 ${msg.key} 读取消息负荷中的变量进行替换
	Key string
	//Partition 分区编号
	Partition int32
}

type ProducerNode struct {
	Config        NodeConfiguration
	kafkaProducer sarama.SyncProducer
	//topic 模板
	topicTemplate str.Template
	//key 模板
	keyTemplate str.Template
}

// Type 返回组件类型
func (x *ProducerNode) Type() string {
	return "x/kafkaProducer"
}

func (x *ProducerNode) New() types.Node {
	return &ProducerNode{
		Config: NodeConfiguration{
			Brokers:   []string{"127.0.0.1:9092"},
			Partition: 0,
		},
	}
}

// Init 初始化组件
func (x *ProducerNode) Init(ruleConfig types.Config, configuration types.Configuration) error {
	err := maps.Map2Struct(configuration, &x.Config)
	if err == nil {
		config := sarama.NewConfig()
		config.Producer.Return.Successes = true // 同步模式需要设置这个参数为true
		x.kafkaProducer, err = sarama.NewSyncProducer(x.Config.Brokers, config)
		x.topicTemplate = str.NewTemplate(x.Config.Topic)
		x.keyTemplate = str.NewTemplate(x.Config.Key)
	}
	return err
}

// OnMsg 处理消息
func (x *ProducerNode) OnMsg(ctx types.RuleContext, msg types.RuleMsg) {
	topic := x.Config.Topic
	key := x.Config.Key
	if !x.topicTemplate.IsNotVar() || !x.keyTemplate.IsNotVar() {
		evn := base.NodeUtils.GetEvnAndMetadata(ctx, msg)
		topic = str.ExecuteTemplate(topic, evn)
		key = str.ExecuteTemplate(key, evn)
	}

	message := &sarama.ProducerMessage{
		Topic:     topic,
		Partition: x.Config.Partition,
		Key:       sarama.StringEncoder(key),
		Value:     sarama.StringEncoder(msg.Data),
	}
	partition, offset, err := x.kafkaProducer.SendMessage(message)
	if err != nil {
		ctx.TellFailure(msg, err)
	} else {
		msg.Metadata.PutValue("partition", strconv.Itoa(int(partition)))
		msg.Metadata.PutValue("offset", strconv.Itoa(int(offset)))
		ctx.TellSuccess(msg)
	}
}

// Destroy 销毁组件
func (x *ProducerNode) Destroy() {
	if x.kafkaProducer != nil {
		_ = x.kafkaProducer.Close()
	}
}
