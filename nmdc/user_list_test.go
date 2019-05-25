package nmdc

import (
	"testing"
)

var userListCases = []casesMessageEntry{
	{
		typ:  "UserIP",
		data: `john doe 192.168.1.2$$user 2 192.168.1.3$$`,
		msg: &UserIP{
			List: []UserAddress{
				{
					Name: "john doe",
					IP:   "192.168.1.2",
				},
				{
					Name: "user 2",
					IP:   "192.168.1.3",
				},
			},
		},
	},
}

func TestUserListUnmarshal(t *testing.T) {
	doMessageTestUnmarshal(t, userListCases)
}

func TestUserListMarshal(t *testing.T) {
	doMessageTestMarshal(t, userListCases)
}

func BenchmarkUserListUnmarshal(b *testing.B) {
	doMessageBenchmarkUnmarshal(b, userListCases)
}

func BenchmarkUserListMarshal(b *testing.B) {
	doMessageBenchmarkMarshal(b, userListCases)
}
