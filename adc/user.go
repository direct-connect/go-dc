package adc

import (
	"bytes"
	"strings"
)

func init() {
	// TODO: register UserInfoMod instead
	RegisterMessage(UserInfo{})
}

var (
	_ Marshaler   = UserInfoMod{}
	_ Unmarshaler = (*UserInfoMod)(nil)
)

type UserInfoMod Fields

func (UserInfoMod) Cmd() MsgType {
	return MsgType{'I', 'N', 'F'}
}

func (m UserInfoMod) MarshalADC(buf *bytes.Buffer) error {
	return Fields(m).MarshalADC(buf)
}

func (m *UserInfoMod) UnmarshalADC(data []byte) error {
	f := (*Fields)(m)
	return f.UnmarshalADC(data)
}

type UserType int

func (t UserType) Is(st UserType) bool { return t&st != 0 }

const (
	UserTypeNone       UserType = 0x00
	UserTypeBot        UserType = 0x01
	UserTypeRegistered UserType = 0x02
	UserTypeOperator   UserType = 0x04
	UserTypeSuperUser  UserType = 0x08
	UserTypeHubOwner   UserType = 0x10
	UserTypeHub        UserType = 0x20
	UserTypeHidden     UserType = 0x40
)

type AwayType int

const (
	AwayTypeNone     AwayType = 0
	AwayTypeNormal   AwayType = 1
	AwayTypeExtended AwayType = 2
)

type UserInfo struct {
	Id   CID    `adc:"ID"`
	Pid  *PID   `adc:"PD"` // sent only to hub
	Name string `adc:"NI,req"`

	Ip4  string `adc:"I4"`
	Ip6  string `adc:"I6"`
	Udp4 int    `adc:"U4"`
	Udp6 int    `adc:"U6"`

	ShareSize  int64 `adc:"SS,req"`
	ShareFiles int   `adc:"SF,req"`

	Version     string `adc:"VE,req"`
	Application string `adc:"AP"`

	MaxUpload   string `adc:"US"`
	MaxDownload string `adc:"DS"`

	Slots         int `adc:"SL,req"`
	SlotsFree     int `adc:"FS,req"`
	AutoSlotLimit int `adc:"AS"`

	Email string `adc:"EM"`
	Desc  string `adc:"DE"`

	HubsNormal     int `adc:"HN,req"`
	HubsRegistered int `adc:"HR,req"`
	HubsOperator   int `adc:"HO,req"`

	Token string `adc:"TO"` // C-C only

	Type UserType `adc:"CT"`
	Away AwayType `adc:"AW"`
	Ref  string   `adc:"RF"`

	Features ExtFeatures `adc:"SU,req"`

	KP string `adc:"KP"`

	// our extensions

	Address string `adc:"EA"`
}

func (UserInfo) Cmd() MsgType {
	return MsgType{'I', 'N', 'F'}
}

func (u *UserInfo) Normalize() {
	if u.Application == "" {
		if i := strings.LastIndex(u.Version, " "); i >= 0 {
			u.Application, u.Version = u.Version[:i], u.Version[i+1:]
		}
	}
}

type HubInfo struct {
	Name        string   `adc:"NI,req"`
	Version     string   `adc:"VE,req"`
	Application string   `adc:"AP"`
	Desc        string   `adc:"DE"`
	Type        UserType `adc:"CT"`

	// PING extension

	Address    string `adc:"HH"` // Hub Host address (ADC/ADCS URL address form)
	Website    string `adc:"WS"` // Hub Website
	Network    string `adc:"NE"` // Hub Network
	Owner      string `adc:"OW"` // Hub Owner name
	Users      int    `adc:"UC"` // Current user count, required
	Share      int    `adc:"SS"` // Total share size
	Files      int    `adc:"SF"` // Total files shared
	MinShare   int    `adc:"MS"` // Minimum share required to enter hub ( bytes )
	MaxShare   int64  `adc:"XS"` // Maximum share for entering hub ( bytes )
	MinSlots   int    `adc:"ML"` // Minimum slots required to enter hub
	MaxSlots   int    `adc:"XL"` // Maximum slots for entering hub
	UsersLimit int    `adc:"MC"` // Maximum possible clients ( users ) who can connect
	Uptime     int    `adc:"UP"` // Hub uptime (seconds)

	// ignored, doesn't matter in practice

	//int `adc:"MU"` // Minimum hubs connected where clients can be users
	//int `adc:"MR"` // Minimum hubs connected where client can be registered
	//int `adc:"MO"` // Minimum hubs connected where client can be operators
	//int `adc:"XU"` // Maximum hubs connected where clients can be users
	//int `adc:"XR"` // Maximum hubs connected where client can be registered
	//int `adc:"XO"` // Maximum hubs connected where client can be operators
}

func (HubInfo) Cmd() MsgType {
	// TODO: it's the same as User, so we won't register this one
	return MsgType{'I', 'N', 'F'}
}
