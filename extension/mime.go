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
	ApplicationCBOR
	ApplicationGraphql
	ApplicationGzip
	ApplicationJavascript
	ApplicationJSON
	ApplicationOctetStream
	ApplicationPDF
	ApplicationThrift
	ApplicationProtobuf
	ApplicationXML
	ApplicationZip
	AudioAAC
	AudioMP3
	AudioMP4
	AudioMPEG3
	AudioMPEG
	AudioOGG
	AudioOpus
	AudioVorbis
	ImageBMP
	ImageGIF
	ImageHEICSequence
	ImageHEIC
	ImageHEIFSequence
	ImageHEIF
	ImageJPEG
	ImagePNG
	ImageTIFF
	MultipartMixed
	TextCSS
	TextCSV
	TextHTML
	TextPlain
	TextXML
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
		ApplicationCBOR:          "application/cbor",
		ApplicationGraphql:       "application/graphql",
		ApplicationGzip:          "application/gzip",
		ApplicationJavascript:    "application/javascript",
		ApplicationJSON:          "application/json",
		ApplicationOctetStream:   "application/octet-stream",
		ApplicationPDF:           "application/pdf",
		ApplicationThrift:        "application/vnd.apache.thrift.binary",
		ApplicationProtobuf:      "application/vnd.google.protobuf",
		ApplicationXML:           "application/xml",
		ApplicationZip:           "application/zip",
		AudioAAC:                 "audio/aac",
		AudioMP3:                 "audio/mp3",
		AudioMP4:                 "audio/mp4",
		AudioMPEG3:               "audio/mpeg3",
		AudioMPEG:                "audio/mpeg",
		AudioOGG:                 "audio/ogg",
		AudioOpus:                "audio/opus",
		AudioVorbis:              "audio/vorbis",
		ImageBMP:                 "image/bmp",
		ImageGIF:                 "image/gif",
		ImageHEICSequence:        "image/heic-sequence",
		ImageHEIC:                "image/heic",
		ImageHEIFSequence:        "image/heif-sequence",
		ImageHEIF:                "image/heif",
		ImageJPEG:                "image/jpeg",
		ImagePNG:                 "image/png",
		ImageTIFF:                "image/tiff",
		MultipartMixed:           "multipart/mixed",
		TextCSS:                  "text/css",
		TextCSV:                  "text/csv",
		TextHTML:                 "text/html",
		TextPlain:                "text/plain",
		TextXML:                  "text/xml",
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
