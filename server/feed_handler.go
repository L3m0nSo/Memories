package server

import (
	"fmt"
	"net/http"
	"strings"

	fieldtypes "github.com/L3m0nSo/Memories/server/columntypes"
	"github.com/L3m0nSo/Memories/server/resource"
	"github.com/artpar/api2go/v2"
	"github.com/gin-gonic/gin"
	"github.com/gorilla/feeds"
	"github.com/jmoiron/sqlx"
)

func CreateFeedHandler(cruds map[string]*resource.DbResource, streams []*resource.StreamProcessor, transaction *sqlx.Tx) func(*gin.Context) {

	streamMap := make(map[string]*resource.StreamProcessor)

	for _, stream := range streams {
		streamMap[stream.GetName()] = stream
	}

	feedsInfo, err := cruds["feed"].GetAllRawObjectsWithTransaction("feed", transaction)
	resource.CheckErr(err, "Failed to load feeds")
	streamInfos, err := cruds["stream"].GetAllRawObjectsWithTransaction("stream", transaction)
	resource.CheckErr(err, "Failed to load stream")

	feedMap := make(map[string]map[string]interface{})
	streamInfoMap := make(map[string]map[string]interface{})
	for _, feed := range feedsInfo {
		feedMap[feed["feed_name"].(string)] = feed
	}
	for _, stream := range streamInfos {
		s, ok := stream["id"].(string)
		if !ok {
			s = fmt.Sprintf("%v", stream["id"])
		}
		streamInfoMap[s] = stream
	}

	return func(c *gin.Context) {
		var feedName = c.Param("feedname")

		var parts = strings.Split(feedName, ".")
		if len(parts) < 2 {
			c.AbortWithStatusJSON(http.StatusBadRequest, gin.H{"error": "Invalid feed request"})
			return
		}
		feedName = parts[0]
		feedExtension := parts[1]

		feedInfo, ok := feedMap[feedName]
		if !ok || feedInfo == nil {
			c.AbortWithStatus(404)
			return
		}

		if feedInfo["enable"].(string) != "1" {
			c.AbortWithStatus(404)
			return
		}
		streamId, ok := feedInfo["stream_id"].(string)
		if !ok {
			c.AbortWithStatus(404)
			return
		}

		streamInfo, ok := streamInfoMap[streamId]
		if !ok {
			c.AbortWithStatus(404)
			return
		}

		streamProcessor, ok := streamMap[streamInfo["stream_name"].(string)]
		if !ok {
			c.AbortWithStatus(404)
			return
		}

		pageSize := feedInfo["page_size"].(string)

		pr := &http.Request{
			Method: "GET",
			URL:    c.Request.URL,
		}

		pr = pr.WithContext(c.Request.Context())

		req := api2go.Request{
			PlainRequest: pr,
			QueryParams: map[string][]string{
				"page[size]": {pageSize},
			},
		}

		_, rows, err := streamProcessor.PaginatedFindAll(req)

		if err != nil {
			c.AbortWithError(500, err)
			return
		}

		createdAtTime, _, _ := fieldtypes.GetTime(feedInfo["created_at"].(string))
		feed := &feeds.Feed{
			Title:       feedInfo["title"].(string),
			Link:        &feeds.Link{Href: feedInfo["link"].(string)},
			Description: feedInfo["description"].(string),
			Author:      &feeds.Author{Name: feedInfo["author_name"].(string), Email: feedInfo["author_email"].(string)},
			Created:     createdAtTime,
		}

		feedItems := make([]*feeds.Item, 0)

		for _, rowInterface := range rows.Result().([]api2go.Api2GoModel) {

			row := rowInterface.GetAttributes()
			createdAtTime, _, _ = fieldtypes.GetTime(row["created_at"].(string))
			feedItems = append(feedItems, &feeds.Item{
				Title:       row["title"].(string),
				Link:        &feeds.Link{Href: row["link"].(string)},
				Description: row["description"].(string),
				Author:      &feeds.Author{Name: row["author_name"].(string), Email: row["author_email"].(string)},
				Created:     createdAtTime,
			})

		}

		feed.Items = feedItems

		var output string
		switch strings.ToLower(feedExtension) {
		case "rss":
			c.Header("Content-Type", "application/xml")
			output, err = feed.ToRss()
		case "atom":
			c.Header("Content-Type", "application/xml")
			output, err = feed.ToAtom()
		case "json":
			c.Header("Content-Type", "application/json")
			output, err = feed.ToJSON()
		default:
			c.Header("Content-Type", "application/xml")
			output, err = feed.ToRss()
		}

		resource.CheckErr(err, "Failed to generate feed [%v]", feedInfo)

		c.Writer.WriteString(output)
		c.AbortWithStatus(200)

	}
}
