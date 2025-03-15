// //*实现了一个基于权重的负载均衡器（LoadBalancer），用于管理节点并根据权重分配请求

package logic

// import (
// 	"fmt"
// 	"math"
// 	"sort"
// 	"strconv"
// 	"strings"
// 	"sync"

// 	"github.com/bilibili/discovery/naming"
// 	log "github.com/golang/glog"
// 	"github.com/gyy0727/mygoim/internal/logic/model"
// )

// const (
// 	_minWeight = 1       //*最小权重值
// 	_maxWeight = 1 << 20 //*最大权重值
// 	_maxNodes  = 5       //*每次返回的最大节点数
// )

// // *定义带权节点结构体
// type weightedNode struct {
// 	region        string   //*区域
// 	hostname      string   //*主机名
// 	addrs         []string //*地址列表
// 	fixedWeight   int64    //*固定权重
// 	currentWeight int64    //*当前权重
// 	currentConns  int64    //*当前连接数
// 	updated       int64    //*更新时间戳
// }

// // *字符串输出当前带权节点
// func (w *weightedNode) String() string {
// 	return fmt.Sprintf("region:%s fixedWeight:%d, currentWeight:%d, currentConns:%d", w.region, w.fixedWeight, w.currentWeight, w.currentConns)
// }

// // *当节点被选中时，增加当前连接数
// func (w *weightedNode) chosen() {
// 	w.currentConns++
// }

// // *重置节点的当前权重
// func (w *weightedNode) reset() {
// 	w.currentWeight = 0
// }

// // *calculateWeight 方法根据节点的固定权重、总权重、总连接数以及增益权重来计算节点的当前权重
// // *它考虑了权重比例与连接比例之间的差异，并根据差异调整当前权重，同时确保当前权重在 _minWeight 和 _maxWeight 之间
// // *如果没有连接，则重置节点状态。
// func (w *weightedNode) calculateWeight(totalWeight, totalConns int64, gainWeight float64) {
// 	//*计算 fixedWeight，它是 w.fixedWeight（节点的固定权重）乘以 gainWeight（增益权重）的结果
// 	fixedWeight := float64(w.fixedWeight) * gainWeight
// 	//*更新 totalWeight，将 fixedWeight 转换为 int64 类型后减去 w.fixedWeight
// 	//*并将结果加到 totalWeight 上。这一步的目的是调整总权重，使其反映新的 fixedWeight
// 	totalWeight += int64(fixedWeight) - w.fixedWeight
// 	//*检查 totalConns 是否大于 0。如果大于 0，表示当前有连接存在，继续执行后续逻辑
// 	//*否则，调用 w.reset() 方法重置节点状态
// 	if totalConns > 0 {
// 		weightRatio := fixedWeight / float64(totalWeight)
// 		var connRatio float64
// 		if totalConns != 0 {
// 			connRatio = float64(w.currentConns) / float64(totalConns) * 0.5
// 		}
// 		diff := weightRatio - connRatio
// 		multiple := diff * float64(totalConns)
// 		floor := math.Floor(multiple)
// 		if floor-multiple >= -0.5 {
// 			w.currentWeight = int64(fixedWeight + floor)
// 		} else {
// 			w.currentWeight = int64(fixedWeight + math.Ceil(multiple))
// 		}
// 		if diff < 0 {

// 			if _minWeight > w.currentWeight {
// 				w.currentWeight = _minWeight
// 			}
// 		} else {

// 			if _maxWeight < w.currentWeight {
// 				w.currentWeight = _maxWeight
// 			}
// 		}
// 	} else {
// 		w.reset()
// 	}
// }

// type LoadBalancer struct {
// 	totalConns  int64                    //*记录所有节点的总连接数
// 	totalWeight int64                    //*记录所有节点的总权重
// 	nodes       map[string]*weightedNode //*存储节点信息的映射表，键为节点的主机名，值为 weightedNode
// 	nodesMutex  sync.Mutex
// }

// // *创建一个新的 LoadBalancer 实例，并初始化 nodes 映射表
// func NewLoadBalancer() *LoadBalancer {
// 	lb := &LoadBalancer{
// 		nodes: make(map[string]*weightedNode),
// 	}
// 	return lb
// }

// // *返回当前负载均衡器中节点的数量
// func (lb *LoadBalancer) Size() int {
// 	return len(lb.nodes)
// }

// // *遍历所有节点，根据区域和区域权重计算每个节点的当前权重。
// // *将节点按当前权重从大到小排序。
// // *如果节点列表不为空，选择权重最高的节点，并增加其连接数和总连接数。
// func (lb *LoadBalancer) weightedNodes(region string, regionWeight float64) (nodes []*weightedNode) {
// 	for _, n := range lb.nodes {
// 		var gainWeight = float64(1.0)
// 		if n.region == region {
// 			gainWeight *= regionWeight
// 		}
// 		n.calculateWeight(lb.totalWeight, lb.totalConns, gainWeight)
// 		nodes = append(nodes, n)
// 	}
// 	sort.Slice(nodes, func(i, j int) bool {
// 		return nodes[i].currentWeight > nodes[j].currentWeight
// 	})
// 	if len(nodes) > 0 {
// 		nodes[0].chosen()
// 		lb.totalConns++
// 	}
// 	return
// }

// // *加锁后调用 weightedNodes 方法获取带权节点列表。
// // *解锁后遍历节点列表，最多返回 _maxNodes 个节点的域名和地址
// func (lb *LoadBalancer) NodeAddrs(region, domain string, regionWeight float64) (domains, addrs []string) {
// 	lb.nodesMutex.Lock()
// 	nodes := lb.weightedNodes(region, regionWeight)
// 	lb.nodesMutex.Unlock()
// 	for i, n := range nodes {
// 		if i == _maxNodes {
// 			break
// 		}
// 		domains = append(domains, n.hostname+domain)
// 		addrs = append(addrs, n.addrs...)
// 	}
// 	return
// }

// // *负责动态更新节点信息，确保负载均衡器能够根据最新的节点状态进行请求分配
// func (lb *LoadBalancer) Update(ins []*naming.Instance) {
// 	var (
// 		totalConns  int64
// 		totalWeight int64
// 		nodes       = make(map[string]*weightedNode, len(ins))
// 	)
// 	if len(ins) == 0 || float32(len(ins))/float32(len(lb.nodes)) < 0.5 {
// 		log.Errorf("load balancer update src:%d target:%d less than half", len(lb.nodes), len(ins))
// 		return
// 	}
// 	lb.nodesMutex.Lock()
// 	for _, in := range ins {
// 		if old, ok := lb.nodes[in.Hostname]; ok && old.updated == in.LastTs {
// 			nodes[in.Hostname] = old
// 			totalConns += old.currentConns
// 			totalWeight += old.fixedWeight
// 		} else {
// 			meta := in.Metadata
// 			weight, err := strconv.ParseInt(meta[model.MetaWeight], 10, 32)
// 			if err != nil {
// 				log.Errorf("instance(%+v) strconv.ParseInt(weight:%s) error(%v)", in, meta[model.MetaWeight], err)
// 				continue
// 			}
// 			conns, err := strconv.ParseInt(meta[model.MetaConnCount], 10, 32)
// 			if err != nil {
// 				log.Errorf("instance(%+v) strconv.ParseInt(conns:%s) error(%v)", in, meta[model.MetaConnCount], err)
// 				continue
// 			}
// 			nodes[in.Hostname] = &weightedNode{
// 				region:       in.Region,
// 				hostname:     in.Hostname,
// 				fixedWeight:  weight,
// 				currentConns: conns,
// 				addrs:        strings.Split(meta[model.MetaAddrs], ","),
// 				updated:      in.LastTs,
// 			}
// 			totalConns += conns
// 			totalWeight += weight
// 		}
// 	}
// 	lb.nodes = nodes
// 	lb.totalConns = totalConns
// 	lb.totalWeight = totalWeight
// 	lb.nodesMutex.Unlock()
// }
