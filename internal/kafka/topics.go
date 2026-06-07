package kafka

type Topic string

const (
	TopicOrderCreated Topic = "orders.created"
)

func (t Topic)String()string{
	return string(t)
}
