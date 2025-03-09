//*Package model 定义了与数据模型相关的常量或结构体。
package model

//*定义元数据相关的常量
const (
	//*MetaWeight 表示元数据中的权重信息。
	//*用于标识某个实体的权重值，通常用于负载均衡或优先级计算。
	MetaWeight = "weight"

	//*MetaOffline 表示元数据中的离线状态信息。
	//*用于标识某个实体是否处于离线状态。
	MetaOffline = "offline"

	//*MetaAddrs 表示元数据中的公共 IP 地址列表信息。
	//*用于存储某个实体的公共 IP 地址列表。
	MetaAddrs = "addrs"

	//*MetaIPCount 表示元数据中的 IP 地址数量信息。
	//*用于标识某个实体的 IP 地址数量。
	MetaIPCount = "ip_count"

	//*MetaConnCount 表示元数据中的连接数量信息。
	//*用于标识某个实体的当前连接数量。
	MetaConnCount = "conn_count"

	//*PlatformWeb 表示平台类型为 Web。
	//*用于标识某个平台或服务的类型为 Web。
	PlatformWeb = "web"
)
