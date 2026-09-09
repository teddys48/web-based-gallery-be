# Web Gallery Backend (Go Fiber + SQLite)

Backend RESTful API berkinerja tinggi untuk aplikasi Web Gallery dengan fitur **Timeline View**, **File-Manager Style Folder View**, **Group by Date pada Folder View**, **Foto & Video Scanner**, **EXIF Metadata & Video Duration Extractor**, **Thumbnail Generator**, **Browser Caching (ETag & Cache-Control)**, dan **Pagination**.

---

## 🚀 Fitur Utama

- **Go Fiber Framework**: Web framework ultra-cepat dan ringan.
- **SQLite Database (WAL Mode)**: Menggunakan driver pure-Go (`glebarez/sqlite`) dengan `PRAGMA journal_mode=WAL;` dan `PRAGMA busy_timeout=5000;` untuk concurrency tinggi tanpa CGO.
- **Dukungan Foto & Video**:
  - **Foto**: `.jpg`, `.jpeg`, `.png`, `.webp`, `.gif`, `.heic`.
  - **Video**: `.mp4`, `.mkv`, `.mov`, `.avi`, `.webm`, `.m4v`, `.flv`, `.3gp`, `.ts`, `.wmv`.
- **Group by Date pada Folder View**: Item media di dalam Folder View dikelompokkan secara otomatis berdasarkan tanggal (`date_groups` dalam format `YYYY-MM-DD`).
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

#### B. Cek Status Scan
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

### 3. File-Manager Folder View API (With Group by Date)

#### A. Konten Folder Per Tingkatan (File Manager View)
Ketika pengguna mengeklik suatu folder, endpoint ini mengembalikan:
- `sub_folders`: Sub-folder langsung di dalam folder tersebut (dengan `thumbnail_url` & `cover_photo_id`).
- `date_groups`: Media di dalam folder tersebut yang **dikelompokkan berdasarkan tanggal** (`date`, `count`, `photos`).
- `photos`: List datar seluruh media di dalam folder tersebut.

- **Endpoint**: `GET /api/v1/folders/contents?folder_path={path}&page={page}&limit={limit}`
- **Query Parameters**:
  - `folder_path` (string, opsional): Path folder yang ingin dibuka (contoh: `media` atau `media/Events`). Jika kosong, otomatis membuka root folder.
  - `page` (int, default `1`): Nomor halaman foto.
  - `limit` (int, default `50`): Jumlah foto per halaman.
- **Request**: `GET /api/v1/folders/contents?folder_path=media/Events&page=1&limit=2`
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
            "hash": "bb562ff2241e5a7bd7fcf08a86556b5c4b5111c8f272990472ea1cd630222f38",
            "media_type": "image",
            "mime_type": "image/jpeg",
            "width": 800,
            "height": 600,
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
        "mime_type": "image/jpeg",
        "width": 800,
        "height": 600,
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

#### B. Full Folder Tree (Tree View untuk Sidebar Navigation)
- **Endpoint**: `GET /api/v1/folders/tree`
- **Request**: `GET /api/v1/folders/tree`
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
