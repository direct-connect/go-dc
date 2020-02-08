package adc

import (
	"bytes"
	"io"

	"github.com/direct-connect/go-dc/lineproto"
)

// NewWriter creates a new ADC protocol writer with a default buffer size.
func NewWriter(w io.Writer) *Writer {
	return NewWriterSize(w, 0)
}

// NewWriterSize creates a new ADC protocol writer with a specified buffer size.
func NewWriterSize(w io.Writer, buf int) *Writer {
	return &Writer{
		Writer: lineproto.NewWriterSize(w, buf),
	}
}

// WriteStream is a stream of ADC messages to a specific destination.
type WriteStream interface {
	// WriteMessage writes a single message to the buffer.
	WriteMessage(msg Message) error
	// Flush flushes the buffer.
	Flush() error
}

// Writer is protocol message writer for ADC protocol. It's not safe for a concurrent use.
type Writer struct {
	*lineproto.Writer

	mbuf bytes.Buffer
}

// WriteKeepAlive writes an empty (keep alive) message.
// It is caller's responsibility to flush the writer.
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

// WriteInfo writes a single InfoPacket to the buffer.
func (w *Writer) WriteInfo(msg Message) error {
	return w.WritePacket(&InfoPacket{
		Msg: msg,
	})
}

// WriteHub writes a single HubPacket to the buffer.
func (w *Writer) WriteHub(msg Message) error {
	return w.WritePacket(&HubPacket{
		Msg: msg,
	})
}

// WriteClient writes a single ClientPacket to the buffer.
func (w *Writer) WriteClient(msg Message) error {
	return w.WritePacket(&ClientPacket{
		Msg: msg,
	})
}

// WriteBroadcast writes a single BroadcastPacket to the buffer.
func (w *Writer) WriteBroadcast(id SID, msg Message) error {
	return w.WritePacket(&BroadcastPacket{
		ID:  id,
		Msg: msg,
	})
}

// WriteDirect writes a single DirectPacket to the buffer.
func (w *Writer) WriteDirect(from, to SID, msg Message) error {
	return w.WritePacket(&DirectPacket{
		ID: from, To: to,
		Msg: msg,
	})
}

// WriteEcho writes a single EchoPacket to the buffer.
func (w *Writer) WriteEcho(from, to SID, msg Message) error {
	return w.WritePacket(&EchoPacket{
		ID: from, To: to,
		Msg: msg,
	})
}

// WriteFeature writes a single FeaturePacket to the buffer.
func (w *Writer) WriteFeature(from SID, sel []FeatureSel, msg Message) error {
	return w.WritePacket(&FeaturePacket{
		ID: from, Sel: sel,
		Msg: msg,
	})
}

// HubStream creates a stream of HubPackets.
func (w *Writer) HubStream() WriteStream {
	return &hubStream{w: w}
}

// InfoStream creates a stream of InfoPackets.
func (w *Writer) InfoStream() WriteStream {
	return &infoStream{w: w}
}

// ClientStream creates a stream of ClientPackets.
func (w *Writer) ClientStream() WriteStream {
	return &clientStream{w: w}
}

// BroadcastStream creates a stream of BroadcastPackets from a given SID.
func (w *Writer) BroadcastStream(from SID) WriteStream {
	return &broadcastStream{w: w, from: from}
}

// DirectStream creates a stream of DirectPackets from a given SID to a given target SID.
func (w *Writer) DirectStream(from, to SID) WriteStream {
	return &directStream{w: w, from: from, to: to}
}

// EchoStream creates a stream of EchoPackets from a given SID to a given target SID.
func (w *Writer) EchoStream(from, to SID) WriteStream {
	return &echoStream{w: w, from: from, to: to}
}

// FeatureStream creates a stream of FeaturePackets from a given SID to peers matching a selector.
func (w *Writer) FeatureStream(from SID, sel []FeatureSel) WriteStream {
	return &featureStream{w: w, from: from, sel: sel}
}

type hubStream struct {
	w *Writer
}

func (s *hubStream) WriteMessage(msg Message) error {
	return s.w.WriteHub(msg)
}

func (s *hubStream) Flush() error {
	return s.w.Flush()
}

type infoStream struct {
	w *Writer
}

func (s *infoStream) WriteMessage(msg Message) error {
	return s.w.WriteInfo(msg)
}

func (s *infoStream) Flush() error {
	return s.w.Flush()
}

type clientStream struct {
	w *Writer
}

func (s *clientStream) WriteMessage(msg Message) error {
	return s.w.WriteClient(msg)
}

func (s *clientStream) Flush() error {
	return s.w.Flush()
}

type broadcastStream struct {
	w    *Writer
	from SID
}

func (s *broadcastStream) WriteMessage(msg Message) error {
	return s.w.WriteBroadcast(s.from, msg)
}

func (s *broadcastStream) Flush() error {
	return s.w.Flush()
}

type directStream struct {
	w    *Writer
	from SID
	to   SID
}

func (s *directStream) WriteMessage(msg Message) error {
	return s.w.WriteDirect(s.from, s.to, msg)
}

func (s *directStream) Flush() error {
	return s.w.Flush()
}

type echoStream struct {
	w    *Writer
	from SID
	to   SID
}

func (s *echoStream) WriteMessage(msg Message) error {
	return s.w.WriteEcho(s.from, s.to, msg)
}

func (s *echoStream) Flush() error {
	return s.w.Flush()
}

type featureStream struct {
	w    *Writer
	from SID
	sel  []FeatureSel
}

func (s *featureStream) WriteMessage(msg Message) error {
	return s.w.WriteFeature(s.from, s.sel, msg)
}

func (s *featureStream) Flush() error {
	return s.w.Flush()
}
