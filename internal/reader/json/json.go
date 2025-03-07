// SPDX-FileCopyrightText: Copyright The Miniflux Authors. All rights reserved.
// SPDX-License-Identifier: Apache-2.0

package json // import "miniflux.app/v2/internal/reader/json"

import (
	"strings"
	"time"

	"miniflux.app/v2/internal/crypto"
	"miniflux.app/v2/internal/logger"
	"miniflux.app/v2/internal/model"
	"miniflux.app/v2/internal/reader/date"
	"miniflux.app/v2/internal/reader/sanitizer"
	"miniflux.app/v2/internal/urllib"
)

type jsonFeed struct {
	Version    string       `json:"version"`
	Title      string       `json:"title"`
	SiteURL    string       `json:"home_page_url"`
	IconURL    string       `json:"icon"`
	FaviconURL string       `json:"favicon"`
	FeedURL    string       `json:"feed_url"`
	Authors    []jsonAuthor `json:"authors"`
	Author     jsonAuthor   `json:"author"`
	Items      []jsonItem   `json:"items"`
}

type jsonAuthor struct {
	Name string `json:"name"`
	URL  string `json:"url"`
}

type jsonItem struct {
	ID            string           `json:"id"`
	URL           string           `json:"url"`
	Title         string           `json:"title"`
	Summary       string           `json:"summary"`
	Text          string           `json:"content_text"`
	HTML          string           `json:"content_html"`
	DatePublished string           `json:"date_published"`
	DateModified  string           `json:"date_modified"`
	Authors       []jsonAuthor     `json:"authors"`
	Author        jsonAuthor       `json:"author"`
	Attachments   []jsonAttachment `json:"attachments"`
	Tags          []string         `json:"tags"`
}

type jsonAttachment struct {
	URL      string `json:"url"`
	MimeType string `json:"mime_type"`
	Title    string `json:"title"`
	Size     int64  `json:"size_in_bytes"`
	Duration int    `json:"duration_in_seconds"`
}

func (j *jsonFeed) GetAuthor() string {
	if len(j.Authors) > 0 {
		return (getAuthor(j.Authors[0]))
	}
	return getAuthor(j.Author)
}

func (j *jsonFeed) Transform(baseURL string) *model.Feed {
	var err error

	feed := new(model.Feed)

	feed.FeedURL, err = urllib.AbsoluteURL(baseURL, j.FeedURL)
	if err != nil {
		feed.FeedURL = j.FeedURL
	}

	feed.SiteURL, err = urllib.AbsoluteURL(baseURL, j.SiteURL)
	if err != nil {
		feed.SiteURL = j.SiteURL
	}

	feed.IconURL = strings.TrimSpace(j.IconURL)

	if feed.IconURL == "" {
		feed.IconURL = strings.TrimSpace(j.FaviconURL)
	}

	feed.Title = strings.TrimSpace(j.Title)
	if feed.Title == "" {
		feed.Title = feed.SiteURL
	}

	for _, item := range j.Items {
		entry := item.Transform()
		entryURL, err := urllib.AbsoluteURL(feed.SiteURL, entry.URL)
		if err == nil {
			entry.URL = entryURL
		}

		if entry.Author == "" {
			entry.Author = j.GetAuthor()
		}

		feed.Entries = append(feed.Entries, entry)
	}

	return feed
}

func (j *jsonItem) GetDate() time.Time {
	for _, value := range []string{j.DatePublished, j.DateModified} {
		if value != "" {
			d, err := date.Parse(value)
			if err != nil {
				logger.Error("json: %v", err)
				return time.Now()
			}

			return d
		}
	}

	return time.Now()
}

func (j *jsonItem) GetAuthor() string {
	if len(j.Authors) > 0 {
		return getAuthor(j.Authors[0])
	}
	return getAuthor(j.Author)
}

func (j *jsonItem) GetHash() string {
	for _, value := range []string{j.ID, j.URL, j.Text + j.HTML + j.Summary} {
		if value != "" {
			return crypto.Hash(value)
		}
	}

	return ""
}

func (j *jsonItem) GetTitle() string {
	if j.Title != "" {
		return j.Title
	}

	for _, value := range []string{j.Summary, j.Text, j.HTML} {
		if value != "" {
			return sanitizer.TruncateHTML(value, 100)
		}
	}

	return j.URL
}

func (j *jsonItem) GetContent() string {
	for _, value := range []string{j.HTML, j.Text, j.Summary} {
		if value != "" {
			return value
		}
	}

	return ""
}

func (j *jsonItem) GetEnclosures() model.EnclosureList {
	enclosures := make(model.EnclosureList, 0)

	for _, attachment := range j.Attachments {
		if attachment.URL == "" {
			continue
		}

		enclosures = append(enclosures, &model.Enclosure{
			URL:      attachment.URL,
			MimeType: attachment.MimeType,
			Size:     attachment.Size,
		})
	}

	return enclosures
}

func (j *jsonItem) Transform() *model.Entry {
	entry := model.NewEntry()
	entry.URL = j.URL
	entry.Date = j.GetDate()
	entry.Author = j.GetAuthor()
	entry.Hash = j.GetHash()
	entry.Content = j.GetContent()
	entry.Title = strings.TrimSpace(j.GetTitle())
	entry.Enclosures = j.GetEnclosures()
	if len(j.Tags) > 0 {
		entry.Tags = j.Tags
	}

	return entry
}

func getAuthor(author jsonAuthor) string {
	if author.Name != "" {
		return strings.TrimSpace(author.Name)
	}

	return ""
}
