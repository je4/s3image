package media

import "io"

type ImageType interface {
	LoadImage(reader io.Reader) error
	StoreImage(format string) (io.Reader, *CoreMeta, error)
	Resize(options *ImageOptions) error
	Close()
}

type ResizeActionType string

const (
	ResizeActionTypeKeep           ResizeActionType = "keep"
	ResizeActionTypeStretch        ResizeActionType = "stretch"
	ResizeActionTypeCrop           ResizeActionType = "crop"
	ResizeActionTypeExtent         ResizeActionType = "extent"
	ResizeActionTypeBackgroundBlur ResizeActionType = "backgroundblur"
)

type ImageOptions struct {
	Width, Height                       int64
	ActionType                          ResizeActionType
	TargetFormat                        string
	OverlayCollection, OverlaySignature string
	BackgroundColor                     string
}
