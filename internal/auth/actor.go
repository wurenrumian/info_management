package auth

import (
	"strconv"

	"github.com/gin-gonic/gin"
)

type Actor struct {
	UserID  uint
	Role    int
	ClassID uint
	Grade   string
}

const actorKey = "actor"

func SetActor(c *gin.Context, actor Actor) {
	c.Set(actorKey, actor)
}

func GetActor(c *gin.Context) (Actor, bool) {
	v, ok := c.Get(actorKey)
	if !ok {
		return Actor{}, false
	}
	a, ok := v.(Actor)
	return a, ok
}

func ParseUintHeader(c *gin.Context, key string) (uint, bool) {
	s := c.GetHeader(key)
	if s == "" {
		return 0, false
	}
	v, err := strconv.ParseUint(s, 10, 64)
	if err != nil {
		return 0, false
	}
	return uint(v), true
}
