package models

import (
	"ricehub/src/utils"
	"time"

	"github.com/google/uuid"
)

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

// TAGS
type TagNameDTO struct {
	Name string `json:"name" binding:"required,min=2,max=16,alpha,ascii"`
}

// RICES
type CreateRiceDTO struct {
	Title       string `form:"title" binding:"required,min=4,max=32,ricetitle"`
	Description string `form:"description" binding:"required,min=4,max=1024"`
}

type UpdateRiceDTO struct {
	Title       *string `json:"title" binding:"omitempty,min=4,max=32,ricetitle"`
	Description *string `json:"description" binding:"omitempty,min=4,max=1024"`
}

// COMMENTS
type AddCommentDTO struct {
	RiceId  string `json:"riceId" binding:"required,uuid"`
	Content string `json:"content" binding:"required,min=8,max=128"`
}

type UpdateCommentDTO struct {
	Content string `json:"content" binding:"required,min=8,max=128"`
}

// REPORTS
type CreateReportDTO struct {
	Reason    string  `json:"reason" binding:"required,min=8,max=1024"`
	RiceId    *string `json:"riceId" binding:"omitempty,uuid"`
	CommentId *string `json:"commentId" binding:"omitempty,uuid"`
}

// Responses
type UserDTO struct {
	Id          uuid.UUID `json:"id"`
	Username    string    `json:"username"`
	DisplayName string    `json:"displayName"`
	AvatarUrl   string    `json:"avatarUrl"`
	CreatedAt   time.Time `json:"createdAt"`
	UpdatedAt   time.Time `json:"updatedAt"`
}

func (u User) ToDTO() UserDTO {
	avatar := utils.Config.CDNUrl + utils.Config.DefaultAvatar
	if u.AvatarPath != nil {
		avatar = utils.Config.CDNUrl + *u.AvatarPath
	}

	return UserDTO{
		Id:          u.Id,
		Username:    u.Username,
		DisplayName: u.DisplayName,
		AvatarUrl:   avatar,
		CreatedAt:   u.CreatedAt,
		UpdatedAt:   u.UpdatedAt,
	}
}

type TagDTO struct {
	Id   int    `json:"id"`
	Name string `json:"name"`
}

func (t Tag) ToDTO() TagDTO {
	return TagDTO{
		Id:   t.Id,
		Name: t.Name,
	}
}

type RiceDotfilesDTO struct {
	FilePath  string    `json:"filePath"`
	CreatedAt time.Time `json:"createdAt"`
	UpdatedAt time.Time `json:"updatedAt"`
}

func (df RiceDotfiles) ToDTO() RiceDotfilesDTO {
	return RiceDotfilesDTO{
		FilePath:  utils.Config.CDNUrl + df.FilePath,
		CreatedAt: df.CreatedAt,
		UpdatedAt: df.UpdatedAt,
	}
}

type RiceDTO struct {
	Id          uuid.UUID       `json:"id"`
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
		Id:          r.Id,
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

func (r RiceWithRelations) ToDTO() RiceDTO {
	previews := make([]string, len(r.Previews))
	for i, preview := range r.Previews {
		previews[i] = utils.Config.CDNUrl + preview.FilePath
	}

	return RiceDTO{
		Id:          r.Rice.Id,
		Title:       r.Rice.Title,
		Slug:        r.Rice.Slug,
		Description: r.Rice.Description,
		Downloads:   r.Dotfiles.DownloadCount,
		Stars:       r.StarCount,
		Previews:    previews,
		Dotfiles:    r.Dotfiles.ToDTO(),
		CreatedAt:   r.Rice.CreatedAt,
		UpdatedAt:   r.Rice.UpdatedAt,
	}
}

type RiceCommentDTO struct {
	Id        uuid.UUID `json:"id"`
	Content   string    `json:"content"`
	CreatedAt time.Time `json:"createdAt"`
	UpdatedAt time.Time `json:"updatedAt"`
}

func (c RiceComment) ToDTO() RiceCommentDTO {
	return RiceCommentDTO{
		Id:        c.Id,
		Content:   c.Content,
		CreatedAt: c.CreatedAt,
		UpdatedAt: c.UpdatedAt,
	}
}

type CommentWithUserDTO struct {
	CommentId   uuid.UUID `json:"commentId"`
	Content     string    `json:"content"`
	DisplayName string    `json:"displayName"`
	Username    string    `json:"username"`
	Avatar      string    `json:"avatar"`
	CreatedAt   time.Time `json:"createdAt"`
	UpdatedAt   time.Time `json:"updatedAt"`
}

func (c CommentWithUser) ToDTO() CommentWithUserDTO {
	return CommentWithUserDTO{
		CommentId:   c.CommentId,
		Content:     c.Content,
		DisplayName: c.DisplayName,
		Username:    c.Username,
		Avatar:      utils.Config.CDNUrl + c.AvatarPath,
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
	Id          uuid.UUID `json:"id"`
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
		Id:          r.Id,
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
	Id          uuid.UUID  `json:"id"`
	ReporterId  uuid.UUID  `json:"reporterId"`
	DisplayName string     `json:"displayName"`
	Username    string     `json:"username"`
	Reason      string     `json:"reason"`
	RiceId      *uuid.UUID `json:"riceId,omitempty"`
	CommentId   *uuid.UUID `json:"commentId,omitempty"`
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
