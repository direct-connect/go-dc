package adc

import (
	"bytes"
	"io"

	"github.com/direct-connect/go-dc/lineproto"
)

func NewWriter(w io.Writer) *Writer {
	return NewWriterSize(w, 0)
}

func NewWriterSize(w io.Writer, buf int) *Writer {
	return &Writer{
		Writer: lineproto.NewWriterSize(w, buf),
	}
}

type WriteStream interface {
	WriteMessage(msg Message) error
	Flush() error
}

// Writer is not safe for concurrent use.
type Writer struct {
	*lineproto.Writer

	mbuf bytes.Buffer
}

func (w *Writer) WriteKeepAlive() error {
	return w.WriteLine([]byte{lineDelim})
}

func (w *Writer) WritePacket(p Packet) error {
	w.mbuf.Reset()
	err := p.MarshalPacketADC(&w.mbuf)
	if err != nil {
		return err
	}
	return w.WriteLine(w.mbuf.Bytes())
}

// Flush the underlying buffer. Should be called after each WritePacket batch.
func (w *Writer) Flush() error {
	return w.Writer.Flush()
}

func (w *Writer) WriteInfoMsg(msg Message) error {
	return w.WritePacket(&InfoPacket{
		Msg: msg,
	})
}

func (w *Writer) WriteHubMsg(msg Message) error {
	return w.WritePacket(&HubPacket{
		Msg: msg,
	})
}

func (w *Writer) WriteClientMsg(msg Message) error {
	return w.WritePacket(&ClientPacket{
		Msg: msg,
	})
}

func (w *Writer) WriteBroadcast(id SID, msg Message) error {
	return w.WritePacket(&BroadcastPacket{
		ID:  id,
		Msg: msg,
	})
}

func (w *Writer) WriteDirect(from, to SID, msg Message) error {
	return w.WritePacket(&DirectPacket{
		ID: from, To: to,
		Msg: msg,
	})
}

func (w *Writer) WriteEcho(from, to SID, msg Message) error {
	return w.WritePacket(&EchoPacket{
		ID: from, To: to,
		Msg: msg,
	})
}

func (w *Writer) Broadcast(from SID) WriteStream {
	return &broadcastStream{w: w, from: from}
}

type broadcastStream struct {
	w    *Writer
	from SID
}

func (r *broadcastStream) WriteMessage(msg Message) error {
	return r.w.WriteBroadcast(r.from, msg)
}

func (r *broadcastStream) Flush() error {
	return r.w.Flush()
}

func (w *Writer) Direct(from, to SID) WriteStream {
	return &directStream{w: w, from: from, to: to}
}

type directStream struct {
	w    *Writer
	from SID
	to   SID
}

func (r *directStream) WriteMessage(msg Message) error {
	return r.w.WriteDirect(r.from, r.to, msg)
}

func (r *directStream) Flush() error {
	return r.w.Flush()
}
