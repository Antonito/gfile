package rtc

import (
	"fmt"

	"github.com/pion/webrtc/v3"
)

/// Configuration of a RTC client.
type Configuration struct {
	ICEServers  []webrtc.ICEServer
	DataChannel DataChannelConfiguration
}

/// DataChannelConfiguration of a RTC client.
type DataChannelConfiguration struct {
	InitParams *webrtc.DataChannelInit

	OnOpen              func(*webrtc.DataChannel)
	OnClose             func()
	OnMessage           func(webrtc.DataChannelMessage)
	OnError             func(error)
	OnBufferedAmountLow func(*webrtc.DataChannel)

	BufferThreshold *uint64
}

/// Client .
type Client struct {
	cfg Configuration

	api            *webrtc.API
	pc             *webrtc.PeerConnection
	mediaEngine    webrtc.MediaEngine
	settingsEngine webrtc.SettingEngine
}

/// NewClient creates a new RTC Client.
func NewClient(cfg Configuration) *Client {
	sess := &Client{
		cfg: cfg,
	}

	return sess
}

func (c *Client) Close() {
	if c.pc != nil {
		c.pc.Close()
		c.pc = nil
	}
}

func (c *Client) makeConnection() error {
	c.api = webrtc.NewAPI(webrtc.WithSettingEngine(c.settingsEngine), webrtc.WithMediaEngine(&c.mediaEngine))

	config := webrtc.Configuration{
		ICEServers: c.cfg.ICEServers,
	}

	pc, err := c.api.NewPeerConnection(config)
	if err != nil {
		return err
	}

	c.pc = pc

	return nil
}

func (c *Client) MakeLocalOffer() (*webrtc.SessionDescription, error) {
	if err := c.makeConnection(); err != nil {
		return nil, err
	}

	ch, err := c.pc.CreateDataChannel("gfile", c.cfg.DataChannel.InitParams)
	if err != nil {
		return nil, err
	}

	c.setupDataChannel(ch)

	offer, err := c.pc.CreateOffer(nil)
	if err != nil {
		return nil, err
	}

	gatherComplete := webrtc.GatheringCompletePromise(c.pc)
	if err := c.pc.SetLocalDescription(offer); err != nil {
		return nil, err
	}
	<-gatherComplete

	return c.pc.LocalDescription(), nil
}

func (c *Client) SetRemoteOffer(offer webrtc.SessionDescription) error {
	if err := c.makeConnection(); err != nil {
		return err
	}

	c.pc.OnICEConnectionStateChange(func(c webrtc.ICEConnectionState) {
		fmt.Printf("webrtc ice connection state: %v\n", c)
	})

	c.pc.OnDataChannel(func(dataChannel *webrtc.DataChannel) {
		c.setupDataChannel(dataChannel)
	})

	return c.pc.SetRemoteDescription(offer)
}

func (c *Client) SetAnswer(answer webrtc.SessionDescription) error {
	c.pc.OnICEConnectionStateChange(func(c webrtc.ICEConnectionState) {
		fmt.Printf("webrtc ice connection state: %v\n", c)
	})

	c.pc.OnDataChannel(func(dataChannel *webrtc.DataChannel) {
		c.setupDataChannel(dataChannel)
	})

	if err := c.pc.SetRemoteDescription(answer); err != nil {
		return err
	}

	return nil
}

func (c *Client) MakeAnswer() (*webrtc.SessionDescription, error) {
	answer, err := c.pc.CreateAnswer(nil)
	if err != nil {
		return nil, err
	}

	if err := c.pc.SetLocalDescription(answer); err != nil {
		return nil, err
	}

	return &answer, nil
}

func (c *Client) setupDataChannel(dataChannel *webrtc.DataChannel) {
	if value := c.cfg.DataChannel.BufferThreshold; value != nil {
		dataChannel.SetBufferedAmountLowThreshold(*value)
	}

	dataChannel.OnError(func(err error) {
		fmt.Printf("datachannel %s error: %v\n", dataChannel.Label(), err)

		if cb := c.cfg.DataChannel.OnError; cb != nil {
			cb(err)
		}
	})

	dataChannel.OnOpen(func() {
		fmt.Printf("datachannel %v opened\n", dataChannel.Label())

		if cb := c.cfg.DataChannel.OnOpen; cb != nil {
			cb(dataChannel)
		}
	})

	dataChannel.OnMessage(func(msg webrtc.DataChannelMessage) {
		if cb := c.cfg.DataChannel.OnMessage; cb != nil {
			cb(msg)
		}
	})

	dataChannel.OnClose(func() {
		fmt.Printf("datachannel %v closed\n", dataChannel.Label())

		if cb := c.cfg.DataChannel.OnClose; cb != nil {
			cb()
		}
	})

	dataChannel.OnBufferedAmountLow(func() {
		if cb := c.cfg.DataChannel.OnBufferedAmountLow; cb != nil {
			cb(dataChannel)
		}
	})
}
