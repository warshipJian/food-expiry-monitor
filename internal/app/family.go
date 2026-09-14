package app

import (
	"crypto/rand"
	"database/sql"
	"fmt"
	"math/big"
	"net/http"
	"strings"

	"github.com/gin-gonic/gin"
)

const inviteAlphabet = "23456789ABCDEFGHJKLMNPQRSTUVWXYZ"

type familyInput struct {
	Name string `json:"name" binding:"required,max=30"`
}

type joinFamilyInput struct {
	InviteCode string `json:"inviteCode" binding:"required,len=6"`
}

type familyMember struct {
	Nickname  string `json:"nickname"`
	AvatarURL string `json:"avatarUrl"`
	Role      string `json:"role"`
}

type familyResponse struct {
	ID         string         `json:"id"`
	Name       string         `json:"name"`
	InviteCode string         `json:"inviteCode"`
	Members    []familyMember `json:"members"`
}

func (s *Service) getFamily(c *gin.Context) {
	family, err := s.findFamilyForUser(userID(c))
	if err == sql.ErrNoRows {
		c.JSON(http.StatusOK, gin.H{"family": nil})
		return
	}
	if err != nil {
		internalError(c, err)
		return
	}
	c.JSON(http.StatusOK, gin.H{"family": family})
}

func (s *Service) createFamily(c *gin.Context) {
	var input familyInput
	if err := c.ShouldBindJSON(&input); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "family name is required"})
		return
	}
	name := strings.TrimSpace(input.Name)
	if name == "" {
		c.JSON(http.StatusBadRequest, gin.H{"error": "family name is required"})
		return
	}
	uid := userID(c)
	if _, err := s.findFamilyForUser(uid); err != sql.ErrNoRows {
		if err != nil {
			internalError(c, err)
		} else {
			c.JSON(http.StatusConflict, gin.H{"error": "already in a family"})
		}
		return
	}
	tx, err := s.db.Begin()
	if err != nil {
		internalError(c, err)
		return
	}
	defer tx.Rollback()
	familyID, inviteCode := newID(), ""
	for i := 0; i < 5; i++ {
		inviteCode = newInviteCode()
		if _, err = tx.Exec("INSERT INTO families (id,name,invite_code) VALUES (?,?,?)", familyID, name, inviteCode); err == nil {
			break
		}
	}
	if err != nil {
		internalError(c, err)
		return
	}
	if _, err = tx.Exec("INSERT INTO family_members (family_id,user_id,role) VALUES (?,?,'owner')", familyID, uid); err != nil {
		internalError(c, err)
		return
	}
	if _, err = tx.Exec("UPDATE foods SET family_id=? WHERE user_id=? AND family_id IS NULL", familyID, uid); err != nil {
		internalError(c, err)
		return
	}
	if err = tx.Commit(); err != nil {
		internalError(c, err)
		return
	}
	family, err := s.findFamilyForUser(uid)
	if err != nil {
		internalError(c, err)
		return
	}
	c.JSON(http.StatusCreated, family)
}

func (s *Service) joinFamily(c *gin.Context) {
	var input joinFamilyInput
	if err := c.ShouldBindJSON(&input); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "邀请码应为 6 位字符"})
		return
	}
	uid, code := userID(c), strings.ToUpper(strings.TrimSpace(input.InviteCode))
	if _, err := s.findFamilyForUser(uid); err != sql.ErrNoRows {
		if err != nil {
			internalError(c, err)
		} else {
			c.JSON(http.StatusConflict, gin.H{"error": "already in a family"})
		}
		return
	}
	var familyID string
	if err := s.db.QueryRow("SELECT id FROM families WHERE invite_code=?", code).Scan(&familyID); err == sql.ErrNoRows {
		c.JSON(http.StatusNotFound, gin.H{"error": "邀请码不存在"})
		return
	} else if err != nil {
		internalError(c, err)
		return
	}
	tx, err := s.db.Begin()
	if err != nil {
		internalError(c, err)
		return
	}
	defer tx.Rollback()
	if _, err = tx.Exec("INSERT INTO family_members (family_id,user_id) VALUES (?,?)", familyID, uid); err != nil {
		internalError(c, err)
		return
	}
	if _, err = tx.Exec("UPDATE foods SET family_id=? WHERE user_id=? AND family_id IS NULL", familyID, uid); err != nil {
		internalError(c, err)
		return
	}
	if err = tx.Commit(); err != nil {
		internalError(c, err)
		return
	}
	family, err := s.findFamilyForUser(uid)
	if err != nil {
		internalError(c, err)
		return
	}
	c.JSON(http.StatusOK, family)
}

func (s *Service) findFamilyForUser(uid string) (familyResponse, error) {
	var family familyResponse
	err := s.db.QueryRow(`SELECT f.id,f.name,f.invite_code FROM families f JOIN family_members fm ON fm.family_id=f.id WHERE fm.user_id=?`, uid).Scan(&family.ID, &family.Name, &family.InviteCode)
	if err != nil {
		return family, err
	}
	rows, err := s.db.Query(`SELECT u.nickname,u.avatar_url,fm.role FROM family_members fm JOIN users u ON u.id=fm.user_id WHERE fm.family_id=? ORDER BY fm.role DESC,fm.created_at ASC`, family.ID)
	if err != nil {
		return family, err
	}
	defer rows.Close()
	family.Members = make([]familyMember, 0)
	for rows.Next() {
		var member familyMember
		if err := rows.Scan(&member.Nickname, &member.AvatarURL, &member.Role); err != nil {
			return family, err
		}
		family.Members = append(family.Members, member)
	}
	return family, rows.Err()
}

func (s *Service) familyIDForUser(uid string) (string, error) {
	var familyID string
	err := s.db.QueryRow("SELECT family_id FROM family_members WHERE user_id=?", uid).Scan(&familyID)
	return familyID, err
}

func newInviteCode() string {
	result := make([]byte, 6)
	for i := range result {
		index, err := rand.Int(rand.Reader, big.NewInt(int64(len(inviteAlphabet))))
		if err != nil {
			panic(err)
		}
		result[i] = inviteAlphabet[index.Int64()]
	}
	return string(result)
}

func familyScopeClause(uid string, familyID string) (string, []any) {
	if familyID != "" {
		return "family_id=?", []any{familyID}
	}
	return "user_id=? AND family_id IS NULL", []any{uid}
}

func scopeError(err error) error { return fmt.Errorf("resolve family scope: %w", err) }
