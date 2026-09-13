package app

import (
	"io"
	"net/http"
	"os"
	"path/filepath"
	"strings"

	"github.com/gin-gonic/gin"
)

const maxAvatarSize = 5 << 20

type profile struct {
	Nickname  string `json:"nickname"`
	AvatarURL string `json:"avatarUrl"`
}

type profileInput struct {
	Nickname string `json:"nickname" binding:"max=20"`
}

func (s *Service) getProfile(c *gin.Context) {
	result, err := s.findProfile(userID(c))
	if err != nil {
		internalError(c, err)
		return
	}
	c.JSON(http.StatusOK, result)
}

func (s *Service) updateProfile(c *gin.Context) {
	var input profileInput
	if err := c.ShouldBindJSON(&input); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "invalid profile"})
		return
	}
	nickname := strings.TrimSpace(input.Nickname)
	if nickname == "" {
		c.JSON(http.StatusBadRequest, gin.H{"error": "nickname cannot be empty"})
		return
	}
	if _, err := s.db.Exec("UPDATE users SET nickname=? WHERE id=?", nickname, userID(c)); err != nil {
		internalError(c, err)
		return
	}
	result, err := s.findProfile(userID(c))
	if err != nil {
		internalError(c, err)
		return
	}
	c.JSON(http.StatusOK, result)
}

func (s *Service) uploadAvatar(c *gin.Context) {
	c.Request.Body = http.MaxBytesReader(c.Writer, c.Request.Body, maxAvatarSize)
	file, _, err := c.Request.FormFile("avatar")
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "please upload an avatar smaller than 5 MB"})
		return
	}
	defer file.Close()
	firstBytes := make([]byte, 512)
	n, err := file.Read(firstBytes)
	if err != nil && err != io.EOF {
		internalError(c, err)
		return
	}
	contentType := http.DetectContentType(firstBytes[:n])
	extensions := map[string]string{"image/jpeg": ".jpg", "image/png": ".png", "image/webp": ".webp"}
	extension, ok := extensions[contentType]
	if !ok {
		c.JSON(http.StatusBadRequest, gin.H{"error": "avatar must be a JPEG, PNG, or WebP image"})
		return
	}
	filename := newID() + extension
	temporary, err := os.CreateTemp(s.cfg.UploadDir, ".avatar-*")
	if err != nil {
		internalError(c, err)
		return
	}
	temporaryName := temporary.Name()
	defer os.Remove(temporaryName)
	if _, err := io.Copy(temporary, io.MultiReader(strings.NewReader(string(firstBytes[:n])), file)); err != nil {
		temporary.Close()
		internalError(c, err)
		return
	}
	if err := temporary.Close(); err != nil {
		internalError(c, err)
		return
	}
	if err := os.Chmod(temporaryName, 0o640); err != nil {
		internalError(c, err)
		return
	}
	if err := os.Rename(temporaryName, filepath.Join(s.cfg.UploadDir, filename)); err != nil {
		internalError(c, err)
		return
	}
	avatarURL := "/uploads/" + filename
	if _, err := s.db.Exec("UPDATE users SET avatar_url=? WHERE id=?", avatarURL, userID(c)); err != nil {
		internalError(c, err)
		return
	}
	result, err := s.findProfile(userID(c))
	if err != nil {
		internalError(c, err)
		return
	}
	c.JSON(http.StatusOK, result)
}

func (s *Service) findProfile(uid string) (profile, error) {
	var result profile
	err := s.db.QueryRow("SELECT nickname, avatar_url FROM users WHERE id=?", uid).Scan(&result.Nickname, &result.AvatarURL)
	return result, err
}
