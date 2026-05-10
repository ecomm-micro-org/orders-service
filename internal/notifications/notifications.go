package notifications

import (
	"context"
	"fmt"

	"github.com/trycourier/courier-go/v4"
	"github.com/trycourier/courier-go/v4/shared"
)

type Notifier struct {
	client courier.Client
}

func NewNotifier() *Notifier {
	return &Notifier{}
}

func (n *Notifier) SendNotificationToUser(template string, message string) error {
	response, err := n.client.Send.Message(context.TODO(), courier.SendMessageParams{
		Message: courier.SendMessageParamsMessage{
			To: courier.SendMessageParamsMessageToUnion{
				OfUserRecipient: &shared.UserRecipientParam{
					UserID: courier.String("your_user_id"),
				},
			},
			Template: courier.String("order-placed"),
			Data: map[string]any{
				"foo": "bar",
			},
		},
	})
	if err != nil {
		return err
	}
	fmt.Printf("%+v\n", response.RequestID)
	return nil
}
