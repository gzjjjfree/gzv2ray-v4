package proto

import (
	"bytes"
	"io"

	"github.com/gzjjjfree/gzv2ray-v4/app/browserforwarder/handler/proto/struc"
)

type WebsocketLength struct {
	Length uint32 `struc:"uint32,big,sizeof=Data"`
	Data   []byte
}

type WebsocketConnectionRequest struct {
	DestinationSize uint32 `struc:"uint32,big,sizeof=Destination"`
	Destination     string
}

func ReadRequest(reader io.Reader) (*WebsocketConnectionRequest, error) {
	var err error
	var data WebsocketLength
	err = struc.Unpack(reader, &data)
	if err != nil {
		return nil, err
	}

	var Request WebsocketConnectionRequest
	err = struc.Unpack(bytes.NewReader(data.Data), &Request)
	if err != nil {
		return nil, err
	}
	return &Request, nil
}

func WriteRequest(writer io.Writer, req *WebsocketConnectionRequest) error {
	var err error
	reqbuf := bytes.NewBuffer(nil)
	err = struc.Pack(reqbuf, req)
	if err != nil {
		return err
	}
	reqdata := reqbuf.Bytes()

	var data WebsocketLength

	data.Data = reqdata
	data.Length = uint32(len(reqdata))

	err = struc.Pack(writer, &data)

	if err != nil {
		return err
	}

	return nil
}
