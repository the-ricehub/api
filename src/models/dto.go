package models

import (
	"ricehub/src/utils"
	"time"

	"github.com/google/uuid"
)

// Helpers
func getUserAvatar(avatarPath *string) string {
	avatar := utils.Config.CDNUrl + utils.Config.DefaultAvatar
	if avatarPath != nil {
		avatar = utils.Config.CDNUrl + *avatarPath
	}
	return avatar
}

// Requests
// AUTH
type RegisterDTO struct {
	Username    string `json:"username" binding:"required,min=4,max=14,alphanum"`
	DisplayName string `json:"displayName" binding:"required,min=3,max=20,displayname"`
	Password    string `json:"password" binding:"required,min=6,max=512"`
}

type LoginDTO struct {
	Username string `json:"username" binding:"required"`
	Password string `json:"password" binding:"required"`
}

// USERS
type UpdateDisplayNameDTO struct {
	DisplayName string `json:"displayName" binding:"required,min=3,max=20,displayname"`
}

type UpdatePasswordDTO struct {
	OldPassword string `json:"oldPassword" binding:"required"`
	NewPassword string `json:"newPassword" binding:"required,min=6,max=256"`
}

type DeleteUserDTO struct {
	Password string `json:"password" binding:"required"`
}

type BanUserDTO struct {
	Reason   string  `json:"reason" binding:"required,min=6,max=1024"`
	Duration *string `json:"duration" binding:"omitempty"`
}

type UpdateUserBanDTO struct {
	Reason   *string `json:"reason" binding:"omitempty,min=6,max=1024"`
	Duration *string `json:"duration" binding:"omitempty"`
}

// TAGS
type TagNameDTO struct {
	Name string `json:"name" binding:"required,min=2,max=16,alpha,ascii"`
}

// RICES
type CreateRiceDTO struct {
	Title       string `form:"title" binding:"required,min=4,max=32,ricetitle"`
	Description string `form:"description" binding:"required,min=4,max=10240"`
}

type UpdateRiceDTO struct {
	Title       *string `json:"title" binding:"omitempty,min=4,max=32,ricetitle"`
	Description *string `json:"description" binding:"omitempty,min=4,max=10240"`
}

// COMMENTS
type AddCommentDTO struct {
	RiceID  string `json:"riceId" binding:"required,uuid"`
	Content string `json:"content" binding:"required,min=8,max=128"`
}

type UpdateCommentDTO struct {
	Content string `json:"content" binding:"required,min=8,max=128"`
}

// REPORTS
type CreateReportDTO struct {
	Reason    string  `json:"reason" binding:"required,min=8,max=1024"`
	RiceID    *string `json:"riceId" binding:"omitempty,uuid"`
	CommentID *string `json:"commentId" binding:"omitempty,uuid"`
}

// Responses
type UserDTO struct {
	ID          uuid.UUID `json:"id"`
	Username    string    `json:"username"`
	DisplayName string    `json:"displayName"`
	AvatarUrl   string    `json:"avatarUrl"`
	IsAdmin     bool      `json:"isAdmin"`
	CreatedAt   time.Time `json:"createdAt"`
	UpdatedAt   time.Time `json:"updatedAt"`
}

func (u User) ToDTO() UserDTO {
	return UserDTO{
		ID:          u.ID,
		Username:    u.Username,
		DisplayName: u.DisplayName,
		AvatarUrl:   getUserAvatar(u.AvatarPath),
		IsAdmin:     u.IsAdmin,
		CreatedAt:   u.CreatedAt,
		UpdatedAt:   u.UpdatedAt,
	}
}

func UsersToDTOs(users []User) []UserDTO {
	dtos := make([]UserDTO, len(users))
	for i, u := range users {
		dtos[i] = u.ToDTO()
	}
	return dtos
}

type TagDTO struct {
	ID   int    `json:"id"`
	Name string `json:"name"`
}

func (t Tag) ToDTO() TagDTO {
	return TagDTO{
		ID:   t.ID,
		Name: t.Name,
	}
}

type RiceDotfilesDTO struct {
	FilePath  string    `json:"filePath"`
	FileSize  int64     `json:"fileSize"`
	CreatedAt time.Time `json:"createdAt"`
	UpdatedAt time.Time `json:"updatedAt"`
}

func (df RiceDotfiles) ToDTO() RiceDotfilesDTO {
	return RiceDotfilesDTO{
		FilePath:  utils.Config.CDNUrl + df.FilePath,
		FileSize:  df.FileSize,
		CreatedAt: df.CreatedAt,
		UpdatedAt: df.UpdatedAt,
	}
}

type RiceDTO struct {
	ID          uuid.UUID       `json:"id"`
	Title       string          `json:"title"`
	Slug        string          `json:"slug"`
	Description string          `json:"description"`
	Downloads   uint            `json:"downloads"`
	Stars       uint            `json:"stars"`
	Previews    []string        `json:"previews"`
	Dotfiles    RiceDotfilesDTO `json:"dotfiles"`
	CreatedAt   time.Time       `json:"createdAt"`
	UpdatedAt   time.Time       `json:"updatedAt"`
}

func (r Rice) ToDTO() RiceDTO {
	return RiceDTO{
		ID:          r.ID,
		Title:       r.Title,
		Slug:        r.Slug,
		Description: r.Description,
		Downloads:   0,
		Stars:       0,
		Previews:    []string{},
		Dotfiles:    RiceDotfilesDTO{},
		CreatedAt:   r.CreatedAt,
		UpdatedAt:   r.UpdatedAt,
	}
}

type RicePreviewDTO struct {
	ID  uuid.UUID `json:"id"`
	Url string    `json:"url"`
}

func (p RicePreview) ToDTO() RicePreviewDTO {
	return RicePreviewDTO{
		ID:  p.ID,
		Url: utils.Config.CDNUrl + p.FilePath,
	}
}

type RiceWithRelationsDTO struct {
	ID          uuid.UUID        `json:"id"`
	Title       string           `json:"title"`
	Slug        string           `json:"slug"`
	Description string           `json:"description"`
	Downloads   uint             `json:"downloads"`
	Stars       uint             `json:"stars"`
	IsStarred   bool             `json:"isStarred"`
	Previews    []RicePreviewDTO `json:"previews"`
	Dotfiles    RiceDotfilesDTO  `json:"dotfiles"`
	Author      UserDTO          `json:"author"`
	CreatedAt   time.Time        `json:"createdAt"`
	UpdatedAt   time.Time        `json:"updatedAt"`
}

func (r RiceWithRelations) ToDTO() RiceWithRelationsDTO {
	previews := make([]RicePreviewDTO, len(r.Previews))
	for i, preview := range r.Previews {
		previews[i] = preview.ToDTO()
	}

	return RiceWithRelationsDTO{
		ID:          r.Rice.ID,
		Title:       r.Rice.Title,
		Slug:        r.Rice.Slug,
		Description: r.Rice.Description,
		Downloads:   r.Dotfiles.DownloadCount,
		Stars:       r.StarCount,
		IsStarred:   r.IsStarred,
		Previews:    previews,
		Dotfiles:    r.Dotfiles.ToDTO(),
		Author:      r.User.ToDTO(),
		CreatedAt:   r.Rice.CreatedAt,
		UpdatedAt:   r.Rice.UpdatedAt,
	}
}

type RiceCommentDTO struct {
	ID        uuid.UUID `json:"id"`
	RiceID    uuid.UUID `json:"riceId"`
	Content   string    `json:"content"`
	CreatedAt time.Time `json:"createdAt"`
	UpdatedAt time.Time `json:"updatedAt"`
}

func (c RiceComment) ToDTO() RiceCommentDTO {
	return RiceCommentDTO{
		ID:        c.ID,
		RiceID:    c.RiceID,
		Content:   c.Content,
		CreatedAt: c.CreatedAt,
		UpdatedAt: c.UpdatedAt,
	}
}

type RiceCommentWithSlugDTO struct {
	ID                 uuid.UUID `json:"id"`
	RiceID             uuid.UUID `json:"riceId"`
	AuthorID           uuid.UUID `json:"authorId"`
	Content            string    `json:"content"`
	RiceSlug           string    `json:"riceSlug"`
	RiceAuthorUsername string    `json:"riceAuthorUsername"`
	CreatedAt          time.Time `json:"createdAt"`
	UpdatedAt          time.Time `json:"updatedAt"`
}

func (c RiceCommentWithSlug) ToDTO() RiceCommentWithSlugDTO {
	return RiceCommentWithSlugDTO{
		ID:                 c.ID,
		RiceID:             c.RiceID,
		AuthorID:           c.AuthorID,
		Content:            c.Content,
		RiceSlug:           c.RiceSlug,
		RiceAuthorUsername: c.RiceAuthorUsername,
		CreatedAt:          c.CreatedAt,
		UpdatedAt:          c.UpdatedAt,
	}
}

type CommentWithUserDTO struct {
	CommentID   uuid.UUID `json:"commentId"`
	Content     string    `json:"content"`
	DisplayName string    `json:"displayName"`
	Username    string    `json:"username"`
	Avatar      string    `json:"avatar"`
	CreatedAt   time.Time `json:"createdAt"`
	UpdatedAt   time.Time `json:"updatedAt"`
}

func (c CommentWithUser) ToDTO() CommentWithUserDTO {
	return CommentWithUserDTO{
		CommentID:   c.CommentID,
		Content:     c.Content,
		DisplayName: c.DisplayName,
		Username:    c.Username,
		Avatar:      getUserAvatar(c.AvatarPath),
		CreatedAt:   c.CreatedAt,
		UpdatedAt:   c.UpdatedAt,
	}
}

func CommentsWithUserToDTOs(comments []CommentWithUser) []CommentWithUserDTO {
	dtos := make([]CommentWithUserDTO, len(comments))
	for i, c := range comments {
		dtos[i] = c.ToDTO()
	}
	return dtos
}

type PartialRiceDTO struct {
	ID          uuid.UUID `json:"id"`
	Title       string    `json:"title"`
	Slug        string    `json:"slug"`
	DisplayName string    `json:"displayName"`
	Username    string    `json:"username"`
	Thumbnail   string    `json:"thumbnail"`
	Stars       uint      `json:"stars"`
	Downloads   uint      `json:"downloads"`
	IsStarred   bool      `json:"isStarred"`
	CreatedAt   time.Time `json:"createdAt"`
}

func (r PartialRice) ToDTO() PartialRiceDTO {
	return PartialRiceDTO{
		ID:          r.ID,
		Title:       r.Title,
		Slug:        r.Slug,
		DisplayName: r.DisplayName,
		Username:    r.Username,
		Thumbnail:   utils.Config.CDNUrl + r.Thumbnail,
		Stars:       r.StarCount,
		Downloads:   r.DownloadCount,
		IsStarred:   r.IsStarred,
		CreatedAt:   r.CreatedAt,
	}
}

func PartialRicesToDTOs(rices []PartialRice) []PartialRiceDTO {
	dtos := make([]PartialRiceDTO, len(rices))
	for i, r := range rices {
		dtos[i] = r.ToDTO()
	}
	return dtos
}

type ReportWithUserDTO struct {
	ID          uuid.UUID  `json:"id"`
	ReporterID  uuid.UUID  `json:"reporterId"`
	DisplayName string     `json:"displayName"`
	Username    string     `json:"username"`
	Reason      string     `json:"reason"`
	RiceID      *uuid.UUID `json:"riceId,omitempty"`
	CommentID   *uuid.UUID `json:"commentId,omitempty"`
	IsClosed    bool       `json:"isClosed"`
	CreatedAt   time.Time  `json:"createdAt"`
}

func (r ReportWithUser) ToDTO() ReportWithUserDTO {
	return ReportWithUserDTO(r)
}

func ReportsToDTO(reports []ReportWithUser) []ReportWithUserDTO {
	dto := make([]ReportWithUserDTO, len(reports))
	for i, report := range reports {
		dto[i] = report.ToDTO()
	}
	return dto
}

type ServiceStatisticsDTO struct {
	UserCount       int `json:"userCount"`
	User24hCount    int `json:"user24hCount"`
	RiceCount       int `json:"riceCount"`
	Rice24hCount    int `json:"rice24hCount"`
	CommentCount    int `json:"commentCount"`
	Comment24hCount int `json:"comment24hCount"`
	ReportCount     int `json:"reportCount"`
	OpenReportCount int `json:"openReportCount"`
}

func (s ServiceStatistics) ToDTO() ServiceStatisticsDTO {
	return ServiceStatisticsDTO(s)
}

type WebsiteVariableDTO struct {
	Value     string    `json:"value"`
	UpdatedAt time.Time `json:"updatedAt"`
}

func (v WebsiteVariable) ToDTO() WebsiteVariableDTO {
	return WebsiteVariableDTO{
		Value:     v.Value,
		UpdatedAt: v.UpdatedAt,
	}
}

type LinkDTO struct {
	URL string `json:"url"`
}

func (link Link) ToDTO() LinkDTO {
	return LinkDTO{
		URL: link.URL,
	}
}

type UserBanDTO struct {
	ID        uuid.UUID  `json:"id"`
	UserID    uuid.UUID  `json:"userId"`
	AdminID   uuid.UUID  `json:"adminId"`
	Reason    string     `json:"reason"`
	IsRevoked bool       `json:"isRevoked"`
	ExpiresAt *time.Time `json:"expiresAt,omitempty"`
	BannedAt  time.Time  `json:"bannedAt"`
	RevokedAt *time.Time `json:"revokedAt,omitempty"`
}

func (b UserBan) ToDTO() UserBanDTO {
	return UserBanDTO(b)
}
