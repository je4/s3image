package media

func CalcSizeMin(origWidth, origHeight, Width, Height int64) (width int64, height int64) {
	//    oW    W
	//    -- = --
	//    oH    H
	origAspect := float64(origWidth) / float64(origHeight)
	newAspect := float64(Width) / float64(Height)

	if origAspect < newAspect {
		height = Height
		width = (Height * origWidth) / origHeight
	} else {
		width = Width
		height = (Width * origHeight) / origWidth
	}
	return
}

func CalcSizeMax(origWidth, origHeight, Width, Height int64) (width int64, height int64) {
	//    oW    W
	//    -- = --
	//    oH    H
	origAspect := float64(origWidth) / float64(origHeight)
	newAspect := float64(Width) / float64(Height)

	if origAspect > newAspect {
		height = Height
		width = (Height * origWidth) / origHeight
	} else {
		width = Width
		height = (Width * origHeight) / origWidth
	}
	return
}
