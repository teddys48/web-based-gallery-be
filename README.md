# Web Gallery Backend (Go Fiber + SQLite)

High-performance RESTful API backend for Web Gallery applications featuring **Large-Scale Drive Optimizations (1TB+ HDD)**, **Dual Worker Pool Scanner**, **Timeline View & Timeline Buckets**, **File-Manager Style Folder View**, **Group by Date in Folder View**, **Get Media by Date API**, **Automatic Video Frame Extraction**, **EXIF Metadata Extractor**, **Thumbnail Generator with Atomic Rename & Decode Validation**, **Browser Caching (ETag & Cache-Control)**, **Pagination**, and **Real-Time Streaming Folder ZIP Downloads**.

---

## 🚀 Key Features

- **Go Fiber Framework**: Ultra-fast, lightweight web framework built on Fasthttp.
- **SQLite Database (WAL Mode)**: Pure-Go driver (`glebarez/sqlite`) with `PRAGMA journal_mode=WAL;` and `PRAGMA busy_timeout=5000;` for high concurrency without CGO.
- **Large-Scale Drive Optimizations (1TB+ HDD & CPU Saver)**:
  - **Early Metadata Filter**: Checks `file_size` & `mod_time` against in-memory DB records before scanning (0 I/O & 0 CPU overhead for unchanged files).
  - **Dual Worker Pools**: Isolated worker pools for `ImageScanWorkers` (photos) and `VideoScanWorkers` (videos).
  - **FFmpeg Thread Limiter**: Restricts FFmpeg CPU usage (`FFMPEG_THREADS=1`) to prevent video frame extraction from saturating CPU cores.
  - **Fast File Hashing**: Instant `sha256(path + size + modtime)` hashing without reading 1 MB chunks from slow HDDs.
  - **Single DB Collector**: Dedicated collector goroutine for safe SQLite batch writes without database lock contention.
- **Thumbnail Engine & Validation**:
  - **Atomic File Rename**: Writes temporary thumbnail files (`.tmp.jpg`) before atomically renaming to `.jpg`.
  - **Deep Image Validation**: Validates thumbnail header integrity via `image.DecodeConfig()`. Corrupt thumbnail files are automatically detected and purged.
  - **On-Demand Generation**: Auto-regenerates missing thumbnails on disk immediately upon HTTP request.
- **Photo & Video Support**:
  - **Photos**: `.jpg`, `.jpeg`, `.png`, `.webp`, `.gif`, `.heic`.
  - **Videos**: `.mp4`, `.mkv`, `.mov`, `.avi`, `.webm`, `.m4v`, `.flv`, `.3gp`, `.ts`, `.wmv`.
- **Get Media by Date API**: Flexible endpoints to filter photos/videos by single date (`YYYY-MM-DD`), date ranges (`start_date` & `end_date`), year & month (`year` & `month`), or aggregated per date (`by-date/grouped`).
- **Group by Date in Folder View**: Media items within the Folder View are automatically grouped by date (`date_groups` in `YYYY-MM-DD` format).
- **Automatic Video Thumbnail & Duration Extraction**: Uses `ffmpeg` for extracting authentic video frame thumbnails (with 3-tier fallback) and `ffprobe` for extracting `duration` (in seconds), `width`, and `height` in a single execution.
- **Folder Cover Thumbnails**: Automatically retrieves thumbnail previews (`thumbnail_path`, `thumbnail_url`, and `cover_photo_id`) from the newest media item inside each folder.
- **File Manager Folder View**: Interactive folder-by-folder directory browsing (`GET /api/v1/folders/contents?folder_path=...`) displaying only direct child subfolders and files.
- **EXIF Metadata Extractor**: Extracts `DateTimeOriginal`, dimensions (`Width` & `Height`), camera make/model (`CameraMake`, `CameraModel`), `FNumber`, `ExposureTime`, `ISO`, `FocalLength`, and GPS coordinates (`Latitude`, `Longitude`).
- **Real-Time Streaming Folder ZIP Downloads**: Download any directory and all its subdirectories recursively as a `.zip` archive using HTTP chunked streaming (`io.Pipe()`) without creating temporary files on disk (read-only safe) and with $O(1)$ memory usage.
- **Browser Caching & Video Streaming**: Leverages `Cache-Control` & `ETag` headers, supports HTTP `304 Not Modified`, and enables HTTP `Range` request streaming for video playback.

---

## 🛠️ Environment Configuration (`.env`)

The application reads environment variables from the `.env` file:

```env
# Server Configuration
PORT=8080

# Storage & Directory Settings
MEDIA_DIR=./media
THUMBNAIL_DIR=./.thumbnails
DB_PATH=./gallery.db

# Thumbnail Generation Settings
THUMB_WIDTH=400
THUMB_HEIGHT=400

# Scanner & Worker Pool Settings (CPU & Drive Optimizations)
MAX_SCAN_WORKERS=4
IMAGE_SCAN_WORKERS=2
VIDEO_SCAN_WORKERS=1
FFMPEG_THREADS=1
```

### Performance Tuning Options Explained:

- `IMAGE_SCAN_WORKERS`: Number of parallel workers for photo EXIF extraction & thumbnail generation (default: `2`).
- `VIDEO_SCAN_WORKERS`: Number of parallel workers for video frame extraction (default: `1`, recommended `1` to limit HDD I/O & CPU load).
- `FFMPEG_THREADS`: Number of CPU threads per FFmpeg process execution (default: `1`).

---

## 📡 API Reference & JSON Examples

### 1. Health Check

- **Endpoint**: `GET /health`
- **Response**:

```json
{
  "status": "ok",
  "time": "2026-09-08T14:11:44+07:00"
}
```

---

### 2. Scanner API

#### A. Trigger Manual Scan

- **Endpoint**: `POST /api/v1/scan/start`
- **Response**: `202 Accepted`

```json
{
  "success": true,
  "message": "Scan initiated in background",
  "data": {
    "is_scanning": true,
    "scanned_count": 0,
    "new_count": 0,
    "updated_count": 0,
    "error_count": 0,
    "total_found": 0,
    "started_at": "2026-09-08T14:11:33.503+07:00"
  }
}
```

#### B. Check Scan Status

- **Endpoint**: `GET /api/v1/scan/status`
- **Response**: `200 OK`

```json
{
  "success": true,
  "message": "Scan status retrieved successfully",
  "data": {
    "is_scanning": false,
    "scanned_count": 7,
    "new_count": 7,
    "updated_count": 0,
    "error_count": 0,
    "total_found": 7,
    "started_at": "2026-09-08T14:11:33.503+07:00",
    "finished_at": "2026-09-08T14:11:33.589+07:00"
  }
}
```

---

### 3. Timeline API

#### A. Get Timeline Photos (Paginated)

- **Endpoint**: `GET /api/v1/photos/timeline?page={page}&limit={limit}`
- **Query Parameters**:
  - `page` (int, default `1`): Page number.
  - `limit` (int, default `50`): Number of photo items per page.
- **Response**:

```json
{
  "success": true,
  "data": [
    {
      "id": 1,
      "file_path": "media/Vacation2025/beach.jpg",
      "file_name": "beach.jpg",
      "folder_path": "media/Vacation2025",
      "file_size": 8196,
      "media_type": "image",
      "mime_type": "image/jpeg",
      "width": 800,
      "height": 600,
      "taken_at": "2026-09-08T14:10:32+07:00",
      "thumbnail_path": ".thumbnails/thumb_db6ff0423f01dba549246de0508adcd6.jpg"
    }
  ],
  "pagination": {
    "page": 1,
    "limit": 50,
    "total_items": 7,
    "total_pages": 1,
    "has_next": false,
    "has_prev": false
  }
}
```

#### B. Timeline Buckets (Year & Month Aggregations)

- **Endpoint**: `GET /api/v1/photos/timeline/buckets`
- **Response**:

```json
{
  "success": true,
  "message": "Timeline buckets retrieved successfully",
  "data": [
    {
      "year": 2026,
      "month": 9,
      "count": 7
    }
  ]
}
```

---

### 4. File-Manager Folder View API (With Group by Date)

#### A. Folder Contents per Level (File Manager View)

When a user clicks on a folder, this endpoint returns:

- `sub_folders`: Direct child subfolders inside the target directory (with `thumbnail_url` & `cover_photo_id`).
- `date_groups`: Media items within the folder **grouped by date** (`date`, `count`, `photos`).
- `photos`: Flat list of all media items inside the folder.

- **Endpoint**: `GET /api/v1/folders/contents?folder_path={path}&page={page}&limit={limit}`
- **Query Parameters**:
  - `folder_path` (string, optional): Target folder path to open (e.g. `media` or `media/Events`). Opens root folder if empty.
  - `page` (int, default `1`): Page number for photos.
  - `limit` (int, default `50`): Number of photo items per page.
- **Request Example**: `GET /api/v1/folders/contents?folder_path=media/Events&page=1&limit=2`
- **Response**:

```json
{
  "success": true,
  "data": {
    "current_folder": "media/Events",
    "parent_folder": "media",
    "sub_folders": [
      {
        "name": "Party",
        "path": "media/Events/Party",
        "photo_count": 2,
        "thumbnail_url": "/api/v1/photos/2/thumbnail",
        "thumbnail_path": ".thumbnails/thumb_831df940749a5d4311345aec035dd703.jpg",
        "cover_photo_id": 2
      }
    ],
    "date_groups": [
      {
        "date": "2026-09-08",
        "count": 1,
        "photos": [
          {
            "id": 1,
            "file_path": "media/Events/cake.jpg",
            "file_name": "cake.jpg",
            "folder_path": "media/Events",
            "file_size": 8196,
            "media_type": "image",
            "taken_at": "2026-09-08T14:10:32+07:00",
            "thumbnail_path": ".thumbnails/thumb_a4589017c1682f1620a688bcdc819e72.jpg"
          }
        ]
      }
    ],
    "photos": [
      {
        "id": 1,
        "file_path": "media/Events/cake.jpg",
        "file_name": "cake.jpg",
        "folder_path": "media/Events",
        "file_size": 8196,
        "media_type": "image",
        "taken_at": "2026-09-08T14:10:32+07:00",
        "thumbnail_path": ".thumbnails/thumb_a4589017c1682f1620a688bcdc819e72.jpg"
      }
    ]
  },
  "pagination": {
    "page": 1,
    "limit": 2,
    "total_items": 1,
    "total_pages": 1,
    "has_next": false,
    "has_prev": false
  }
}
```

#### B. Full Folder Tree (Tree View for Sidebar Navigation)

- **Endpoint**: `GET /api/v1/folders/tree`
- **Response**:

```json
{
  "success": true,
  "message": "Folder tree retrieved successfully",
  "data": [
    {
      "name": "Events",
      "path": "media/Events",
      "photo_count": 0,
      "thumbnail_url": "/api/v1/photos/2/thumbnail",
      "thumbnail_path": ".thumbnails/thumb_831df940749a5d4311345aec035dd703.jpg",
      "cover_photo_id": 2,
      "sub_folders": [
        {
          "name": "Party",
          "path": "media/Events/Party",
          "photo_count": 2,
          "thumbnail_url": "/api/v1/photos/2/thumbnail",
          "thumbnail_path": ".thumbnails/thumb_831df940749a5d4311345aec035dd703.jpg",
          "cover_photo_id": 2
        }
      ]
    }
  ]
}
```

#### C. Stream Download Folder ZIP (Streaming ZIP Archive)

Downloads all files and subdirectories within a specified folder as a `.zip` archive. The compression process is streamed directly to the HTTP response (`io.Pipe()`) without loading entire files into RAM and without generating temporary files in `MEDIA_DIR`.

- **Endpoint**: `GET /api/v1/folders/download`
- **Query Parameters**:
  - `path` (string, optional): Relative folder path to download (e.g. `media/Events` or `media/Events/Party`). If omitted or `.`, downloads the entire root `MEDIA_DIR`.
- **Request Example**: `GET /api/v1/folders/download?path=media/Events`
- **Response Headers**:
  - `Content-Type`: `application/zip`
  - `Content-Disposition`: `attachment; filename="Events.zip"`
- **Response Body**: Binary ZIP byte stream.
- **Error Responses**:
  - `400 Bad Request`: Empty folder or folder containing no readable files.
  - `403 Forbidden`: Path traversal attempt (e.g. using `../` or symlinks pointing outside `MEDIA_DIR`).
  - `404 Not Found`: Folder path does not exist on disk.

---

### 5. Get Media by Date API

#### A. Filter Media by Date (Paginated)

Retrieves a list of photos/videos filtered by specific date (`date`), date range (`start_date` & `end_date`), year & month (`year` & `month`), or media type (`media_type`).

- **Endpoint**: `GET /api/v1/photos/by-date`
- **Query Parameters**:
  - `date` (string, optional): Specific date `YYYY-MM-DD` (e.g. `2026-09-08`) or month `YYYY-MM` (e.g. `2026-09`).
  - `start_date` (string, optional): Range start date `YYYY-MM-DD` (e.g. `2026-09-01`).
  - `end_date` (string, optional): Range end date `YYYY-MM-DD` (e.g. `2026-09-30`).
  - `year` (int, optional): Year filter (e.g. `2026`).
  - `month` (int, optional): Month filter `1-12` (e.g. `9`).
  - `media_type` (string, optional): Media type filter (`image` or `video`).
  - `page` (int, default `1`): Page number.
  - `limit` (int, default `50`): Item count per page.
- **Request Example**: `GET /api/v1/photos/by-date?date=2026-09-08&media_type=image&page=1&limit=50`
- **Response**:

```json
{
  "success": true,
  "data": [
    {
      "id": 1,
      "file_path": "media/Events/cake.jpg",
      "file_name": "cake.jpg",
      "folder_path": "media/Events",
      "file_size": 8196,
      "hash": "3982040b2cd491765ee670714d9abf66d1e903e0489845977b5da8fc81b87b75",
      "media_type": "image",
      "mime_type": "image/jpeg",
      "width": 800,
      "height": 600,
      "taken_at": "2026-09-08T14:10:32+07:00",
      "thumbnail_path": ".thumbnails/thumb_db6ff0423f01dba549246de0508adcd6.jpg"
    }
  ],
  "pagination": {
    "page": 1,
    "limit": 50,
    "total_items": 1,
    "total_pages": 1,
    "has_next": false,
    "has_prev": false
  }
}
```

#### B. Filter Media by Date (Grouped by Date)

Retrieves media items **grouped per date** (`YYYY-MM-DD`) based on date or media type filters.

- **Endpoint**: `GET /api/v1/photos/by-date/grouped`
- **Query Parameters**:
  - `date`, `start_date`, `end_date`, `year`, `month`, `media_type` (same as above).
- **Request Example**: `GET /api/v1/photos/by-date/grouped?year=2026&month=9`
- **Response**:

```json
{
  "success": true,
  "message": "Photos grouped by date retrieved successfully",
  "data": [
    {
      "date": "2026-09-08",
      "count": 7,
      "photos": [
        {
          "id": 1,
          "file_path": "media/Events/cake.jpg",
          "file_name": "cake.jpg",
          "media_type": "image",
          "taken_at": "2026-09-08T14:10:32+07:00"
        }
      ]
    }
  ]
}
```

---

### 6. Photo Detail & Raw Media Streaming API

#### A. Photo/Video Metadata Detail

- **Endpoint**: `GET /api/v1/photos/:id`
- **Response**:

```json
{
  "success": true,
  "message": "Photo retrieved successfully",
  "data": {
    "id": 1,
    "file_path": "media/Events/cake.jpg",
    "file_name": "cake.jpg",
    "folder_path": "media/Events",
    "file_size": 8196,
    "media_type": "image",
    "mime_type": "image/jpeg",
    "width": 800,
    "height": 600,
    "taken_at": "2026-09-08T14:10:32+07:00",
    "thumbnail_path": ".thumbnails/thumb_db6ff0423f01dba549246de0508adcd6.jpg"
  }
}
```

#### B. Stream Raw Media / Video Playback

- **Endpoint**: `GET /api/v1/photos/:id/raw`
- **Response Headers**:
  - `Content-Type`: `image/jpeg` or `video/mp4`
  - `Accept-Ranges`: `bytes`
  - `Cache-Control`: `public, max-age=86400`
  - `ETag`: `"media/Events/cake.jpg-size-modtime"`

---

### 7. Thumbnail Service API

#### A. Stream Thumbnail by Photo ID (Auto-Generate On-Demand)

Fetches the thumbnail file by Photo ID. If the thumbnail file is missing from disk, the server **automatically regenerates it on demand**.

- **Endpoint**: `GET /api/v1/photos/:id/thumbnail` or `GET /api/v1/thumbnails/:id`
- **Response Headers**:
  - `Content-Type`: `image/jpeg`
  - `Cache-Control`: `public, max-age=31536000, immutable`
  - `ETag`: `"thumb_xxx.jpg-size-modtime"`
- **HTTP Status**: `200 OK` (or `304 Not Modified` if ETag matches).

#### B. Stream Thumbnail by Media Path

Fetches or generates a thumbnail for any media item based on the `path` query parameter.

- **Endpoint**: `GET /api/v1/thumbnails?path={filePath}`
- **Query Parameter**:
  - `path` (string, required): Relative path to photo or video file (e.g. `media/Vacation2025/beach.jpg` or `media/sample_video.mp4`).
- **Response Headers**: `Content-Type: image/jpeg`, `Cache-Control`, `ETag`.

---

## 💻 Local Development Setup

```bash
# 1. Clone the repository & navigate into the folder
git clone https://github.com/user/gallery-be.git
cd gallery-be

# 2. Copy the example environment file
cp .env.example .env

# 3. Run the server
go run cmd/server/main.go
```

---

## 🐳 Running with Docker

```bash
# Start containers via Docker Compose
docker compose up -d

# View container logs
docker compose logs -f
```
