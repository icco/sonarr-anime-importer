# sonarr-mal-importer
This is basically a wrapper for [Jikan](jikan.moe) that converts a Jikan API call to a list with TVDB IDs that Sonarr can import the results.

**This API will spam calls that have pagination so make sure you set a limit in the query parameters so you don't get rate limited or IP banned!!**

Pulls MyAnimeList and TVDB ID associations from https://raw.githubusercontent.com/Kometa-Team/Anime-IDs/master/anime_ids.json.

## Supported Requests
### GET /anime
See https://docs.api.jikan.moe/#tag/anime/operation/getAnimeSearch for parameters.