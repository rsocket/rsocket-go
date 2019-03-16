package extension

// MIME is MIME types in number.
// Please see: https://github.com/rsocket/rsocket/blob/master/Extensions/WellKnownMimeTypes.md
type MIME int8

func (p MIME) String() string {
	return mimeTypes[p]
}

var (
	mimeTypes  map[MIME]string
	mimeTypesR map[string]MIME
)

// ParseMIME parse a string to MIME.
func ParseMIME(str string) (mime MIME, ok bool) {
	mime, ok = mimeTypesR[str]
	if !ok {
		mime = -1
	}
	return
}

// All MIMEs
const (
	ApplicationAvro MIME = iota
	ApplicationCbor
	ApplicationGraphql
	ApplicationGzip
	ApplicationJavascript
	ApplicationJson
	ApplicationOctetStream
	ApplicationPdf
	ApplicationThrift
	ApplicationProtobuf
	ApplicationXml
	ApplicationZip
	AudioAac
	AudioMp3
	AudioMp4
	AudioMpeg3
	AudioMpeg
	AudioOgg
	AudioOpus
	AudioVorbis
	ImageBmp
	ImageGif
	ImageHeicSequence
	ImageHeic
	ImageHeifSequence
	ImageHeif
	ImageJpeg
	ImagePng
	ImageTiff
	MultipartMixed
	TextCss
	TextCsv
	TextHtml
	TextPlain
	TextXml
	VideoH264
	VideoH265
	VideoVP8
	MessageZipkin
	MessageRouting
	MessageCompositeMetadata
)

func init() {
	mimeTypes = map[MIME]string{
		ApplicationAvro:          "application/avro",
		ApplicationCbor:          "application/cbor",
		ApplicationGraphql:       "application/graphql",
		ApplicationGzip:          "application/gzip",
		ApplicationJavascript:    "application/javascript",
		ApplicationJson:          "application/json",
		ApplicationOctetStream:   "application/octet-stream",
		ApplicationPdf:           "application/pdf",
		ApplicationThrift:        "application/vnd.apache.thrift.binary",
		ApplicationProtobuf:      "application/vnd.google.protobuf",
		ApplicationXml:           "application/xml",
		ApplicationZip:           "application/zip",
		AudioAac:                 "audio/aac",
		AudioMp3:                 "audio/mp3",
		AudioMp4:                 "audio/mp4",
		AudioMpeg3:               "audio/mpeg3",
		AudioMpeg:                "audio/mpeg",
		AudioOgg:                 "audio/ogg",
		AudioOpus:                "audio/opus",
		AudioVorbis:              "audio/vorbis",
		ImageBmp:                 "image/bmp",
		ImageGif:                 "image/gif",
		ImageHeicSequence:        "image/heic-sequence",
		ImageHeic:                "image/heic",
		ImageHeifSequence:        "image/heif-sequence",
		ImageHeif:                "image/heif",
		ImageJpeg:                "image/jpeg",
		ImagePng:                 "image/png",
		ImageTiff:                "image/tiff",
		MultipartMixed:           "multipart/mixed",
		TextCss:                  "text/css",
		TextCsv:                  "text/csv",
		TextHtml:                 "text/html",
		TextPlain:                "text/plain",
		TextXml:                  "text/xml",
		VideoH264:                "video/H264",
		VideoH265:                "video/H265",
		VideoVP8:                 "video/VP8",
		MessageZipkin:            "message/x.rsocket.tracing-zipkin.v0",
		MessageRouting:           "message/x.rsocket.routing.v0",
		MessageCompositeMetadata: "message/x.rsocket.composite-metadata.v0",
	}
	mimeTypesR = make(map[string]MIME, len(mimeTypes))
	for k, v := range mimeTypes {
		mimeTypesR[v] = k
	}
}
