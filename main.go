package main

import (
	"encoding/json"
	"io"
	"log"
	"net/http"
	"os"
	"reflect"
	"strconv"
	"strings"
	"sync"
	"time"
)

type ResponseItem struct {
	Title     string `json:"title"`
	TitleEng  string `json:"titleEnglish,omitempty"`
	MalId     int    `json:"malId,omitempty"`
	AnilistId int    `json:"anilistId,omitempty"`
	TvdbId    int    `json:"tvdbId"`
}
type AnimeEntry struct {
	TvdbId    int         `json:"tvdb_id"`
	MalId     interface{} `json:"mal_id"`
	AnilistId int         `json:"anilist_id"`
}
type ConcurrentMap struct {
	mal map[int]int
	mut sync.RWMutex
}

func (m *ConcurrentMap) GetByMalId(i int) int {
	m.mut.RLock()
	defer m.mut.RUnlock()
	return m.mal[i]
}

var lastBuiltAnimeIdList time.Time

func main() {
	log.Println("sonarr-anime-importer v0.2.0")
	log.Println("Building Anime ID Associations...")
	var idMap = new(ConcurrentMap)
	buildIdMap(idMap)
	permaSkipMalStr := os.Getenv("ALWAYS_SKIP_MAL_IDS")
	permaSkipMalIds := strings.Split(permaSkipMalStr, ",")
	if permaSkipMalStr != "" {
		log.Printf("Always skipping MAL IDs: %v\n", permaSkipMalIds)
	}
	permaSkipAnilistStr := os.Getenv("ALWAYS_SKIP_ANILIST_IDS")
	permaSkipAnilistIds := strings.Split(permaSkipAnilistStr, ",")
	if permaSkipAnilistStr != "" {
		log.Printf("Always skipping Anilist IDs: %v\n", permaSkipAnilistIds)
	}
	buildIdMapMiddleware := newRebuildStaleIdMapMiddleware(idMap)
	http.HandleFunc("/v1/mal/anime", loggerMiddleware(buildIdMapMiddleware(handleMalAnimeSearch(idMap, permaSkipMalIds))))
	http.HandleFunc("/v1/anilist/anime", loggerMiddleware(buildIdMapMiddleware(handleAnilistAnimeSearch(idMap, permaSkipAnilistIds))))
	log.Println("Listening on :3333")
	log.Fatal(http.ListenAndServe(":3333", nil))
}

func buildIdMap(idMap *ConcurrentMap) {
	// build/re-build the mal -> tvdb association table
	idMap.mut.Lock()
	defer idMap.mut.Unlock()
	var idListBytes []byte
	resp, err := http.Get("https://raw.githubusercontent.com/Kometa-Team/Anime-IDs/master/anime_ids.json")
	if err != nil {
		log.Fatal("Error fetching anime_ids.json: ", err)
	}
	defer resp.Body.Close()
	idListBytes, err = io.ReadAll(resp.Body)
	if err != nil {
		log.Fatal("Error reading anime_ids.json: ", err)
	}

	var animeMap map[string]AnimeEntry
	err = json.Unmarshal(idListBytes, &animeMap)
	if err != nil {
		log.Fatal("Error unmarshalling anime_ids.json: ", err)
	}
	idMap.mal = make(map[int]int, 0)
	for _, entry := range animeMap {
		if entry.MalId == nil {
			continue
		}
		var malIdList []int
		switch t := reflect.TypeOf(entry.MalId); t.Kind() {
		case reflect.String:
			s := strings.Split(entry.MalId.(string), ",")
			for _, ss := range s {
				id, err := strconv.Atoi(ss)
				if err != nil {
					log.Fatal("Error building anime id associations: ", err)
				}
				malIdList = append(malIdList, id)
			}
		case reflect.Float64:
			malIdList = append(malIdList, int(entry.MalId.(float64)))
		}
		for _, val := range malIdList {
			idMap.mal[val] = entry.TvdbId
		}
		if entry.AnilistId == 0 {
			continue
		}
	}
	lastBuiltAnimeIdList = time.Now()
}
