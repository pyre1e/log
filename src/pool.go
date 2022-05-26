package main

import (
	"context"
	"errors"
	"time"

	"github.com/ClickHouse/clickhouse-go/v2"
	"github.com/ClickHouse/clickhouse-go/v2/lib/driver"
)

type appPoolFreeStack struct {
	ids     []int
	pointer int
}

func (stack *appPoolFreeStack) push(id int) error {

	if stack.pointer >= len(stack.ids) {
		return errors.New("Cannot free more connections than is pool size")
	}

	stack.ids[stack.pointer] = id
	stack.pointer++

	return nil
}

func (stack *appPoolFreeStack) pop() int {
	if stack.pointer == 0 {
		return -1
	}

	stack.pointer--

	return stack.ids[stack.pointer]
}

type poolConnection struct {
	conn      driver.Conn
	created   time.Time
	available bool
}

func (pc *poolConnection) Renew() {
	if pc.conn != nil {
		pc.conn.Close()
	}
	if err := pc.Connect(); err != nil {
		return
	}
}

func (pc *poolConnection) Connect() error {
	var err error
	pc.conn, err = clickhouse.Open(&clickhouse.Options{
		Addr: []string{"clickhouse:9000"},
		Auth: clickhouse.Auth{
			Database: "default",
			Username: "default",
			Password: "",
		},
		MaxIdleConns: 4,
		Compression: &clickhouse.Compression{
			Method: 0x02,
		},
	})
	if err != nil {
		pc.available = false
		return err
	}

	return nil
}

type AppPool struct {
	connections []poolConnection
	free        *appPoolFreeStack
	size        int
	newconn     chan bool
}

func (pool *AppPool) Init(size int) {
	pool.size = size
	pool.connections = make([]poolConnection, size)
	pool.free = &appPoolFreeStack{
		ids:     make([]int, size),
		pointer: 0,
	}
	for i := 0; i < size; i++ {
		pool.free.push(i)
	}
}

func (pool *AppPool) WaitForConnection() {
	tmp := &poolConnection{}
	for {
		err := tmp.Connect()
		if err != nil {
			time.Sleep(time.Second)
			continue
		}
		err = tmp.conn.Ping(context.Background())
		if err != nil {
			time.Sleep(time.Second)
			continue
		}
		return
	}
}

func (pool *AppPool) Acquire() (*poolConnection, int, error) {

	id := pool.getNextId()
	pc := &pool.connections[id]

	if !pc.available {
		err := pc.Connect()
		if err != nil {
			return nil, 0, err
		}
	}

	return pc, id, nil
}

func (pool *AppPool) Release(id int) {
	pool.free.push(id)
	go func() {
		pool.newconn <- true
	}()
}

func (pool *AppPool) getNextId() int {
	for {
		if id := pool.free.pop(); id != -1 {
			return id
		}
		<-pool.newconn
	}
}
