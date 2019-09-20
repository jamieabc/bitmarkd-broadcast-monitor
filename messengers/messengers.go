package messengers

// Messenger - interface for messenger
type Messenger interface {
	Sender
	Validator
}

// Sender - interface for messenger send
type Sender interface {
	Send(...interface{}) error
}

// Validator - interface for deciding messenger object is valid or not
type Validator interface {
	Validate() bool
}
