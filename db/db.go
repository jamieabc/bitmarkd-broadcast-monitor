package db

import "io"

//DBWriter - db interface
type DBWriter interface {
	io.Writer
	io.Closer
}
