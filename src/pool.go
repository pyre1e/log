package main

import (
	"errors"
	"time"

	"github.com/ClickHouse/clickhouse-go/v2"
	"github.com/ClickHouse/clickhouse-go/v2/lib/driver"
)

type ClickHousePool interface {
	Init(count int) error
	Acquire() (*driver.Conn, int, error)
	Release(id int) error
}

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

func (pc *poolConnection) Renew(period time.Duration) {
	for {
		time.Sleep(period)

		if pc.conn != nil {
			pc.conn.Close()
		}

		if err := pc.Connect(); err != nil {
			return
		}
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
	lifetime    int
}

func (pool *AppPool) Init(size int, lifetime int) {
	pool.size = size
	pool.lifetime = lifetime
	pool.connections = make([]poolConnection, size)
	pool.free = &appPoolFreeStack{
		ids:     make([]int, size),
		pointer: 0,
	}
	for i := 0; i < size; i++ {
		pool.free.push(i)
	}
}

func (pool *AppPool) Acquire() (*driver.Conn, int, error) {

	id := pool.getNextId()
	pc := &pool.connections[id]

	if !pc.available {
		err := pc.Connect()
		if err != nil {
			return nil, 0, err
		}

		go pc.Renew(time.Second * time.Duration(pool.lifetime))
	}

	return &pc.conn, id, nil
}

func (pool *AppPool) Release(id int) {
	pool.free.push(id)
}

func (pool *AppPool) getNextId() int {
	for {
		if id := pool.free.pop(); id != -1 {
			return id
		}
		time.Sleep(time.Millisecond * 50)
	}
}
