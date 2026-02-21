package models

import (
	"time"

	"github.com/google/uuid"
)

type User struct {
	Id          uuid.UUID
	Username    string
	DisplayName string `json:"display_name"`
	Password    string
	AvatarPath  *string
	IsAdmin     bool
	CreatedAt   time.Time
	UpdatedAt   time.Time
}

type Tag struct {
	Id        int
	Name      string
	CreatedAt time.Time
	UpdatedAt time.Time
}

// some fields contain json struct tag
// because we're querying it as JSONB
type Rice struct {
	Id          uuid.UUID
	AuthorId    uuid.UUID `json:"author_id"`
	Title       string
	Slug        string
	Description string
	CreatedAt   time.Time `json:"created_at"`
	UpdatedAt   time.Time `json:"updated_at"`
}

type RiceDotfiles struct {
	RiceId        uuid.UUID `json:"rice_id"`
	FilePath      string    `json:"file_path"`
	FileSize      int64     `json:"file_size"`
	DownloadCount uint      `json:"download_count"`
	CreatedAt     time.Time `json:"created_at"`
	UpdatedAt     time.Time `json:"updated_at"`
}

type RicePreview struct {
	Id        uuid.UUID
	RiceId    uuid.UUID `json:"rice_id"`
	FilePath  string    `json:"file_path"`
	CreatedAt time.Time `json:"created_at"`
}

type RiceComment struct {
	Id        uuid.UUID
	RiceId    uuid.UUID
	AuthorId  uuid.UUID
	Content   string
	CreatedAt time.Time
	UpdatedAt time.Time
}

type RiceCommentWithSlug struct {
	Id                 uuid.UUID
	RiceId             uuid.UUID
	AuthorId           uuid.UUID
	Content            string
	RiceSlug           string
	RiceAuthorUsername string
	CreatedAt          time.Time
	UpdatedAt          time.Time
}

type CommentWithUser struct {
	CommentId   uuid.UUID
	Content     string
	DisplayName string
	Username    string
	AvatarPath  *string
	CreatedAt   time.Time
	UpdatedAt   time.Time
}

type RiceWithRelations struct {
	Rice      Rice
	User      User
	Dotfiles  RiceDotfiles
	Previews  []RicePreview
	StarCount uint
	IsStarred bool
}

type PartialRice struct {
	Id            uuid.UUID
	Title         string
	Slug          string
	DisplayName   string
	Username      string
	Thumbnail     string
	StarCount     uint
	DownloadCount uint
	IsStarred     bool
	CreatedAt     time.Time
}

type ReportWithUser struct {
	Id          uuid.UUID
	ReporterId  uuid.UUID
	DisplayName string
	Username    string
	Reason      string
	RiceId      *uuid.UUID
	CommentId   *uuid.UUID
	IsClosed    bool
	CreatedAt   time.Time
}

type ServiceStatistics struct {
	UserCount       int
	User24hCount    int
	RiceCount       int
	Rice24hCount    int
	CommentCount    int
	Comment24hCount int
	ReportCount     int
	OpenReportCount int
}
