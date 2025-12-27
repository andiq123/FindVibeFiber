# FindVibeFiber API Documentation

Simple and efficient API reference for the frontend.

## Base URL
`http://localhost:8080` (Local)

---

## 🔐 Authentication

### Authenticate / Register User
`GET /:username`

Returns the user object. If the user doesn't exist, it is automatically created.

**Response (200 OK):**
```json
{
  "id": "ada43644-b179-46a7-92fd-118a6ef59833",
  "name": "andiq"
}
```

---

## 🎵 Search & Suggestions

### Get Search Suggestions
`GET /suggest?q={query}`

Returns a list of strings for search autocomplete. Hardcoded for Romania/YouTube context.

**Response (200 OK):**
```json
["pizza tower", "pizza tower ost", "pizza song"]
```

### Search Music
`GET /search?q={query}`

Scrapes and returns music results.

**Response (200 OK):**
```json
[
  {
    "id": "6fddb330-8ccb-4017-9620-40bcf105905b",
    "title": "Song Title",
    "artist": "Artist Name",
    "image": "https://...",
    "link": "https://..."
  }
]
```

---

## ⭐ Favorites

### Get User Favorites
`GET /favorites/:userId`

**Response (200 OK):**
```json
[
  {
    "id": "6fddb330-8ccb-4017-9620-40bcf105905b",
    "title": "Song Title",
    "artist": "Artist Name",
    "image": "https://...",
    "link": "https://...",
    "userId": "ada436... "
  }
]
```

### Add to Favorites
`POST /favorites/:userId`

**Request Body:**
```json
{
  "id": "unique-song-id",
  "title": "Song Title",
  "artist": "Artist Name",
  "image": "https://...",
  "link": "https://..."
}
```

**Responses:**
- `200 OK`: `{"message": "song added"}`
- `409 Conflict`: `{"error": "resource already exists"}` (Song already in favorites)

### Remove from Favorites
`DELETE /favorites/:songId`

**Response (200 OK):**
```json
{"message": "song deleted"}
```

### Reorder Favorites
`PUT /favorites`

**Request Body:**
```json
[
  { "songId": "song-1", "order": 1 },
  { "songId": "song-2", "order": 2 }
]
```

---

## 💓 Utility

### Health Check
`GET /health`

**Response (200 OK):**
```json
{"status": "ok"}
```
