// Package netchan creates network channels. The network channels are managed by
// glow agent.
package netchan

import (
	"bytes"
	"crypto/tls"
	"encoding/gob"
	"flag"
	"log"
	"reflect"
	"strconv"
	"sync"

	"github.com/chrislusf/glow/netchan/receiver"
	"github.com/chrislusf/glow/netchan/sender"
	"github.com/chrislusf/glow/util"
)

type NetworkOption struct {
	AgentPort int
}

var Option NetworkOption

func init() {
	flag.IntVar(&Option.AgentPort, "glow.agent.port", 8931, "agent port")
}

func GetLocalSendChannel(tlsConfig *tls.Config, name string, wg *sync.WaitGroup) (chan []byte, error) {
	return sender.NewSendChannel(tlsConfig, name, Option.AgentPort, wg)
}

func GetLocalReadChannel(tlsConfig *tls.Config, name string, chanBufferSize int) (chan []byte, error) {
	return GetDirectReadChannel(tlsConfig, name, "localhost:"+strconv.Itoa(Option.AgentPort), chanBufferSize)
}

func GetDirectReadChannel(tlsConfig *tls.Config, name, location string, chanBufferSize int) (chan []byte, error) {
	rc := receiver.NewReceiveChannel(tlsConfig, name, 0)
	return rc.GetDirectChannel(location, chanBufferSize)
}

func GetDirectSendChannel(tlsConfig *tls.Config, name string, target string, wg *sync.WaitGroup) (chan []byte, error) {
	return sender.NewDirectSendChannel(tlsConfig, name, target, wg)
}

func ConnectRawReadChannelToTyped(c chan []byte, out chan reflect.Value, t reflect.Type, wg *sync.WaitGroup) (status *util.ChannelStatus) {
	status = util.NewChannelStatus()
	wg.Add(1)
	go func() {
		defer wg.Done()
		status.ReportStart()

		for data := range c {
			dec := gob.NewDecoder(bytes.NewBuffer(data))
			v := reflect.New(t)
			if err := dec.DecodeValue(v); err != nil {
				log.Fatal("data type:", v.Kind(), " decode error:", err)
			} else {
				out <- reflect.Indirect(v)
				status.ReportAdd(1)
			}
		}

		close(out)
		status.ReportClose()
	}()
	return status
}

func ConnectTypedWriteChannelToRaw(writeChan reflect.Value, c chan []byte, wg *sync.WaitGroup) (status *util.ChannelStatus) {
	status = util.NewChannelStatus()
	wg.Add(1)
	go func() {
		defer wg.Done()
		status.ReportStart()

		var t reflect.Value
		for ok := true; ok; {
			if t, ok = writeChan.Recv(); ok {
				var buf bytes.Buffer
				enc := gob.NewEncoder(&buf)
				if err := enc.EncodeValue(t); err != nil {
					log.Fatal("data type:", t.Type().String(), " ", t.Kind(), " encode error:", err)
				}
				c <- buf.Bytes()
				status.ReportAdd(1)
			}
		}
		close(c)
		status.ReportClose()
	}()
	return status
}
