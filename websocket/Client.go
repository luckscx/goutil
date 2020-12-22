package part

import (
	"time"
	"errors"
	"reflect"

	"github.com/gorilla/websocket"

	s "github.com/qydysky/part/signal"
)

type Client struct {
	Url string
	SendChan chan interface{}
	RecvChan chan []byte

	TO int
	Header map[string][]string
	
	Ping Ping

	Msg_normal_close string
	Func_normal_close func()
	Func_abort_close func()
	
	err error
	signal *s.Signal
}

type ws_msg struct {
	Type int
	Msg []byte
}

type Ping struct {
	Msg []byte
	Period int
	had_pong bool
}

func New_client(config Client) (o *Client) {
	tmp := Client{
		TO: 300 * 1000,
		Func_normal_close:func(){},
		Func_abort_close:func(){},
		SendChan: make(chan interface{}, 1e4),
		RecvChan: make(chan []byte, 1e4),
	}
	tmp.Url = config.Url
	if v := config.TO;v != 0 {tmp.TO = v}
	tmp.Header = config.Header
	tmp.Msg_normal_close = config.Msg_normal_close
	if v := config.Func_normal_close;v != nil {tmp.Func_normal_close = v}
	if v := config.Func_abort_close;v != nil {tmp.Func_abort_close = v}
	if config.Ping.Period != 0 {tmp.Ping = config.Ping}
	return &tmp
}

func (i *Client) Handle() (o *Client) {
	o = i

	if o.signal.Islive() {return}
	o.signal = s.Init()

	if o.Url == "" {
		o.signal.Done()
		o.err = errors.New(`url == ""`)
		return
	}

	go func(){
		defer func(){
			close(o.RecvChan)
			o.signal.Done()
		}()

		c, _, err := websocket.DefaultDialer.Dial(o.Url, o.Header)
		if err != nil {return}
		defer c.Close()

		done := make(chan struct{})

		go func() {
			defer close(done)
	
			for {
				c.SetReadDeadline(time.Now().Add(time.Millisecond*time.Duration(o.TO)))
				msg_type, message, err := c.ReadMessage()
				if err != nil {
					if e, ok := err.(*websocket.CloseError); ok {
						switch e.Code {
						case websocket.CloseNormalClosure:o.Func_normal_close()
						case websocket.CloseAbnormalClosure:o.Func_abort_close()
						default:
						}
						o.err = e
					}
					return
				}
				switch msg_type {
				case websocket.PingMessage:
					o.SendChan <- ws_msg{
						Type:websocket.PongMessage,
						Msg:message,
					}
				case websocket.PongMessage:
					o.Ping.had_pong = true
				default:o.RecvChan <- message
				}
			}
		}()
	
		for {
			select {
			case <- done:
				return
			case t := <- o.SendChan:
				var err error
				switch reflect.ValueOf(t).Type().Name() {
				case `ws_msg`:
					err = c.WriteMessage(t.(ws_msg).Type, t.(ws_msg).Msg)
				default:
					err = c.WriteMessage(websocket.TextMessage, t.([]byte))
				}
				if err != nil {
					o.err = err
					return
				}
				c.SetWriteDeadline(time.Now().Add(time.Millisecond*time.Duration(o.TO)))
			case <- o.signal.Chan:
				err := c.WriteMessage(websocket.CloseMessage, websocket.FormatCloseMessage(websocket.CloseNormalClosure, o.Msg_normal_close))
				if err != nil {o.err = err}
				select {
				case <- done:
				case <- time.After(time.Second):
				}
				return
			}
		}	
	}()
	return
}

func (o *Client) Heartbeat() (err error) {
	if !o.signal.Islive() {return errors.New(`not alive`)}

	var ticker_ping = time.NewTicker(time.Duration(o.TO)*time.Millisecond)
	if o.Ping.Period > 0 {
		if o.Ping.Period < o.TO {
			ticker_ping.Reset(time.Duration(o.Ping.Period)*time.Millisecond)
			o.Ping.had_pong = true
		} else {
			err = errors.New(`Ping.Period < o.TO`)
		}
	} else {ticker_ping.Stop()}

	go func(ticker_ping *time.Ticker){
		defer ticker_ping.Stop()
		for {
			select {
			case <-ticker_ping.C:
				if !o.Ping.had_pong {
					o.err = errors.New(`Pong fail!`)
					o.Close()
					return
				}
				o.SendChan <- ws_msg{
					Type:websocket.PingMessage,
					Msg:o.Ping.Msg,
				}
				o.Ping.had_pong = false
			case <- o.signal.Chan:
				return
			}
		}
	}(ticker_ping)

	return
}

func (o *Client) Close() {
	o.signal.Done()
}

func (o *Client) Isclose() bool {
	return !o.signal.Islive()
}

func (o *Client) Error() error {
	return o.err
}