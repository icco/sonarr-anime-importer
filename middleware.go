package main

import (
	"log"
	"net/http"
	"time"
)

func newRebuildStaleIdMapMiddleware(idMap *ConcurrentMap) func(http.Handler) http.Handler {
	return func(next http.Handler) http.Handler {
		return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			if time.Since(lastBuiltAnimeIdList) > 24*time.Hour {
				log.Println("Anime ID association table expired, building new table...")
				buildIdMap(idMap)
			}
			next.ServeHTTP(w, r)
		})
	}
}

func loggerMiddleware(next http.Handler) http.HandlerFunc {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		log.Printf("%s %s?%s", r.Method, r.URL.Path, r.URL.RawQuery)
		next.ServeHTTP(w, r)
	})
}
