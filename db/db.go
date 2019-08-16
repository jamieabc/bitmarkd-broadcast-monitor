package db

import "io"

//DBWriter - db interface
type DBWriter interface {
	io.Writer
	io.Closer
	Setter
}

//Setter - set data to write
type Setter interface {
	Fields(map[string]interface{})
	Tags(map[string]string)
}
