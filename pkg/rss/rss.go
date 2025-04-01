package rss

import (
	"context"
	"encoding/json"
	"fmt"
	"goNews/pkg/db"
	"io"
	"net/http"
	"os"
	"regexp"
	"time"
)

type rss struct {
	Links  []string `json:"rss"`
	Period int      `json:"request_period"`
}

func Rss(db *db.DB) error {
	file, err := os.ReadFile("./src/config.json")
	if err != nil {
		return fmt.Errorf("не удалось прочитать config.json: %v", err)
	}
	rssConf := rss{}
	err = json.Unmarshal(file, &rssConf)
	if err != nil {
		return fmt.Errorf("не удалось распарсить config.json: %v", err)
	}

	for {
		for _, link := range rssConf.Links {
			go func(link string) {
				resp, err := http.Get(link)
				if err != nil {
					fmt.Printf("ошибка HTTP-запроса к %s: %v\n", link, err)
					return
				}

				body, err := io.ReadAll(resp.Body)
				if err != nil {
					fmt.Printf("ошибка чтения ответа от %s: %v\n", link, err)
					return
				}

				reItem := regexp.MustCompile(`(?s)<item>.*?</item>`)
				items := reItem.FindAllString(string(body), -1)

				reTitle := regexp.MustCompile(`<title><!\[CDATA\[(.*?)]]></title>`)
				rePubDate := regexp.MustCompile(`<pubDate>(.*?)</pubDate>`)
				reDescription := regexp.MustCompile(`<description>(.*?)</description>`)
				reCData := regexp.MustCompile(`<!\[CDATA\[(.*?)]]>`)
				reTags := regexp.MustCompile(`(?s)<.*?>`)

				for _, item := range items {
					titleMatch := reTitle.FindStringSubmatch(item)
					pubDateMatch := rePubDate.FindStringSubmatch(item)
					descriptionMatch := reDescription.FindStringSubmatch(item)

					if len(titleMatch) > 1 && len(pubDateMatch) > 1 && len(descriptionMatch) > 1 {
						title := titleMatch[1]
						pubDate := pubDateMatch[1]
						description := descriptionMatch[1]

						cdataMatch := reCData.FindStringSubmatch(description)
						if len(cdataMatch) > 1 {
							description = cdataMatch[1]
						}

						description = reTags.ReplaceAllString(description, "")

						_, err := db.Pool.Query(context.Background(),
							"INSERT INTO news (name, description, publication_date, link) SELECT $1, $2, $3, $4 WHERE NOT EXISTS (SELECT 1 FROM news WHERE name = $1);",
							title, description, pubDate, link)
						if err != nil {
							fmt.Printf("ошибка записи в БД для %s: %v\n", title, err)
						}
					}
				}
			}(link)
		}
		time.Sleep(time.Duration(rssConf.Period) * time.Second)
	}
}
