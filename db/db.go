package db

import (
	"io"
	"sync"
)

//DBWriter - db interface
type DBWriter interface {
	Adder
	io.Closer
	Looper
	sync.Locker
}

//Adder - set Data to write
type Adder interface {
	Add(InfluxData)
}

//Looper - loop to periodic write
type Looper interface {
	Loop(chan struct{})
}
