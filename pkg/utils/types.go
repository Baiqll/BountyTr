package utils

type NewScope struct {
	NewFailTarget string
	NewTarget     string
	NewApp        string
	NewPublicURL  string
	NewPrivateURL string
	IsFromAPI     bool // 是否来自 API（非 GitHub 数据）
}

type HandleClassifier struct{
	Type string
	Handle string
}