package messaging

type Messenger interface {
	SendMsg(msg string) error
}
