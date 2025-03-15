// builder.go

package discovery

import "google.golang.org/grpc/resolver"

const (
	etcdScheme = "etcd"
)

type builder struct{}

func (builder) Build(target resolver.Target, cc resolver.ClientConn, opts resolver.BuildOptions) (resolver.Resolver, error) {
	mr := manuResolver{
		cc:     cc,
		target: target,
		r:      EResolver,
	}
	//*记录解析器
	mr.r.SetManuResolver(target.URL.Host, mr)
	//*记录需要解析的节点
	mr.r.SetTargetNode(target.URL.Host)
	return mr, nil
}

func (builder) Scheme() string {
	return etcdScheme
}

func init() {
	resolver.Register(builder{})
}
