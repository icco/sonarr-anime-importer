# sonarr-mal-importer
This is basically a wrapper for [Jikan](jikan.moe) that converts a Jikan API call to a list with TVDB IDs that Sonarr can import the results.

**This API will spam calls that have pagination so make sure you set a limit in the query parameters so you don't get rate limited or IP banned!!**

Pulls MyAnimeList and TVDB ID associations from https://raw.githubusercontent.com/Kometa-Team/Anime-IDs/master/anime_ids.json.

## Supported Requests
### GET /anime
See https://docs.api.jikan.moe/#tag/anime/operation/getAnimeSearch for parameters.

Additional parameters supported:
- `allow_duplicates`: skips de-duplication of results

Example request:
```bash
# fetches the top 10 most popular currently airing tv anime
curl "http://localhost:3333/anime?type=tv&status=airing&order_by=popularity&sort=asc&limit=10"
```

## Environment
One configuration environment variable is supported:
- `ALWAYS_SKIP_MAL_IDS`: Comma-separated list of MyAnimeList IDs to always skip. These do not count towards the return limit.

## Docker Compose
```yaml
services:
  sonarr-mal-importer:
    image: gabehf/sonarr-mal-importer:latest
    container_name: sonarr-mal-importer
    ports:
      - 3333:3333
    environment:
      - ALWAYS_SKIP_MAL_IDS=12345,67890 # Comma-separated
    restart: unless-stopped

```

# TODO
- [x] Add de-duplication and a query param to disable it
- [x] Add perma-skip by MALId option in environment variable
- [ ] Only do "a.k.a." when logging if the anime has different romanized and english titles

# Albums that fueled development
| Album                   | Artist                       |
|-------------------------|------------------------------|
| ZOO!!                   | Necry Talkie (ネクライトーキー) |
| FREAK                   | Necry Talkie (ネクライトーキー) |
| Expert In A Dying Field | The Beths                    |
| Vivid                   | ADOY                         |