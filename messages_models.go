package rmqrpc

type MessageRequest[T any] struct {
	Subject string      `json:"subject"`
	Payload T           `json:"payload"`
	Meta    MessageMeta `json:"meta"`
}

type MessageResponse[T any] struct {
	Response *T            `json:"response,omitempty"`
	Error    *MessageError `json:"err,omitempty"`
}

type MessageError struct {
	Status  int    `json:"status"`
	Message string `json:"message"`
	Name    string `json:"name"`
}

type MessageReqEvent[T any] struct {
	Pattern string            `json:"pattern"`
	Data    MessageRequest[T] `json:"data"`
	ID      string            `json:"id"`
}

type MessageResEvent[T any] struct {
	Response   MessageResponse[T] `json:"response"`
	IsDisposed bool               `json:"isDisposed"`
}

type MessageMeta struct {
	TraceId string `json:"traceId"`
}

func MakeMessage[T any](pattern string, payload T, traceId string) MessageReqEvent[T] {
	return MessageReqEvent[T]{
		Pattern: pattern,
		Data: MessageRequest[T]{
			Subject: pattern,
			Payload: payload,
			Meta:    MessageMeta{TraceId: traceId},
		},
	}

}
