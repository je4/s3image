package media

import (
	"bytes"
	"github.com/pkg/errors"
	"gopkg.in/gographics/imagick.v3/imagick"
	"io"
	"math"
)

type ImageMagickV3 struct {
	mw     *imagick.MagickWand
	frames int64
}

func NewImageMagickV3(reader io.Reader) (*ImageMagickV3, error) {
	im := &ImageMagickV3{mw: imagick.NewMagickWand()}
	if err := im.LoadImage(reader); err != nil {
		return nil, err
	}
	return im, nil
}

func (im *ImageMagickV3) Close() {
	im.mw.Destroy()
}

func (im *ImageMagickV3) LoadImage(reader io.Reader) error {
	var buf bytes.Buffer
	if _, err := buf.ReadFrom(reader); err != nil {
		return errors.Wrapf(err, "cannot read raw image blob")
	}
	if err := im.mw.ReadImageBlob(buf.Bytes()); err != nil {
		return errors.Wrapf(err, "cannot read image from blob")
	}
	return nil
}

func (im *ImageMagickV3) StoreImage(format string) (io.Reader, *CoreMeta, error) {
	var buf *bytes.Reader
	if err := im.mw.SetFormat(format); err != nil {
		return nil, nil, errors.Wrapf(err, "cannot set format %s", format)
	}

	if im.frames > 1 {
		buf = bytes.NewReader(im.mw.GetImagesBlob())
	} else {
		buf = bytes.NewReader(im.mw.GetImageBlob())
	}

	cm := &CoreMeta{
		Width:    int64(im.mw.GetImageWidth()),
		Height:   int64(im.mw.GetImageHeight()),
		Duration: 0,
		Format:   im.mw.GetFormat(),
		Mimetype: "application/octet-stream",
		Size:     buf.Size(),
	}
	return buf, cm, nil
}

func (im *ImageMagickV3) Resize(options *ImageOptions) error {
	im.mw.ResetIterator()
	im.frames = 0
	for im.mw.NextImage() {
		im.frames++
		//
		// calculate missing size parameter
		//
		if options.Width == 0 && options.Height == 0 {
			options.Width = int64(im.mw.GetImageWidth())
			options.Height = int64(im.mw.GetImageHeight())
		}
		if options.Width == 0 {
			options.Width = int64(math.Round(float64(options.Height) * float64(im.mw.GetImageWidth()) / float64(im.mw.GetImageHeight())))
		}
		if options.Height == 0 {
			options.Height = int64(math.Round(float64(options.Width) * float64(im.mw.GetImageHeight()) / float64(im.mw.GetImageWidth())))
		}

		if err := im.mw.AutoOrientImage(); err != nil {
			return errors.Wrapf(err, "cannot auto orient image")
		}

		switch options.ActionType {
		case "keep":
			nw, nh := CalcSizeMin(int64(im.mw.GetImageWidth()), int64(im.mw.GetImageHeight()), options.Width, options.Height)
			if err := im.mw.ResizeImage(uint(nw), uint(nh), imagick.FILTER_LANCZOS); err != nil {
				return errors.Wrapf(err, "cannot resizeimage(%v, %v)", uint(nw), uint(nh))
			}
		case "stretch":
			if err := im.mw.ResizeImage(uint(options.Width), uint(options.Height), imagick.FILTER_LANCZOS); err != nil {
				return errors.Wrapf(err, "cannot resizeimage(%v, %v)", uint(options.Width), uint(options.Height))
			}
		case "crop":
			nw, nh := CalcSizeMax(int64(im.mw.GetImageWidth()), int64(im.mw.GetImageHeight()), options.Width, options.Height)
			if err := im.mw.ResizeImage(uint(nw), uint(nh), imagick.FILTER_LANCZOS); err != nil {
				return errors.Wrapf(err, "cannot resizeimage(%v, %v)", uint(nw), uint(nh))
			}
			x := (options.Width - nw) / 2
			y := (options.Height - nh) / 2
			if err := im.mw.CropImage(uint(options.Width), uint(options.Height), int(x), int(y)); err != nil {
				return errors.Wrapf(err, "cannot cropimage(%v, %v, %v, %v", uint(options.Width), uint(options.Height), int(x), int(y))
			}
		case "extent":
			nw, nh := CalcSizeMin(int64(im.mw.GetImageWidth()), int64(im.mw.GetImageHeight()), int64(options.Width), int64(options.Height))
			if err := im.mw.ResizeImage(uint(nw), uint(nh), imagick.FILTER_LANCZOS); err != nil {
				return errors.Wrapf(err, "cannot resizeimage(%v, %v)", uint(nw), uint(nh))
			}
			im.mw.SetGravity(imagick.GRAVITY_CENTER)
			pw := imagick.NewPixelWand()
			defer pw.Destroy()
			pw.SetColor(options.BackgroundColor)
			im.mw.SetImageBackgroundColor(pw)
			w := uint(options.Width)
			h := uint(options.Height)
			x := (int(options.Width) - int(nw)) / 2
			y := (int(options.Height) - int(nh)) / 2
			if err := im.mw.ExtentImage(w, h, -x, -y); err != nil {
				return errors.Wrapf(err, "cannot extentimage(%v, %v, %v, %v)", w, h, x, y)
			}
		case "backgroundblur":
			foreground := im.mw.Clone()
			defer foreground.Destroy()
			nw, nh := CalcSizeMin(int64(im.mw.GetImageWidth()), int64(im.mw.GetImageHeight()), int64(options.Width), int64(options.Height))
			if err := foreground.ResizeImage(uint(nw), uint(nh), imagick.FILTER_LANCZOS); err != nil {
				return errors.Wrapf(err, "cannot resizeimage(%v, %v) - foreground", uint(nw), uint(nh))
			}

			if err := im.mw.ResizeImage(uint(options.Width), uint(options.Height), imagick.FILTER_LANCZOS); err != nil {
				return errors.Wrapf(err, "cannot resizeimage(%v, %v)", uint(options.Width), uint(options.Height))
			}

			if err := im.mw.BlurImage(20, 30.0); err != nil {
				return errors.Wrapf(err, "cannot blurimage(%v, %v)", 20.0, 30.0)
			}

			if err := im.mw.CompositeImageGravity(foreground, imagick.COMPOSITE_OP_COPY, imagick.GRAVITY_CENTER); err != nil {
				return errors.Wrapf(err, "cannot composite images")
			}
		}
	}
	return nil
}
