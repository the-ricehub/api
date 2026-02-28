package utils

// Try to construct URL from avatar path
// or use the default one if user didn't set any
func GetUserAvatar(avatarPath *string) string {
	avatar := Config.CDNUrl + Config.DefaultAvatar
	if avatarPath != nil {
		avatar = Config.CDNUrl + *avatarPath
	}
	return avatar
}
