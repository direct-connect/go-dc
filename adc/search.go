package adc

import "strings"

func init() {
	RegisterMessage(SearchRequest{})
	RegisterMessage(SearchResult{})
}

const (
	searchAnd         = "AN"
	searchNot         = "NO"
	searchExt         = "EX"
	searchSizeLess    = "LE"
	searchSizeGreater = "GE"
	searchSizeEqual   = "EQ"
	searchToken       = "TO"
	searchType        = "TY"
	searchTTH         = "TR"
	searchTTHDepth    = "TD"
)

type FileType int

const (
	FileTypeAny  FileType = 0
	FileTypeFile FileType = 1
	FileTypeDir  FileType = 2
)

var _ Message = SearchRequest{}

type SearchRequest struct {
	Token string   `adc:"TO"`
	And   []string `adc:"AN"`
	Not   []string `adc:"NO"`
	Ext   []string `adc:"EX"`

	Le int64 `adc:"LE"`
	Ge int64 `adc:"GE"`
	Eq int64 `adc:"EQ"`

	Type FileType `adc:"TY"`

	// TIGR ext
	TTH *TTH `adc:"TR"`

	// SEGA ext
	Group ExtGroup `adc:"GR"`
	NoExt []string `adc:"RX"`
}

func (SearchRequest) Cmd() MsgType {
	return MsgType{'S', 'C', 'H'}
}

var _ Message = SearchResult{}

type SearchResult struct {
	Token string `adc:"TO"`
	Path  string `adc:"FN"`
	Size  int64  `adc:"SI"`
	Slots int    `adc:"SL"`

	// TIGR ext
	TTH *TTH `adc:"TR"`
}

func (SearchResult) Cmd() MsgType {
	return MsgType{'R', 'E', 'S'}
}

const (
	ExtNone  ExtGroup = 0x00
	ExtAudio ExtGroup = 0x01
	ExtArch  ExtGroup = 0x02
	ExtDoc   ExtGroup = 0x04
	ExtExe   ExtGroup = 0x08
	ExtImage ExtGroup = 0x10
	ExtVideo ExtGroup = 0x20
)

var extGroups = map[string]ExtGroup{
	// audio
	"ape": ExtAudio, "flac": ExtAudio, "m4a": ExtAudio,
	"mid": ExtAudio, "mp3": ExtAudio, "mpc": ExtAudio,
	"ogg": ExtAudio, "ra": ExtAudio, "wav": ExtAudio,
	"wma": ExtAudio,
	// compressed
	"7z": ExtArch, "ace": ExtArch, "arj": ExtArch,
	"bz2": ExtArch, "gz": ExtArch, "lha": ExtArch,
	"lzh": ExtArch, "rar": ExtArch, "tar": ExtArch,
	"tz": ExtArch, "z": ExtArch, "zip": ExtArch,
	// document
	"doc": ExtDoc, "docx": ExtDoc, "htm": ExtDoc,
	"html": ExtDoc, "nfo": ExtDoc, "odf": ExtDoc,
	"odp": ExtDoc, "ods": ExtDoc, "odt": ExtDoc,
	"pdf": ExtDoc, "ppt": ExtDoc, "pptx": ExtDoc,
	"rtf": ExtDoc, "txt": ExtDoc, "xls": ExtDoc,
	"xlsx": ExtDoc, "xml": ExtDoc, "xps": ExtDoc,
	// executable
	"app": ExtExe, "bat": ExtExe, "cmd": ExtExe,
	"com": ExtExe, "dll": ExtExe, "exe": ExtExe,
	"jar": ExtExe, "msi": ExtExe, "ps1": ExtExe,
	"vbs": ExtExe, "wsf": ExtExe,
	// picture
	"bmp": ExtImage, "cdr": ExtImage, "eps": ExtImage,
	"gif": ExtImage, "ico": ExtImage, "img": ExtImage,
	"jpeg": ExtImage, "jpg": ExtImage, "png": ExtImage,
	"ps": ExtImage, "psd": ExtImage, "sfw": ExtImage,
	"tga": ExtImage, "tif": ExtImage, "webp": ExtImage,
	// video
	"3gp": ExtVideo, "asf": ExtVideo, "asx": ExtVideo,
	"avi": ExtVideo, "divx": ExtVideo, "flv": ExtVideo,
	"mkv": ExtVideo, "mov": ExtVideo, "mp4": ExtVideo,
	"mpeg": ExtVideo, "mpg": ExtVideo, "ogm": ExtVideo,
	"pxp": ExtVideo, "qt": ExtVideo, "rm": ExtVideo,
	"rmvb": ExtVideo, "swf": ExtVideo, "vob": ExtVideo,
	"webm": ExtVideo, "wmv": ExtVideo,
}

type ExtGroup int

func (t ExtGroup) Has(st ExtGroup) bool { return t&st != 0 }

func (t ExtGroup) Matches(ext string) bool {
	if t == 0 {
		return true
	}
	if i := strings.LastIndex(ext, "."); i >= 0 {
		ext = ext[i+1:]
	}
	ext = strings.ToLower(ext)
	return t.Has(extGroups[ext])
}
