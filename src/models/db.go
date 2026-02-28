package models

import (
	"time"

	"github.com/google/uuid"
)

type User struct {
	ID          uuid.UUID
	Username    string
	DisplayName string `json:"display_name"`
	Password    string
	AvatarPath  *string   `json:"avatar_path"`
	IsAdmin     bool      `json:"is_admin"`
	IsBanned    bool      `json:"is_banned"`
	CreatedAt   time.Time `json:"created_at"`
	UpdatedAt   time.Time `json:"updated_at"`
}

type UserWithBan struct {
	User User
	Ban  UserBan
}

type Tag struct {
	ID        int
	Name      string
	CreatedAt time.Time
	UpdatedAt time.Time
}

// some fields contain json struct tag
// because we're querying it as JSONB
type Rice struct {
	ID          uuid.UUID
	AuthorID    uuid.UUID `json:"author_id"`
	Title       string
	Slug        string
	Description string
	CreatedAt   time.Time `json:"created_at"`
	UpdatedAt   time.Time `json:"updated_at"`
}

type RiceDotfiles struct {
	RiceID        uuid.UUID `json:"rice_id"`
	FilePath      string    `json:"file_path"`
	FileSize      int64     `json:"file_size"`
	DownloadCount uint      `json:"download_count"`
	CreatedAt     time.Time `json:"created_at"`
	UpdatedAt     time.Time `json:"updated_at"`
}

type RicePreview struct {
	ID        uuid.UUID
	RiceID    uuid.UUID `json:"rice_id"`
	FilePath  string    `json:"file_path"`
	CreatedAt time.Time `json:"created_at"`
}

type RiceComment struct {
	ID        uuid.UUID
	RiceID    uuid.UUID
	AuthorID  uuid.UUID
	Content   string
	CreatedAt time.Time
	UpdatedAt time.Time
}

type RiceCommentWithSlug struct {
	ID                 uuid.UUID
	RiceID             uuid.UUID
	AuthorID           uuid.UUID
	Content            string
	RiceSlug           string
	RiceAuthorUsername string
	CreatedAt          time.Time
	UpdatedAt          time.Time
}

type CommentWithUser struct {
	CommentID   uuid.UUID
	Content     string
	DisplayName string
	Username    string
	AvatarPath  *string
	IsBanned    bool
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
	ID            uuid.UUID
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
	ID          uuid.UUID
	ReporterID  uuid.UUID
	DisplayName string
	Username    string
	Reason      string
	RiceID      *uuid.UUID
	CommentID   *uuid.UUID
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

type WebsiteVariable struct {
	Key       string
	Value     string
	CreatedAt time.Time
	UpdatedAt time.Time
}

type Link struct {
	Name      string
	URL       string
	CreatedAt time.Time
	UpdatedAt time.Time
}

type UserState struct {
	UserExists bool
	UserBanned bool
}

type UserBan struct {
	ID        uuid.UUID
	UserID    uuid.UUID `json:"user_id"`
	AdminID   uuid.UUID `json:"admin_id"`
	Reason    string
	IsRevoked bool       `json:"is_revoked"`
	ExpiresAt *time.Time `json:"expires_at"`
	BannedAt  time.Time  `json:"banned_at"`
	RevokedAt *time.Time `json:"revoked_at"`
}
