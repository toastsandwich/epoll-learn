package server

import (
	"bytes"
	"fmt"
	"sync"
)

type header struct {
	Key   []byte
	Value []byte
}

type headers []header

func newHeaders() headers {
	return make(headers, 0)
}

func (h *headers) Add(k, v []byte) {
	*h = append(*h, header{k, v})
}

func (h headers) Get(k []byte) []byte {
	for _, header := range h {
		if bytes.EqualFold(header.Key, k) {
			return header.Value
		}
	}
	return nil
}

type Request struct {
	Version []byte
	Method  []byte
	Path    []byte
	Headers headers
	Body    []byte // optional
}

var reqPool = sync.Pool{
	New: func() any {
		return &Request{}
	},
}

func getRequest() *Request {
	r := reqPool.Get().(*Request)
	r.Version, r.Path, r.Method, r.Headers, r.Body = nil, nil, nil, nil, nil
	return r
}

func putRequest(r *Request) {
	reqPool.Put(r)
}

/*
GET /hello.txt HTTP/1.0\r\n
User-Agent: TestClient\r\n
\r\n
*/

func parseRequest(p []byte) (*Request, error) {
	req := getRequest()

	firstLine, restLines, found := bytes.Cut(p, []byte("\r\n"))
	if !found {
		return nil, fmt.Errorf("error parsing (no info line)")
	}

	methodTill := bytes.IndexByte(firstLine, ' ')
	if methodTill == -1 {
		return nil, fmt.Errorf("error parsing (no method)")
	}

	pathTill := bytes.IndexByte(firstLine[methodTill+1:], ' ') + (methodTill + 1)
	if pathTill == -1 {
		return nil, fmt.Errorf("error parsing (no path)")
	}

	versTill := len(firstLine[pathTill+1:]) + (pathTill + 1)
	if versTill == -1 {
		return nil, fmt.Errorf("error parsing (no version)")
	}

	req.Method = p[:methodTill]
	req.Path = p[methodTill:pathTill]
	req.Version = p[pathTill:versTill]

	headerLine, body, _ := bytes.Cut(restLines, []byte("\r\n\r\n"))
	for len(headerLine) > 0 {
		var line []byte
		line, headerLine, _ = bytes.Cut(headerLine, []byte("\r\n"))

		if len(line) == 0 {
			break
		}

		key, val, found := bytes.Cut(line, []byte(":"))
		if !found {
			continue
		}
		if req.Headers == nil {
			req.Headers = newHeaders()
		}
		key = bytes.TrimSpace(key)
		val = bytes.TrimSpace(val)
		req.Headers.Add(key, val)
	}

	if len(body) > 0 {
		req.Body = body
	}

	return req, nil
}

func toBytes(req *Request) []byte {
	return []byte("done")
}

type Response struct {
}

func toByte(res *Response) []byte {
	return []byte{}
}
