# Web Gallery Backend (Go Fiber + SQLite)

Backend RESTful API berkinerja tinggi untuk aplikasi Web Gallery dengan fitur **Timeline View**, **File-Manager Style Folder View**, **Foto & Video Scanner**, **EXIF Metadata & Video Duration Extractor**, **Thumbnail Generator**, **Browser Caching (ETag & Cache-Control)**, dan **Pagination**.

---

## 🚀 Fitur Utama

- **Go Fiber Framework**: Web framework ultra-cepat dan ringan.
- **SQLite Database (WAL Mode)**: Menggunakan driver pure-Go (`glebarez/sqlite`) dengan `PRAGMA journal_mode=WAL;` dan `PRAGMA busy_timeout=5000;` untuk concurrency tinggi tanpa CGO.
- **Dukungan Foto & Video**:
  - **Foto**: `.jpg`, `.jpeg`, `.png`, `.webp`, `.gif`, `.heic`.
  - **Video**: `.mp4`, `.mkv`, `.mov`, `.avi`, `.webm`, `.m4v`, `.flv`, `.3gp`, `.ts`, `.wmv`.
- **Automatic Video Thumbnail & Duration Extraction**: Menggunakan `ffmpeg` untuk mengekstrak thumbnail frame video dan `ffprobe` untuk menghitung `duration` (dalam detik).
- **Auto Environment Loading**: Otomatis membaca konfigurasi dari berkas `.env` saat aplikasi dinyalakan.
- **Folder Cover Thumbnails**: Setiap folder secara otomatis memiliki thumbnail preview (`thumbnail_path`, `thumbnail_url`, dan `cover_photo_id`) yang diambil dari media terbaru di dalam folder tersebut.
- **File Manager Folder View**: Menelusuri direktori secara interaktif per folder (`GET /api/v1/folders/contents?folder_path=...`) yang hanya menampilkan folder anak langsung dan media di dalam folder tersebut.
- **Media Scanner**: Walk directory secara rekursif, mendeteksi file baru/diperbarui, dan membersihkan media yang sudah dihapus dari disk secara otomatis.
- **EXIF Metadata Extractor**: Mengekstrak `DateTimeOriginal`, dimensi `Width` & `Height`, merk/model kamera (`CameraMake`, `CameraModel`), `FNumber`, `ExposureTime`, `ISO`, `FocalLength`, dan koordinat GPS (`Latitude`, `Longitude`).
- **Browser Caching & Video Streaming**: Menggunakan header `Cache-Control`, `ETag`, support HTTP `304 Not Modified`, serta support HTTP `Range` streaming untuk pemutaran video.

---

## 🛠️ Konfigurasi Environment (`.env`)

Aplikasi membaca variabel dari berkas `.env`:

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

# Scanner Settings
MAX_SCAN_WORKERS=4
```

---

## 📡 API Reference & JSON Examples

### 1. Health Check
- **Endpoint**: `GET /health`
- **Request**: `GET /health`
- **Response**:
```json
{
  "status": "ok",
  "time": "2026-09-08T14:11:44+07:00"
}
```

---

### 2. Scanner API

#### A. Memicu Manual Scan
- **Endpoint**: `POST /api/v1/scan/start`
- **Request**: `POST /api/v1/scan/start`
- **Response**:
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

#### B. Cek Status Scan
- **Endpoint**: `GET /api/v1/scan/status`
- **Request**: `GET /api/v1/scan/status`
- **Response**:
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

### 3. File-Manager Folder View API

#### A. Konten Folder Per Tingkatan (File Manager View)
- **Endpoint**: `GET /api/v1/folders/contents?folder_path={path}&page={page}&limit={limit}`
- **Request**: `GET /api/v1/folders/contents?folder_path=media&page=1&limit=2`
- **Response**:
```json
{
  "success": true,
  "data": {
    "current_folder": "media",
    "parent_folder": "",
    "sub_folders": [
      {
        "name": "Events",
        "path": "media/Events",
        "photo_count": 2,
        "thumbnail_url": "/api/v1/photos/2/thumbnail",
        "thumbnail_path": ".thumbnails/thumb_831df940749a5d4311345aec035dd703.jpg",
        "cover_photo_id": 2
      }
    ],
    "photos": [
      {
        "id": 7,
        "file_path": "media/sample_video.mp4",
        "file_name": "sample_video.mp4",
        "folder_path": "media",
        "file_size": 22284,
        "hash": "28399d05cdaefb8b388af883f3c5ebd43318bc704524fcda89b2b7da98826167",
        "media_type": "video",
        "mime_type": "video/mp4",
        "width": 640,
        "height": 480,
        "duration": 3.0,
        "taken_at": "2026-09-08T15:34:59+07:00",
        "thumbnail_path": ".thumbnails/thumb_726494a0ab1b58490ee61b9947227aae.jpg",
        "mod_time": "2026-09-08T15:34:59+07:00"
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

---

### 4. Timeline View API

#### A. Foto & Video Timeline (Paginated)
- **Endpoint**: `GET /api/v1/photos/timeline?page={page}&limit={limit}`
- **Response**:
```json
{
  "success": true,
  "data": [
    {
      "id": 7,
      "file_path": "media/sample_video.mp4",
      "file_name": "sample_video.mp4",
      "folder_path": "media",
      "file_size": 22284,
      "media_type": "video",
      "mime_type": "video/mp4",
      "width": 640,
      "height": 480,
      "duration": 3.0,
      "taken_at": "2026-09-08T15:34:59+07:00",
      "thumbnail_path": ".thumbnails/thumb_726494a0ab1b58490ee61b9947227aae.jpg"
    }
  ],
  "pagination": {
    "page": 1,
    "limit": 2,
    "total_items": 7,
    "total_pages": 4,
    "has_next": true,
    "has_prev": false
  }
}
```

---

### 5. Media Detail & Streaming API

#### A. Detail Metadata Foto/Video
- **Endpoint**: `GET /api/v1/photos/:id`
- **Request**: `GET /api/v1/photos/7`
- **Response**:
```json
{
  "success": true,
  "message": "Photo retrieved successfully",
  "data": {
    "id": 7,
    "file_path": "media/sample_video.mp4",
    "file_name": "sample_video.mp4",
    "folder_path": "media",
    "file_size": 22284,
    "media_type": "video",
    "mime_type": "video/mp4",
    "width": 640,
    "height": 480,
    "duration": 3.0,
    "taken_at": "2026-09-08T15:34:59+07:00",
    "thumbnail_path": ".thumbnails/thumb_726494a0ab1b58490ee61b9947227aae.jpg"
  }
}
```

#### B. Serve Raw Media / Video Stream (HTTP Range)
- **Endpoint**: `GET /api/v1/photos/:id/raw`
- **Response**: Stream file biner video (`video/mp4`, `video/webm`, dll.) dengan dukungan penuh HTTP Range (`206 Partial Content`) untuk pemutaran video / seeking di HTML5 `<video>` player.

---

## 💻 Cara Jalankan Lokal (Local Development)

```bash
# 1. Clone repository & masuk ke folder
git clone https://github.com/user/gallery-be.git
cd gallery-be

# 2. Salin file lingkungan
cp .env.example .env

# 3. Jalankan server
go run cmd/server/main.go
```

---

## 🐳 Cara Jalankan dengan Docker

```bash
# Jalankan container via Docker Compose
docker compose up -d

# Cek log container
docker compose logs -f
```
