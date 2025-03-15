// resolver.go

package discovery

import (
	log "github.com/golang/glog"
	"google.golang.org/grpc/resolver"
)

type IResolver interface {
	GetServiceNodes(host string) []*Node
	SetTargetNode(host string)
	SetManuResolver(host string, m resolver.Resolver)
}

type manuResolver struct {
	cc     resolver.ClientConn
	target resolver.Target
	r      IResolver
}

func (m manuResolver) ResolveNow(options resolver.ResolveNowOptions) {
	nodes := m.r.GetServiceNodes(m.target.URL.Host)
	addresses := make([]resolver.Address, 0)
	for i := range nodes {
		addresses = append(addresses, resolver.Address{
			Addr: nodes[i].Addr,
		})
	}
	if err := m.cc.UpdateState(resolver.State{
		Addresses: addresses,
	}); err != nil {
		log.Errorf("resolver update cc state error:%s", err.Error())
	}
}

func (manuResolver) Close() {

}
