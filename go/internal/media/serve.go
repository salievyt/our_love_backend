package media

import (
	"net/http"
	"os"
	"path"
	"path/filepath"
	"strings"
	"time"

	"github.com/salievyt/our_love_backend/go/internal/httpx"
)

// Типы, которые браузер может показать; всё остальное отдаётся как вложение,
// чтобы загруженный HTML не исполнился на нашем домене.
var inlineTypes = map[string]string{
	".jpg": "image/jpeg", ".jpeg": "image/jpeg", ".jpe": "image/jpeg", ".jfif": "image/jpeg",
	".png": "image/png", ".apng": "image/apng", ".gif": "image/gif", ".webp": "image/webp",
	".m4a": "audio/mp4", ".aac": "audio/aac", ".mp3": "audio/mpeg", ".wav": "audio/wav",
	".ogg": "audio/ogg", ".oga": "audio/ogg", ".opus": "audio/ogg", ".caf": "audio/x-caf", ".3gp": "audio/3gpp",
}

// Server — GET /media/{path...}: файл только по подписанной ссылке (как serve_media в Django).
//
// Accel=true (за nginx): ответ X-Accel-Redirect на internal-локацию /protected-media/,
// файл отдаёт nginx. Accel=false (локальная разработка без nginx): файл отдаёт сам Go.
type Server struct {
	Signer *Signer
	Root   string
	Accel  bool
}

func (s *Server) ServeHTTP(w http.ResponseWriter, r *http.Request) {
	name := r.PathValue("path")
	q := r.URL.Query()
	if name == "" || !s.Signer.Verify(name, q.Get("exp"), q.Get("sig"), time.Now()) {
		httpx.NotFound(w)
		return
	}

	h := w.Header()
	h.Set("X-Content-Type-Options", "nosniff")
	h.Set("Cache-Control", "private, max-age=3600")
	ctype, inline := inlineTypes[strings.ToLower(path.Ext(name))]
	if !inline {
		h.Set("Content-Disposition", "attachment")
	}

	if s.Accel {
		// Тип выставит nginx по расширению из своего белого списка
		h.Set("X-Accel-Redirect", "/protected-media/"+Quote(name))
		w.WriteHeader(http.StatusOK)
		return
	}

	full, ok := s.resolve(name)
	if !ok {
		httpx.NotFound(w)
		return
	}
	f, err := os.Open(full)
	if err != nil {
		httpx.NotFound(w)
		return
	}
	defer f.Close()
	info, err := f.Stat()
	if err != nil || info.IsDir() {
		httpx.NotFound(w)
		return
	}
	if !inline {
		ctype = "application/octet-stream"
	}
	h.Set("Content-Type", ctype)
	h.Set("Content-Security-Policy", "default-src 'none'; sandbox")
	http.ServeContent(w, r, "", info.ModTime(), f)
}

// resolve переводит имя из ссылки в путь внутри Root и не даёт выйти за его пределы.
func (s *Server) resolve(name string) (string, bool) {
	root, err := filepath.Abs(s.Root)
	if err != nil {
		return "", false
	}
	full := filepath.Join(root, filepath.FromSlash(name))
	rel, err := filepath.Rel(root, full)
	if err != nil || rel == ".." || strings.HasPrefix(rel, ".."+string(filepath.Separator)) {
		return "", false
	}
	return full, true
}
