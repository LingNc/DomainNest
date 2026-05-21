package ws

const (
	TypeNewMessage         = "new_message"
	TypeNewNotification    = "new_notification"
	TypeFriendRequest      = "friend_request"
	TypeUnreadUpdate       = "unread_update"
	TypeConversationUpdate = "conversation_update"
	TypeDomainTreeUpdate   = "domain_tree_update"
)

// Envelope is the JSON wrapper sent over the wire.
type Envelope struct {
	Type    string      `json:"type"`
	Payload interface{} `json:"payload"`
}
