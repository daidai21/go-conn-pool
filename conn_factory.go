package go_conn_pool

// ConnFactory 连接工厂
type ConnFactory interface {
	// Factory 生成连接的方法
	Factory() (interface{}, error)

	// Close 关闭连接的方法
	Close(interface{}) error

	// Ping 检查连接是否有效的方法
	Ping(interface{}) error
}

// TODO: 支持接口函数化那个
