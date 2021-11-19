package media

import "io"

type ImageType interface {
	LoadImage(reader io.Reader) error
	StoreImage(format string) (io.Reader, *CoreMeta, error)
	Resize(options *ImageOptions) error
	Close()
}

type ImageOptions struct {
	Width, Height                       int64
	ActionType                          string
	TargetFormat                        string
	OverlayCollection, OverlaySignature string
	BackgroundColor                     string
}
