package controller

import (
	"io"
	"net/http"
	"net/url"
	"strings"

	"github.com/gin-gonic/gin"
)

const leaderboardBaseURL = "http://192.168.4.19:7860"

func GetLeaderboard(c *gin.Context) {
	domain := strings.TrimSpace(c.DefaultQuery("domain", "general"))
	target := leaderboardBaseURL + "/api/leaderboard?domain=" + url.QueryEscape(domain)
	resp, err := http.Get(target)
	if err != nil {
		c.JSON(http.StatusBadGateway, gin.H{"success": false, "message": err.Error()})
		return
	}
	defer resp.Body.Close()
	body, _ := io.ReadAll(resp.Body)
	c.Data(resp.StatusCode, "application/json", body)
}

// DeleteLeaderboardItem is root-only operation that proxies to leaderboard service.
// It supports common delete styles:
// 1) DELETE /api/leaderboard/{id}
// 2) DELETE /api/leaderboard?id={id}
func DeleteLeaderboardItem(c *gin.Context) {
	id := strings.TrimSpace(c.Param("id"))
	if id == "" {
		c.JSON(http.StatusBadRequest, gin.H{"success": false, "message": "missing leaderboard id"})
		return
	}

	tryURLs := []string{
		leaderboardBaseURL + "/api/leaderboard/" + url.PathEscape(id),
		leaderboardBaseURL + "/api/leaderboard?id=" + url.QueryEscape(id),
	}

	var lastStatus = http.StatusBadGateway
	var lastBody []byte
	for _, u := range tryURLs {
		req, _ := http.NewRequest(http.MethodDelete, u, nil)
		resp, err := http.DefaultClient.Do(req)
		if err != nil {
			lastBody = []byte(`{"success":false,"message":"` + err.Error() + `"}`)
			continue
		}
		body, _ := io.ReadAll(resp.Body)
		resp.Body.Close()
		lastStatus = resp.StatusCode
		lastBody = body
		if resp.StatusCode >= 200 && resp.StatusCode < 300 {
			c.Data(resp.StatusCode, "application/json", body)
			return
		}
	}

	c.Data(lastStatus, "application/json", lastBody)
}

