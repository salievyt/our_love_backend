package media

import (
	"bufio"
	"bytes"
	"crypto/rand"
	"encoding/binary"
	"encoding/hex"
	"errors"
	"fmt"
	"image"
	_ "image/gif" // форматы для image.DecodeConfig
	_ "image/jpeg"
	_ "image/png"
	"io"
	"mime/multipart"
	"os"
	"path/filepath"
	"strings"
	"unicode/utf8"
)

const (
	// MaxImageMB — MAX_IMAGE_MB из config/uploads.py.
	MaxImageMB = 15
	// MaxImageBytes — граница MaxFileSizeValidator: больше — ошибка.
	MaxImageBytes = MaxImageMB * 1024 * 1024
	// maxNameLen — max_length=100 у ImageField: DRF проверяет имя файла клиента.
	maxNameLen = 100
	// maxPixels — порог DecompressionBombError в Pillow (2 × MAX_IMAGE_PIXELS):
	// такие картинки Django тоже отклоняет как невалидные.
	maxPixels = 2 * 89_478_485
)

// imageExts — расширения, которые принимает Go. Django берёт список у Pillow
// (validate_image_file_extension) — он шире, но клиенты шлют только эти форматы.
var imageExts = []string{"jpg", "jpeg", "jpe", "jfif", "png", "apng", "gif", "webp"}

// Сообщения DRF/Django (ru).
const (
	msgInvalidImage = "Загрузите правильное изображение. Файл, который вы загрузили, поврежден или не является изображением."
	msgEmpty        = "Отправленный файл пуст."
	msgNoName       = "Невозможно определить имя файла."
)

// ValidateImage проверяет загрузку как ImageField + IMAGE_VALIDATORS в Django и
// возвращает сообщения об ошибках (пусто — файл годится). Порядок как в DRF:
// сначала сам файл (имя, пустота, картинка ли), и только если он годится —
// валидаторы модели (расширение и размер, оба сразу).
func ValidateImage(filename string, size int64, content io.Reader) []string {
	if filename == "" {
		return []string{msgNoName}
	}
	if size == 0 {
		return []string{msgEmpty}
	}
	if n := utf8.RuneCountInString(filename); n > maxNameLen {
		return []string{fmt.Sprintf("Убедитесь, что это имя файла содержит не более %d символов (сейчас %d).", maxNameLen, n)}
	}
	if !isImage(content) {
		return []string{msgInvalidImage}
	}

	var errs []string
	if ext := extension(filename); !allowedExt(ext) {
		errs = append(errs, fmt.Sprintf("Расширение файлов “%s” не поддерживается. Разрешенные расширения: %s.",
			ext, strings.Join(imageExts, ", ")))
	}
	if size > MaxImageBytes {
		errs = append(errs, fmt.Sprintf("Файл больше %d МБ", MaxImageMB))
	}
	return errs
}

func allowedExt(ext string) bool {
	for _, e := range imageExts {
		if ext == e {
			return true
		}
	}
	return false
}

// extension — Path(name).suffix[1:].lower(), как FileExtensionValidator.
func extension(filename string) string {
	return strings.TrimPrefix(splitExt(filename), ".")
}

// splitExt — os.path.splitext(basename)[1].lower(): ведущие точки
// (".png") расширением не считаются.
func splitExt(filename string) string {
	base := filename[strings.LastIndexAny(filename, `/\`)+1:]
	dot := strings.LastIndexByte(base, '.')
	if dot <= 0 || strings.Trim(base[:dot], ".") == "" {
		return ""
	}
	return strings.ToLower(base[dot:])
}

// isImage читает только заголовок: JPEG, PNG и GIF разбирает stdlib
// (image.DecodeConfig), WebP — по структуре контейнера RIFF (stdlib его не умеет).
func isImage(content io.Reader) bool {
	br := bufio.NewReader(content)
	if head, _ := br.Peek(30); len(head) >= 12 && string(head[:4]) == "RIFF" && string(head[8:12]) == "WEBP" {
		w, h, ok := webpSize(head)
		return ok && w*h <= maxPixels
	}
	cfg, _, err := image.DecodeConfig(br)
	return err == nil && cfg.Width > 0 && cfg.Height > 0 && cfg.Width*cfg.Height <= maxPixels
}

// webpSize разбирает размеры из первого чанка WebP (VP8, VP8L или VP8X).
func webpSize(b []byte) (w, h int, ok bool) {
	if len(b) < 30 {
		return 0, 0, false
	}
	switch string(b[12:16]) {
	case "VP8 ": // lossy: ключевой кадр, стартовый код 9d 01 2a
		if b[20]&1 != 0 || !bytes.Equal(b[23:26], []byte{0x9d, 0x01, 0x2a}) {
			return 0, 0, false
		}
		w = int(binary.LittleEndian.Uint16(b[26:28]) & 0x3fff)
		h = int(binary.LittleEndian.Uint16(b[28:30]) & 0x3fff)
	case "VP8L": // lossless: сигнатура 0x2f, затем 14 бит ширины−1 и высоты−1
		if b[20] != 0x2f {
			return 0, 0, false
		}
		bits := binary.LittleEndian.Uint32(b[21:25])
		w = int(bits&0x3fff) + 1
		h = int(bits>>14&0x3fff) + 1
	case "VP8X": // расширенный: 24 бита ширины−1 и высоты−1
		w = int(uint32(b[24])|uint32(b[25])<<8|uint32(b[26])<<16) + 1
		h = int(uint32(b[27])|uint32(b[28])<<8|uint32(b[29])<<16) + 1
	default:
		return 0, 0, false
	}
	return w, h, w > 0 && h > 0
}

// ValidateFile — ValidateImage для части multipart.
func ValidateFile(fh *multipart.FileHeader) ([]string, error) {
	f, err := fh.Open()
	if err != nil {
		return nil, err
	}
	defer f.Close()
	return ValidateImage(fh.Filename, fh.Size, f), nil
}

// Storage — файловое хранилище в MEDIA_ROOT (как FileSystemStorage в Django).
type Storage struct {
	Root string
}

// Name — путь как UploadTo(prefix): "<prefix>/<uuid4 hex><ext>". Имя от клиента
// не используется (клиенты всегда шлют "photo.jpg"), берётся только расширение.
func Name(prefix, filename string) string {
	ext := []rune(splitExt(filename)) // [:10] в Python режет символы, не байты
	if len(ext) > 10 {
		ext = ext[:10]
	}
	return strings.Trim(prefix, "/") + "/" + uuid4Hex() + string(ext)
}

func uuid4Hex() string {
	var b [16]byte
	_, _ = rand.Read(b[:]) // crypto/rand.Read не возвращает ошибок
	b[6] = b[6]&0x0f | 0x40
	b[8] = b[8]&0x3f | 0x80
	return hex.EncodeToString(b[:])
}

// Save сохраняет уже проверенный файл и возвращает имя для колонки ImageField.
func (s Storage) Save(prefix string, fh *multipart.FileHeader) (string, error) {
	src, err := fh.Open()
	if err != nil {
		return "", err
	}
	defer src.Close()

	name := Name(prefix, fh.Filename)
	path := filepath.Join(s.Root, filepath.FromSlash(name))
	if err := os.MkdirAll(filepath.Dir(path), 0o755); err != nil {
		return "", fmt.Errorf("media dir: %w", err)
	}
	// O_EXCL: никогда не перезаписываем чужой файл (FileSystemStorage тоже)
	dst, err := os.OpenFile(path, os.O_WRONLY|os.O_CREATE|os.O_EXCL, 0o644)
	if err != nil {
		return "", fmt.Errorf("media create: %w", err)
	}
	if _, err := io.Copy(dst, src); err != nil {
		dst.Close()
		os.Remove(path)
		return "", fmt.Errorf("media write: %w", err)
	}
	if err := dst.Close(); err != nil {
		os.Remove(path)
		return "", fmt.Errorf("media write: %w", err)
	}
	return name, nil
}

// Remove удаляет файл, сохранённый Save (откат, если запись в базу не удалась).
func (s Storage) Remove(name string) error {
	if name == "" || strings.Contains(name, "..") {
		return errors.New("media: bad name")
	}
	return os.Remove(filepath.Join(s.Root, filepath.FromSlash(name)))
}
