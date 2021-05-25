package message

import (
	"bytes"
	"errors"
	"fmt"
)

const (
	TAB = '\t'
	LF  = '\n'
)

const (
	POLICY_WITHOUT_STATUS = "POLICY_WITHOUT_STATUS"
	POLICY_WITH_DURATION  = "POLICY_WITH_DURATION"
	POLICY_DEFAULT        = "POLICY_DEFAULT"
)

type ReadableEncoder struct {
	encoderBase
}

func NewReadableEncoder() *ReadableEncoder {
	return &ReadableEncoder{}
}

func (e *ReadableEncoder) writeString(buf *bytes.Buffer, s string) (err error) {
	if _, err = buf.WriteString(s); err != nil {
		return
	}
	if _, err = buf.WriteRune(TAB); err != nil {
		return
	}
	return
}

func (e *ReadableEncoder) writeRaw(buf *bytes.Buffer, s string) (err error) {
	if _, err = buf.Write([]byte(s)); err != nil {
		return
	}
	if _, err = buf.WriteRune(TAB); err != nil {
		return
	}
	return
}

func (e *ReadableEncoder) writeRawByte(buf *bytes.Buffer, s []byte) (err error) {
	if _, err = buf.Write(s); err != nil {
		return
	}
	if _, err = buf.WriteRune(TAB); err != nil {
		return
	}
	return
}

func (e *ReadableEncoder) encodeLine(buf *bytes.Buffer, message Messager, leader rune, policy string) (err error) {
	if _, err = buf.WriteRune(leader); err != nil {
		return
	}
	if m, ok := message.(*Transaction); ok && leader == 'T' {
		if err = e.writeString(buf, m.GetTime().Add(m.GetDuration()).Format("2006-01-02 15:04:05.999")); err != nil {
			return
		}
	} else {
		if err = e.writeString(buf, message.GetTime().Format("2006-01-02 15:04:05.999")); err != nil {
			return
		}
	}

	if err = e.writeRaw(buf, message.GetType()); err != nil {
		return
	}
	if err = e.writeRaw(buf, message.GetName()); err != nil {
		return
	}

	if policy != POLICY_WITHOUT_STATUS {

		if err = e.writeRaw(buf, message.GetStatus()); err != nil {
			return
		}

		if m, ok := message.(*Transaction); ok && policy == POLICY_WITH_DURATION {
			if _, err = buf.WriteString(fmt.Sprint(m.GetDuration().Microseconds())); err != nil {
				return
			}
			if err = e.writeString(buf, "us"); err != nil {
				return
			}
		}

		if err = e.writeRawByte(buf, message.GetData().Bytes()); err != nil {
			return
		}
	}

	if _, err = buf.WriteRune(LF); err != nil {
		return
	}

	return
}

func (e *ReadableEncoder) EncodeMessage(buf *bytes.Buffer, message Messager) (err error) {
	return encodeMessage(e, buf, message)
}

func (e *ReadableEncoder) EncodeHeader(buf *bytes.Buffer, header *Header) (err error) {
	if err = e.writeString(buf, ReadableProtocol); err != nil {
		return
	}
	if err = e.writeString(buf, header.Domain); err != nil {
		return
	}
	if err = e.writeString(buf, header.Hostname); err != nil {
		return
	}
	if err = e.writeString(buf, header.Ip); err != nil {
		return
	}
	if err = e.writeString(buf, defaultThreadGroupName); err != nil {
		return
	}
	if err = e.writeString(buf, defaultThreadId); err != nil {
		return
	}
	if err = e.writeString(buf, defaultThreadName); err != nil {
		return
	}
	if err = e.writeString(buf, header.MessageId); err != nil {
		return
	}
	if err = e.writeString(buf, header.ParentMessageId); err != nil {
		return
	}
	if err = e.writeString(buf, header.RootMessageId); err != nil {
		return
	}
	// sessionToken.
	if _, err = buf.WriteString(""); err != nil {
		return
	}
	if _, err = buf.WriteRune(LF); err != nil {
		return
	}
	return
}

func (e *ReadableEncoder) EncodeTransaction(buf *bytes.Buffer, trans *Transaction) (err error) {
	if trans == nil {
		err = errors.New("trans is null")
		return
	}

	children := trans.GetChildren()
	if len(children) == 0 {
		return e.encodeLine(buf, trans, 'A', POLICY_WITH_DURATION)
	} else {
		if err = e.encodeLine(buf, trans, 't', POLICY_WITHOUT_STATUS); err != nil {
			return
		}
		for _, message := range children {
			if err = e.EncodeMessage(buf, message); err != nil {
				return
			}
		}
		if err = e.encodeLine(buf, trans, 'T', POLICY_WITH_DURATION); err != nil {
			return
		}
	}
	return
}

func (e *ReadableEncoder) EncodeEvent(buf *bytes.Buffer, m *Event) (err error) {
	return e.encodeLine(buf, m, 'E', POLICY_DEFAULT)
}

func (e *ReadableEncoder) EncodeHeartbeat(buf *bytes.Buffer, m *Heartbeat) (err error) {
	return e.encodeLine(buf, m, 'H', POLICY_DEFAULT)
}

func (e *ReadableEncoder) EncodeMetric(buf *bytes.Buffer, m *Metric) (err error) {
	return e.encodeLine(buf, m, 'M', POLICY_DEFAULT)
}
