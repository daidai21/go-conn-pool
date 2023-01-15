package go_conn_pool

import "errors"

var (
	ErrClosed        = errors.New("conn pool is closed")
	ErrConnOverLimit = errors.New("conn over limit")
)

type ConnPool interface {
	// Get 获取一个连接
	Get() (interface{}, error)

	// Put 归还一个连接
	Put(interface{}) error

	// Close 关闭连接池
	Close()

	// Len 返回当前池子内有效连接数量
	Len() int
}
