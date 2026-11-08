package metadata

import "time"

// Video is the metadata adapter's representation of a video. It deliberately
// does not depend on the root core package to avoid an import cycle; the
// orchestration layer maps it into core.VideoData.
type Video struct {
	ID          string
	Title       string
	Channel     string
	Description string
	PublishedAt time.Time
	Duration    string // formatted as "mm:ss" or "hh:mm:ss"
	ViewCount   uint64
	LikeCount   uint64
}
