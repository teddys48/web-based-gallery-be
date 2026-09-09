# Web Gallery Backend (Go Fiber + SQLite)

Backend RESTful API berkinerja tinggi untuk aplikasi Web Gallery dengan fitur **Optimasi Drive Skala Besar (1TB+ HDD)**, **Dual Worker Pool Scanner**, **Timeline View & Timeline Buckets**, **File-Manager Style Folder View**, **Group by Date pada Folder View**, **Get Media by Date API**, **Automatic Video Frame Extraction**, **EXIF Metadata Extractor**, **Thumbnail Generator dengan Atomic Rename & Validasi Dekode**, **Browser Caching (ETag & Cache-Control)**, dan **Pagination**.

---

## 🚀 Fitur Utama

- **Go Fiber Framework**: Web framework ultra-cepat dan ringan berbasis Fasthttp.
- **SQLite Database (WAL Mode)**: Menggunakan driver pure-Go (`glebarez/sqlite`) dengan `PRAGMA journal_mode=WAL;` dan `PRAGMA busy_timeout=5000;` untuk concurrency tinggi tanpa CGO.
- **Optimasi Drive Skala Besar (1TB+ HDD & CPU Saver)**:
  - **Early Metadata Filter**: Cek `file_size` & `mod_time` dari memori DB sebelum memproses file (0 I/O & 0 CPU untuk file unchanged).
  - **Dual Worker Pools**: Terpisah untuk `ImageScanWorkers` (foto) dan `VideoScanWorkers` (video).
  - **FFmpeg Thread Limiter**: Pembatasan thread FFmpeg (`FFMPEG_THREADS=1`) agar pemrosesan video tidak menghabiskan seluruh core CPU.
  - **Fast File Hashing**: Hashing instan `sha256(path + size + modtime)` tanpa membaca 1 MB file dari HDD.
  - **Single DB Collector**: Goroutine kolektor khusus untuk menulis hasil scan ke SQLite secara aman tanpa lock contention.
- **Thumbnail Engine & Validation**:
  - **Atomic File Rename**: Penulisan file thumbnail ke `.tmp.jpg` sebelum di-rename secara atomis ke `.jpg`.
  - **Deep Image Validation**: Memeriksa integritas header file thumbnail via `image.DecodeConfig()`. File korup otomatis terdeteksi dan dibersihkan.
  - **On-Demand Generation**: Jika thumbnail hilang di disk, server otomatis membuatnya ulang secara langsung saat diakses.
- **Dukungan Foto & Video**:
  - **Foto**: `.jpg`, `.jpeg`, `.png`, `.webp`, `.gif`, `.heic`.
  - **Video**: `.mp4`, `.mkv`, `.mov`, `.avi`, `.webm`, `.m4v`, `.flv`, `.3gp`, `.ts`, `.wmv`.
- **Get Media by Date API**: Rute fleksibel untuk memfilter foto/video berdasarkan tanggal tunggal (`YYYY-MM-DD`), rentang tanggal (`start_date` & `end_date`), tahun & bulan (`year` & `month`), atau dikelompokkan per tanggal (`by-date/grouped`).
- **Group by Date pada Folder View**: Item media di dalam Folder View dikelompokkan secara otomatis berdasarkan tanggal (`date_groups` dalam format `YYYY-MM-DD`).
- **Automatic Video Thumbnail & Duration Extraction**: Menggunakan `ffmpeg` untuk mengekstrak thumbnail frame video asli (dengan fallback 3-tier) dan `ffprobe` untuk menghitung `duration` (dalam detik), `width`, dan `height` dalam satu panggilan.
- **Folder Cover Thumbnails**: Setiap folder secara otomatis memiliki thumbnail preview (`thumbnail_path`, `thumbnail_url`, dan `cover_photo_id`) yang diambil dari media terbaru di dalam folder tersebut.
- **File Manager Folder View**: Menelusuri direktori secara interaktif per folder (`GET /api/v1/folders/contents?folder_path=...`) yang hanya menampilkan folder anak langsung dan media di dalam folder tersebut.
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

# Scanner & Worker Pool Settings (Optimasi CPU & Drive)
MAX_SCAN_WORKERS=4
IMAGE_SCAN_WORKERS=2
VIDEO_SCAN_WORKERS=1
FFMPEG_THREADS=1
```

### Penjelasan Opsi Tuning Performance:
- `IMAGE_SCAN_WORKERS`: Jumlah worker paralel untuk ekstraksi EXIF & thumbnail foto (default: `2`).
- `VIDEO_SCAN_WORKERS`: Jumlah worker paralel untuk ekstraksi frame video (default: `1`, disarankan 1 untuk membatasi I/O HDD & CPU).
- `FFMPEG_THREADS`: Jumlah thread CPU per eksekusi perintah FFmpeg (default: `1`).

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

### 3. Timeline API

#### A. Get Timeline Photos (Paginated)
- **Endpoint**: `GET /api/v1/photos/timeline?page={page}&limit={limit}`
- **Query Parameters**:
  - `page` (int, default `1`): Nomor halaman.
  - `limit` (int, default `50`): Jumlah foto per halaman.
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

#### B. Timeline Buckets (Agregasi Tahun & Bulan)
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

#### B. Full Folder Tree (Tree View untuk Sidebar Navigation)
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

---

### 5. Get Media by Date API

#### A. Filter Media Berdasarkan Tanggal (Paginated)
Mengambil daftar foto/video berdasarkan filter tanggal spesifik (`date`), rentang tanggal (`start_date` & `end_date`), tahun & bulan (`year` & `month`), serta jenis media (`media_type`).

- **Endpoint**: `GET /api/v1/photos/by-date`
- **Query Parameters**:
  - `date` (string, opsional): Tanggal spesifik `YYYY-MM-DD` (contoh: `2026-09-08`) atau `YYYY-MM` (contoh: `2026-09`).
  - `start_date` (string, opsional): Tanggal awal rentang `YYYY-MM-DD` (contoh: `2026-09-01`).
  - `end_date` (string, opsional): Tanggal akhir rentang `YYYY-MM-DD` (contoh: `2026-09-30`).
  - `year` (int, opsional): Filter tahun (contoh: `2026`).
  - `month` (int, opsional): Filter bulan `1-12` (contoh: `9`).
  - `media_type` (string, opsional): Filter tipe media (`image` atau `video`).
  - `page` (int, default `1`): Nomor halaman.
  - `limit` (int, default `50`): Jumlah item per halaman.
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

#### B. Filter Media Berdasarkan Tanggal (Grouped by Date)
Mengambil daftar media yang **dikelompokkan per tanggal** (`YYYY-MM-DD`) berdasarkan filter tanggal atau tipe media.

- **Endpoint**: `GET /api/v1/photos/by-date/grouped`
- **Query Parameters**:
  - `date`, `start_date`, `end_date`, `year`, `month`, `media_type` (sama seperti di atas).
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

#### A. Detail Metadata Foto/Video
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

#### B. Stream Raw Media / Pemutaran Video
- **Endpoint**: `GET /api/v1/photos/:id/raw`
- **Response Headers**:
  - `Content-Type`: `image/jpeg` atau `video/mp4`
  - `Accept-Ranges`: `bytes`
  - `Cache-Control`: `public, max-age=86400`
  - `ETag`: `"media/Events/cake.jpg-size-modtime"`

---

### 7. Thumbnail Service API

#### A. Stream Thumbnail Berdasarkan Photo ID (Auto-Generate On-Demand)
Mengambil file thumbnail berdasarkan Photo ID. Jika file thumbnail hilang dari disk, server akan **otomatis membuat ulang (*on-demand*)**.

- **Endpoint**: `GET /api/v1/photos/:id/thumbnail` atau `GET /api/v1/thumbnails/:id`
- **Response Headers**:
  - `Content-Type`: `image/jpeg`
  - `Cache-Control`: `public, max-age=31536000, immutable`
  - `ETag`: `"thumb_xxx.jpg-size-modtime"`
- **HTTP Status**: `200 OK` (atau `304 Not Modified` jika ETag cocok).

#### B. Stream Thumbnail Berdasarkan Path Media
Mengambil atau membuat thumbnail untuk media apapun berdasarkan query parameter `path`.

- **Endpoint**: `GET /api/v1/thumbnails?path={filePath}`
- **Query Parameter**:
  - `path` (string, wajib): Relative path berkas foto atau video (contoh: `media/Vacation2025/beach.jpg` atau `media/sample_video.mp4`).
- **Response Headers**: `Content-Type: image/jpeg`, `Cache-Control`, `ETag`.

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
