package job

import (
	"context"
	"sync"
	"github.com/golang/protobuf/proto"
	pb "github.com/gyy0727/mygoim/api/logic"
	"github.com/gyy0727/mygoim/internal/job/conf"
	"github.com/Shopify/sarama"
	log "github.com/golang/glog"
	discovery "github.com/gyy0727/mygoim/pkg/discovery"
)


// *Job is push job.
type Job struct {
	c            *conf.Config
	consumer     sarama.ConsumerGroup  //*消费者
	cometServers map[string]*Comet //*连接comet层的rpc客户端
	rooms        map[string]*Room  //*房间
	roomsMutex   sync.RWMutex      //*房间锁
}

//*新建一个job实例
func New(c *conf.Config) *Job {
	j := &Job{
		c:        c,
		consumer: newKafkaSub(c.Kafka),
		rooms:    make(map[string]*Room),
	}
	j.watchComet()
	return j
}

func (j *Job) Setup(sarama.ConsumerGroupSession) error {
	return nil
}

func (j *Job) Cleanup(sarama.ConsumerGroupSession) error {
	return nil
}

func (j *Job) ConsumeClaim(session sarama.ConsumerGroupSession, claim sarama.ConsumerGroupClaim) error {
	for msg := range claim.Messages() {
		pushMsg := new(pb.PushMsg)
		if err := proto.Unmarshal(msg.Value, pushMsg); err != nil {
			log.Errorf("proto.Unmarshal(%v) error(%v)", msg, err)
			continue
		}
		if err := j.push(context.Background(), pushMsg); err != nil {
			log.Errorf("j.push(%v) error(%v)", pushMsg, err)
		}
		log.Infof("consume: %s/%d/%d\t%s\t%+v", msg.Topic, msg.Partition, msg.Offset, msg.Key, pushMsg)
		session.MarkMessage(msg, "") // 标记消息为已处理
	}
	return nil
}


//*新建一个kafka消费者
func newKafkaSub(c *conf.Kafka) sarama.ConsumerGroup {
	config := sarama.NewConfig()
	config.Consumer.Return.Errors = true
	config.Consumer.Offsets.Initial = sarama.OffsetOldest // 从最早的消息开始消费

	consumer, err := sarama.NewConsumerGroup(c.Brokers, c.Group, config)
	if err != nil {
		panic(err)
	}
	return consumer
}


//*关闭job
func (j *Job) Close() error {
	if j.consumer != nil {
		return j.consumer.Close()
	}
	return nil
}

//*从kafka消费数据
func (j *Job) Consume() {
	for {
		if err := j.consumer.Consume(context.Background(), []string{j.c.Kafka.Topic}, j); err != nil {
			log.Errorf("consumer error(%v)", err)
		}
	}
}


func (j *Job) watchComet(){
	discovery.EtcdResolverInit()
	discovery.EResolver.SetTargetNode("goim/logic")
	node := discovery.EResolver.GetServiceNodes("goim/logic")
	j.cometServers = make(map[string]*Comet) // 初始化 cometServers
	for _,nd := range node{
		comet ,err :=NewComet(nd.Addr, j.c.Comet)
		if err!= nil{
			panic(err)
		}
		j.cometServers[nd.Addr]=comet
	}
}