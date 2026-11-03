package groff

import (
	"errors"
	"fmt"
	"io"
)

func beginBlockCapture(handler BlockHandler, header BlockHeader, limit int) (*blockCapture, error) {
	rule, err := handler.Begin(header)
	if err == nil {
		err = validateBlockRule(rule)
	}
	if err != nil {
		return nil, blockSyntax(header.Line.Span.Start, err)
	}
	return newBlockCapture(header, rule, limit), nil
}
func completeCapturedBlock(handler BlockHandler, capture *blockCapture) (Token, error) {
	err := capture.finishError()
	var token Token
	if err == nil {
		token, err = handler.ParseBlock(capture.source.clone())
		if err == nil && nilInterface(token) {
			err = fmt.Errorf("%w: %s returned a nil block token", ErrInvalidHandler, capture.source.Header.Command)
		}
		if err != nil {
			err = blockSyntax(capture.source.Header.Line.Span.Start, err)
		}
	}
	if err != nil {
		return &TokenUnknownControl{RawLine: capture.source.Source()}, err
	}
	return token, nil
}
func readCapturedBlock(handler BlockHandler, capture *blockCapture, stream *documentStream) (Token, error) {
	for {
		line, readErr := stream.next()
		if readErr != nil && readErr != io.EOF {
			if line.raw != "" {
				capture.source.Body = append(capture.source.Body, BlockLine{RawSource: line.raw, Span: line.span})
			}
			if tail := stream.drainTail(); tail.raw != "" {
				capture.source.Tail = BlockLine{RawSource: tail.raw, Span: tail.span}
			}
			return &TokenUnknownControl{RawLine: capture.source.Source()}, errors.Join(capture.failure, readErr)
		}
		if line.raw == "" && readErr == io.EOF {
			return completeCapturedBlock(handler, capture)
		}
		if err := capture.consume(BlockLine{RawSource: line.raw, Span: line.span}); err != nil {
			if tail := stream.drainTail(); tail.raw != "" {
				capture.source.Tail = BlockLine{RawSource: tail.raw, Span: tail.span}
			}
			return &TokenUnknownControl{RawLine: capture.source.Source()}, err
		}
		if capture.source.Closed {
			return completeCapturedBlock(handler, capture)
		}
	}
}
