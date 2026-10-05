# 🐛 Task API — Debugging Challenge

REST API sederhana (Task Manager) ditulis dengan Go memakai struktur **clean code**
(domain → repository → service → handler, dependency injection via constructor).

**TAPI: kode ini sengaja ditanami beberapa bug** yang umum ditemui di proyek dunia nyata —
termasuk bug yang mengganggu performa. Tugasmu: temukan dan perbaiki semuanya. 😈

## Menjalankan

```bash
go build ./...
go run ./cmd/server        # listen di :8080
```

## Endpoint

| Method | Path                          | Keterangan                    |
|--------|-------------------------------|-------------------------------|
| GET    | /health                       | healthcheck                   |
| POST   | /api/v1/tasks                 | buat task                     |
| GET    | /api/v1/tasks                 | list (?status=&q=&offset=&limit=) |
| GET    | /api/v1/tasks/{id}            | detail task                   |
| PUT    | /api/v1/tasks/{id}            | update task                   |
| DELETE | /api/v1/tasks/{id}            | hapus task                    |

Contoh:
```bash
curl -X POST localhost:8080/api/v1/tasks -d '{"title":"belajar debug","priority":3}'
curl localhost:8080/api/v1/tasks/1
curl -X PUT localhost:8080/api/v1/tasks/1 -d '{"status":"doing"}'
```

## Struktur

```
cmd/server/main.go                # bootstrap & graceful shutdown
internal/domain/task.go           # entitas + error domain
internal/repository/task_repo.go  # interface + memory implementation
internal/service/task_service.go  # business logic + Notifier
internal/handler/task_handler.go  # HTTP layer
internal/middleware/middleware.go # RequestID, Logging, Recoverer
internal/cache/cache.go           # in-memory TTL cache
```

## Petunjuk (tanpa spoiler!)

Ada **±9 bug**. Beberapa kategori yang perlu kamu curigai:

1. 🔥 **Bug fatal**: ada request yang bisa membuat **seluruh proses server mati total**
   (bukan cuma 500 — cek apa yang terjadi pada goroutine di `service`, dan kenapa
   middleware `Recoverer` tidak menyelamatkannya).
2. 👻 **Stale cache**: coba `PUT` lalu langsung `GET` id yang sama — datamu "hilang" / kembali ke versi lama.
3. 💥 **Input jahat**: mainkan query `?limit=` dan body JSON dengan tipe data salah
   (`{"priority":"abc"}`), lalu lihat isi log server.
4. 🏎️ **Performa**: salah satu fungsi di `repository` mengunci resource secara berlebihan
   sehingga request concurrent melambat drastis. Buktikan pakai `go test -race` atau benchmark.
5. 🧵 **Data race**: ada counter global di `middleware` yang tidak aman untuk konkurensi. Jalankan
   `go test -race ./...` sambil menembak endpoint dengan banyak request paralel
   (`ab -n 1000 -c 50` atau `hey`).
6. 🚰 **Memory leak**: goroutine "pembersih" di `cache` tidak pernah benar-benar berhenti
   meskipun sudah ada method `Stop()` — ticker-nya dibiarkan berjalan selamanya.
7. 🌳 **Context diabaikan**: perhatikan `ctx` yang diterima lalu dibuang/diganti
   (lihat `Notifier`) — ini membunuh cancellation & timeout di produksi.
8. 🧟 **Zombie rows**: setelah `DELETE`, beberapa operasi masih bisa "melihat" data lama.
9. ⌨️ **Typo** di konstanta domain yang bikin pusing saat dibaca.

Alat yang membantu: `go vet ./...`, `gofmt -l .`, `go test -race`, `pprof`,
dan membaca log dengan teliti.

Semangat debugging! 🕵️
