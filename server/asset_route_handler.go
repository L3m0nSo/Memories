package server

import (
	"fmt"
	"github.com/artpar/api2go"
	"github.com/daptin/daptin/server/resource"
	"github.com/gin-gonic/gin"
	log "github.com/sirupsen/logrus"
	"io"
	"net/http"
	"os"
	"path/filepath"
	"strconv"
	"strings"
	"time"
)

func AssetRouteHandler(cruds map[string]*resource.DbResource) func(c *gin.Context) {
	return func(c *gin.Context) {
		typeName := c.Param("typename")
		resourceUuid := c.Param("resource_id")
		columnNameWithExt := c.Param("columnname")
		columnNameWithoutExt := columnNameWithExt

		if strings.Index(columnNameWithoutExt, ".") > -1 {
			columnNameWithoutExt = columnNameWithoutExt[:strings.LastIndex(columnNameWithoutExt, ".")]
		}

		// Generate a cache key for this request
		cacheKey := fmt.Sprintf("%s:%s:%s:%s:%s",
			typeName,
			resourceUuid,
			columnNameWithoutExt,
			c.Query("index"),
			c.Query("file"))

		// Check if we have a cached file for this request
		if cachedFile, found := fileCache.Get(cacheKey); found {
			// Check if client has fresh copy using ETag
			if clientEtag := c.GetHeader("If-None-Match"); clientEtag != "" && clientEtag == cachedFile.ETag {
				c.Header("Cache-Control", "public, max-age=31536000") // 1 year for 304 responses
				c.Header("ETag", cachedFile.ETag)
				c.AbortWithStatus(http.StatusNotModified)
				return
			}

			// Set basic headers from cache
			c.Header("Content-Type", cachedFile.MimeType)
			c.Header("ETag", cachedFile.ETag)

			// Set cache control based on expiry time
			maxAge := int(time.Until(cachedFile.ExpiresAt).Seconds())
			if maxAge <= 0 {
				maxAge = 60 // Minimum 1 minute for almost expired resources
			}
			c.Header("Cache-Control", fmt.Sprintf("public, max-age=%d", maxAge))

			// Add content disposition if needed
			if cachedFile.IsDownload {
				c.Header("Content-Disposition", fmt.Sprintf("attachment; filename=\"%v\"", filepath.Base(cachedFile.Path)))
			} else {
				c.Header("Content-Disposition", fmt.Sprintf("inline; filename=\"%v\"", filepath.Base(cachedFile.Path)))
			}

			// Check if client accepts gzip and we have compressed data
			if cachedFile.GzipData != nil && len(cachedFile.GzipData) > 0 && strings.Contains(c.GetHeader("Accept-Encoding"), "gzip") {
				c.Header("Content-Encoding", "gzip")
				c.Header("Vary", "Accept-Encoding")
				c.Data(http.StatusOK, cachedFile.MimeType, cachedFile.GzipData)
				return
			}

			// Serve uncompressed data
			c.Data(http.StatusOK, cachedFile.MimeType, cachedFile.Data)
			return
		}

		// Parse column name and extension
		//parts := strings.SplitN(columnNameWithExt, ".", 2)
		//if len(parts) == 0 {
		//	c.AbortWithStatus(http.StatusBadRequest)
		//	return
		//}
		columnName := columnNameWithoutExt

		// Fast path: check if the table exists
		table, ok := cruds[typeName]
		if !ok || table == nil {
			log.Errorf("table not found [%v]", typeName)
			c.AbortWithStatus(http.StatusNotFound)
			return
		}

		// Fast path: check if the column exists
		colInfo, ok := table.TableInfo().GetColumnByName(columnName)
		if !ok || colInfo == nil || (!colInfo.IsForeignKey && colInfo.ColumnType != "markdown") {
			log.Errorf("column [%v] info not found", columnName)
			c.AbortWithStatus(http.StatusNotFound)
			return
		}

		// Handle markdown directly (simple case)
		if colInfo.ColumnType == "markdown" {
			// Fetch data
			pr := &http.Request{
				Method: "GET",
				URL:    c.Request.URL,
			}
			pr = pr.WithContext(c.Request.Context())

			req := api2go.Request{
				PlainRequest: pr,
			}

			obj, err := cruds[typeName].FindOne(resourceUuid, req)
			if err != nil {
				c.AbortWithStatus(http.StatusInternalServerError)
				return
			}

			row := obj.Result().(api2go.Api2GoModel)
			colData := row.GetAttributes()[columnName]
			if colData == nil {
				c.AbortWithStatus(http.StatusNotFound)
				return
			}

			markdownContent := colData.(string)

			// Generate ETag
			etag := generateETag([]byte(markdownContent), time.Now())

			// Check if client has fresh copy
			if clientEtag := c.GetHeader("If-None-Match"); clientEtag != "" && clientEtag == etag {
				c.Header("ETag", etag)
				c.Header("Cache-Control", "public, max-age=86400") // 1 day
				c.AbortWithStatus(http.StatusNotModified)
				return
			}

			// Cache the markdown content
			htmlContent := fmt.Sprintf("<pre>%s</pre>", markdownContent)
			cachedMarkdown := &CachedFile{
				Data:       []byte(htmlContent),
				ETag:       etag,
				Modtime:    time.Now(),
				MimeType:   "text/html; charset=utf-8",
				Size:       len(htmlContent),
				Path:       fmt.Sprintf("%s/%s/%s", typeName, resourceUuid, columnNameWithExt),
				IsDownload: false,
				ExpiresAt:  CalculateExpiry("text/html", ""),
			}

			// Create compressed version if large enough
			if len(htmlContent) > CompressionThreshold {
				if compressedData, err := CompressData([]byte(htmlContent)); err == nil {
					cachedMarkdown.GzipData = compressedData
				}
			}

			fileCache.Set(cacheKey, cachedMarkdown)

			// Return markdown as HTML with appropriate headers
			c.Header("Content-Type", "text/html; charset=utf-8")
			c.Header("Cache-Control", fmt.Sprintf("public, max-age=%d", int(time.Until(cachedMarkdown.ExpiresAt).Seconds())))
			c.Header("ETag", etag)

			// Use compression if client accepts it and we have compressed data
			if cachedMarkdown.GzipData != nil && strings.Contains(c.GetHeader("Accept-Encoding"), "gzip") {
				c.Header("Content-Encoding", "gzip")
				c.Header("Vary", "Accept-Encoding")
				c.Data(http.StatusOK, "text/html; charset=utf-8", cachedMarkdown.GzipData)
				return
			}

			c.Data(http.StatusOK, "text/html; charset=utf-8", cachedMarkdown.Data)
			return
		}

		// Handle foreign key (file data)
		if colInfo.IsForeignKey {
			// Get cache for this path
			assetCache, ok := cruds["world"].AssetFolderCache[typeName][columnName]
			if !ok {
				c.AbortWithStatus(http.StatusNotFound)
				return
			}

			// Find the file to serve
			pr := &http.Request{
				Method: "GET",
				URL:    c.Request.URL,
			}
			pr = pr.WithContext(c.Request.Context())

			req := api2go.Request{
				PlainRequest: pr,
			}

			obj, err := cruds[typeName].FindOne(resourceUuid, req)
			if err != nil {
				c.AbortWithStatus(http.StatusInternalServerError)
				return
			}

			row := obj.Result().(api2go.Api2GoModel)
			colData := row.GetAttributes()[columnName]
			if colData == nil {
				c.AbortWithStatus(http.StatusNotFound)
				return
			}

			// Find the correct file
			fileNameToServe := ""
			fileType := "application/octet-stream"
			colDataMapArray := colData.([]map[string]interface{})

			indexByQuery := c.Query("index")
			var indexByQueryInt = -1
			indexByQueryInt, err = strconv.Atoi(indexByQuery)
			nameByQuery := c.Query("file")

			// Logic to find the right file based on index or name
			if err == nil && indexByQueryInt > -1 && indexByQueryInt < len(colDataMapArray) {
				fileData := colDataMapArray[indexByQueryInt]
				fileName := fileData["name"].(string)
				queryFile := nameByQuery

				if queryFile == fileName || queryFile == "" {
					// Determine filename
					if fileData["path"] != nil && len(fileData["path"].(string)) > 0 {
						fileNameToServe = fileData["path"].(string) + "/" + fileName
					} else {
						fileNameToServe = fileName
					}

					// Determine mime type
					if typFromData, ok := fileData["type"]; ok {
						if typeStr, isStr := typFromData.(string); isStr {
							fileType = typeStr
						} else {
							fileType = GetMimeType(fileNameToServe)
						}
					} else {
						fileType = GetMimeType(fileNameToServe)
					}
				}
			} else {
				for _, fileData := range colDataMapArray {
					fileName := fileData["name"].(string)
					queryFile := nameByQuery

					if queryFile == fileName || queryFile == "" {
						// Determine filename
						if fileData["path"] != nil && len(fileData["path"].(string)) > 0 {
							fileNameToServe = fileData["path"].(string) + "/" + fileName
						} else {
							fileNameToServe = fileName
						}

						// Determine mime type
						if typFromData, ok := fileData["type"]; ok {
							if typeStr, isStr := typFromData.(string); isStr {
								fileType = typeStr
							} else {
								fileType = GetMimeType(fileNameToServe)
							}
						} else {
							fileType = GetMimeType(fileNameToServe)
						}

						break
					}
				}
			}

			if fileNameToServe == "" {
				c.AbortWithStatus(http.StatusNotFound)
				return
			}

			// Get file path
			filePath := assetCache.LocalSyncPath + string(os.PathSeparator) + fileNameToServe

			// Check if it's an image that needs processing
			if isImage := strings.HasPrefix(fileType, "image/"); isImage && c.Query("processImage") == "true" {
				// Use separate function for image processing
				file, err := cruds["world"].AssetFolderCache[typeName][columnName].GetFileByName(fileNameToServe)
				if err != nil {
					_ = c.AbortWithError(500, err)
					return
				}
				defer file.Close()
				HandleImageProcessing(c, file)
				return
			}

			// Check if file exists and get file info
			fileInfo, err := os.Stat(filePath)
			if err != nil {
				c.AbortWithStatus(http.StatusNotFound)
				return
			}

			// Determine if this should be a download
			isDownload := ShouldBeDownloaded(fileType, fileNameToServe)

			// Set response headers for all cases
			c.Header("Content-Type", fileType)

			// For downloads, add content disposition
			if isDownload {
				c.Header("Content-Disposition", fmt.Sprintf("attachment; filename=\"%v\"", fileNameToServe))
			} else {
				c.Header("Content-Disposition", fmt.Sprintf("inline; filename=\"%v\"", fileNameToServe))
			}

			// Calculate expiry time
			expiryTime := CalculateExpiry(fileType, filePath)

			// Set cache control header based on expiry
			maxAge := int(time.Until(expiryTime).Seconds())
			c.Header("Cache-Control", fmt.Sprintf("public, max-age=%d", maxAge))

			// Use optimized file serving for small files that can be cached
			if fileInfo.Size() <= MaxFileCacheSize {
				// Open file
				file, err := os.Open(filePath)
				if err != nil {
					c.AbortWithStatus(http.StatusInternalServerError)
					return
				}
				defer file.Close()

				// Read file into memory
				data, err := io.ReadAll(file)
				if err != nil {
					c.AbortWithStatus(http.StatusInternalServerError)
					return
				}

				// Generate ETag for client-side caching
				etag := generateETag(data, fileInfo.ModTime())

				// Check if client has fresh copy before we do anything else
				if clientEtag := c.GetHeader("If-None-Match"); clientEtag != "" && clientEtag == etag {
					c.Header("ETag", etag)
					c.AbortWithStatus(http.StatusNotModified)
					return
				}

				// Create cache entry
				newCachedFile := &CachedFile{
					Data:       data,
					ETag:       etag,
					Modtime:    fileInfo.ModTime(),
					MimeType:   fileType,
					Size:       len(data),
					Path:       filePath,
					IsDownload: isDownload,
					ExpiresAt:  expiryTime,
				}

				// Pre-compress text files for better performance
				needsCompression := ShouldCompress(fileType) && len(data) > CompressionThreshold
				if needsCompression {
					if compressedData, err := CompressData(data); err == nil {
						newCachedFile.GzipData = compressedData
					}
				}

				// Get file stat for validation
				if fileStat, err := GetFileStat(filePath); err == nil {
					newCachedFile.FileStat = fileStat
				}

				// Add to cache for future requests
				fileCache.Set(cacheKey, newCachedFile)

				// Set ETag header
				c.Header("ETag", etag)

				// Use compression if client accepts it and we have compressed data
				if newCachedFile.GzipData != nil && strings.Contains(c.GetHeader("Accept-Encoding"), "gzip") {
					c.Header("Content-Encoding", "gzip")
					c.Header("Vary", "Accept-Encoding")
					c.Data(http.StatusOK, fileType, newCachedFile.GzipData)
					return
				}

				// Serve uncompressed data
				c.Data(http.StatusOK, fileType, data)
				return
			}

			// For larger files, use http.ServeContent for efficient range requests
			// This is important for video/audio streaming
			file, err := os.Open(filePath)
			if err != nil {
				c.AbortWithStatus(http.StatusInternalServerError)
				return
			}
			defer file.Close()

			// Set ETag for large files too
			// Instead of reading the entire file, use file info to generate ETag
			etag := fmt.Sprintf("\"%x-%x\"", fileInfo.ModTime().Unix(), fileInfo.Size())
			c.Header("ETag", etag)

			// Check if client has fresh copy
			if clientEtag := c.GetHeader("If-None-Match"); clientEtag != "" && clientEtag == etag {
				c.AbortWithStatus(http.StatusNotModified)
				return
			}

			http.ServeContent(c.Writer, c.Request, fileNameToServe, fileInfo.ModTime(), file)
		}
	}
}
