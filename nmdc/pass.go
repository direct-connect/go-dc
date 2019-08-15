package nmdc

func init() {
	RegisterMessage(&MyPass{})
	RegisterMessage(&BadPass{})
	RegisterMessage(&GetPass{})
	RegisterMessage(&LogedIn{})
}

type MyPass struct {
	String
}

func (*MyPass) Type() string {
	return "MyPass"
}

type BadPass struct {
	NoArgs
}

func (*BadPass) Type() string {
	return "BadPass"
}

type GetPass struct {
	NoArgs
}

func (*GetPass) Type() string {
	return "GetPass"
}

type LogedIn struct {
	Name
}

func (*LogedIn) Type() string {
	return "LogedIn"
}
