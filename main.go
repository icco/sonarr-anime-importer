package main

import (
	"encoding/json"
	"io"
	"log"
	"net/http"
	"net/url"
	"os"
	"reflect"
	"slices"
	"strconv"
	"strings"
	"sync"
	"time"

	"github.com/darenliang/jikan-go"
)

type ResponseItem struct {
	Title  string `json:"title"`
	MalId  int    `json:"malId"`
	TvdbId int    `json:"tvdbId"`
}
type AnimeEntry struct {
	TvdbId int         `json:"tvdb_id"`
	MalId  interface{} `json:"mal_id"`
}
type ConcurrentMap struct {
	m   map[int]int
	mut sync.RWMutex
}

func (m *ConcurrentMap) Get(i int) int {
	m.mut.RLock()
	defer m.mut.RUnlock()
	return m.m[i]
}

var lastBuiltAnimeIdList time.Time

func main() {
	log.Println("sonarr-mal-importer v0.1.1")
	log.Println("Building Anime ID Associations...")
	var malToTvdb = new(ConcurrentMap)
	buildIdMap(malToTvdb)
	permaSkipMalStr := os.Getenv("ALWAYS_SKIP_MAL_IDS")
	permaSkipMalIds := strings.Split(permaSkipMalStr, ",")
	if permaSkipMalStr != "" {
		log.Printf("Always skipping: %v\n", permaSkipMalIds)
	}
	http.HandleFunc("/anime", handleAnimeSearch(malToTvdb, permaSkipMalIds))
	log.Println("Listening on :3333")
	log.Fatal(http.ListenAndServe(":3333", nil))
}

func handleAnimeSearch(malToTvdb *ConcurrentMap, permaSkipMalIds []string) func(w http.ResponseWriter, r *http.Request) {
	return func(w http.ResponseWriter, r *http.Request) {
		log.Printf("%s %s?%s", r.Method, r.URL.Path, r.URL.RawQuery)
		if time.Since(lastBuiltAnimeIdList) > 24*time.Hour {
			log.Println("Anime ID association table expired, building new table...")
			buildIdMap(malToTvdb)
		}
		search, err := getAnimeSearch(malToTvdb, permaSkipMalIds, r)
		if err != nil {
			w.WriteHeader(500)
		} else {
			w.Write([]byte(search))
		}
	}
}

func getAnimeSearch(malToTvdb *ConcurrentMap, permaSkipMalIds []string, r *http.Request) (string, error) {
	q := r.URL.Query()

	limit, err := strconv.Atoi(q.Get("limit"))
	if err != nil {
		limit = 9999 // limit not specified or invalid
	}

	skipDedup := parseBoolParam(q, "allow_duplicates")

	// for some reason Jikan responds with 400 Bad Request for any limit >25
	// so instead, we just limit when mapping the data and remove the limit from the Jikan request
	q.Del("limit")

	hasNextPage := true
	page := 0
	resp := []ResponseItem{}
	count := 0
	usedIds := make(map[int]bool, 0)
	for hasNextPage {
		page++
		q.Set("page", strconv.Itoa(page))
		result, err := jikan.GetAnimeSearch(q)
		if err != nil {
			log.Println("Error sending request to Jikan: ", err)
			return "", err
		}

		// map the data
		for _, item := range result.Data {
			if malToTvdb.Get(item.MalId) == 0 {
				log.Printf("MyAnimeList ID %d (%s a.k.a. %s) has no associated TVDB ID, skipping...\n", item.MalId, item.Title, item.TitleEnglish)
				continue
			}
			if usedIds[item.MalId] && !skipDedup {
				log.Printf("MyAnimeList ID %d (%s a.k.a. %s) is a duplicate, skipping...\n", item.MalId, item.Title, item.TitleEnglish)
				continue
			}
			if slices.Contains(permaSkipMalIds, strconv.Itoa(item.MalId)) {
				log.Printf("MyAnimeList ID %d (%s a.k.a. %s) is set to always skip, skipping...\n", item.MalId, item.Title, item.TitleEnglish)
				continue
			}
			count++
			if count > limit {
				break
			}
			resp = append(resp,
				ResponseItem{
					item.Title,
					item.MalId,
					malToTvdb.Get(item.MalId),
				})
			usedIds[item.MalId] = true
		}
		hasNextPage = result.Pagination.HasNextPage
		if count > limit {
			break
		}
		if hasNextPage {
			time.Sleep(500 * time.Millisecond) // sleep between requests for new page to try and avoid rate limits
		}
	}

	respJson, err := json.MarshalIndent(resp, "", "  ")
	if err != nil {
		log.Println("Error marshalling response: ", err)
		return "", err
	}
	return string(respJson), nil
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
	idListBytes, err = io.ReadAll(resp.Body)
	if err != nil {
		log.Fatal("Error reading anime_ids.json: ", err)
	}

	var animeMap map[string]AnimeEntry
	err = json.Unmarshal(idListBytes, &animeMap)
	if err != nil {
		log.Fatal("Error unmarshalling anime_ids.json: ", err)
	}
	idMap.m = make(map[int]int, 0)
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
			idMap.m[val] = entry.TvdbId
		}
	}
	lastBuiltAnimeIdList = time.Now()
}

// parses the boolean param "name" from url.Values "values"
func parseBoolParam(values url.Values, name string) bool {
	param := values.Get(name)

	if param != "" {
		val, err := strconv.ParseBool(param)
		if err == nil {
			return val
		}
	} else if _, exists := values[name]; exists {
		return true
	}
	return false
}
