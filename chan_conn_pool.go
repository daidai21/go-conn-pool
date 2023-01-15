package go_conn_pool

import (
	"errors"
	"fmt"
	"sync"
	"time"
)

var _ ConnPool = (*chanConnPool)(nil)

type chanConnPool struct {
	mu      sync.Mutex
	factory ConnFactory // 工厂

	conns chan *interface{}

	connCnt int // 当前的连接数量，包含借出去的

	maxIdleCount int           // 最多的空闲连接数量
	maxOpen      int           // 最大的连接数量， =0表示无上限
	maxLifetime  time.Duration // 连接池里面的连接最大存活时长
	maxIdleTime  time.Duration // 设置最大空闲时间
}

func NewChanConnPool(conf *ChanConnConf) (ConnPool, error) {
	if conf != nil {
		if conf.maxIdleCount <= 0 ||
			conf.maxIdleCount > conf.maxOpen ||
			conf.maxLifetime <= 0 ||
			conf.maxIdleTime <= 0 ||
			conf.Factory == nil {
			return nil, errors.New("ChanConnPool invalid param by init")
		}
	}

	p := &chanConnPool{
		mu:           sync.Mutex{},
		factory:      conf.Factory,
		conns:        make(chan *interface{}, conf.maxIdleCount),
		connCnt:      0,
		maxIdleCount: conf.maxIdleCount,
		maxOpen:      conf.maxOpen,
		maxLifetime:  conf.maxLifetime,
		maxIdleTime:  conf.maxIdleTime,
	}

	// 初始化化连接
	for i := 0; i < conf.maxIdleCount; i++ {
		conn, e := p.factory.Factory()
		if e != nil {
			p.Close()
			return nil, fmt.Errorf("factory is not able to fill the pool: %s", e)
		}
		p.conns <- &conn
		p.connCnt++
	}

	return p, nil
}

type ChanConnConf struct {
	maxIdleCount int           // 最多的空闲连接数量
	maxOpen      int           // 最大的连接数量， =0表示无上限
	maxLifetime  time.Duration // 连接池里面的连接最大存活时长
	maxIdleTime  time.Duration // 设置最大空闲时间

	Factory ConnFactory // 工厂
}

func (c *chanConnPool) Get() (interface{}, error) {
	if c.conns == nil {
		return nil, ErrClosed
	}
	for {
		select {
		// 从 conn 中获取现成的连接
		case conn := <-c.conns:
			if conn == nil {
				return nil, ErrClosed
			}
			// 判断连接是否有效
			if e := c.factory.Ping(conn); e != nil {
				_ = c.factory.Close(conn)
				c.connCnt--
				continue
			}
			return conn, nil
		// 没有空闲连接
		default:
			c.mu.Lock()
			if c.connCnt > c.maxOpen {
				return nil, ErrConnOverLimit
			}
			conn, e := c.factory.Factory()
			if e != nil {
				return nil, e
			}
			if e := c.factory.Ping(conn); e != nil {
				_ = c.factory.Close(e)
				return nil, e
			}
			c.connCnt++
			return conn, nil
		}

	}

}

func (c *chanConnPool) Put(conn interface{}) error {
	if conn == nil {
		return errors.New("connection is nil. rejecting")
	}

	c.mu.Lock()
	defer c.mu.Unlock()

	// 如果连接数量已经大于最大数量
	if len(c.conns) >= c.maxOpen {
		_ = c.factory.Close(conn)
		c.connCnt--
		return nil
	}
	c.conns <- &conn
	return nil
}

func (c *chanConnPool) Close() {
	c.mu.Lock()
	defer c.mu.Unlock()

	for conn := range c.conns {
		_ = c.factory.Close(conn)
	}
	c.factory = nil
	c.connCnt = 0
}

// Len 这里的返回的数量可能大于真实可用的数量
func (c *chanConnPool) Len() int {
	return c.connCnt
}
