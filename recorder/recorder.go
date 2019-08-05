package recorder

import (
	"time"
)

//Recorder - recorder interface
type Recorder interface {
	Add(time.Time, ...interface{})
	Summary() interface{}
}

func nextID(idx int) int {
	if recordSize-1 == idx {
		return 0
	} else {
		return idx + 1
	}
}

func prevID(currentID int) int {
	if 0 == currentID {
		return recordSize - 1
	}
	return currentID - 1
}
