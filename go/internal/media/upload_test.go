package media

import (
	"bytes"
	"encoding/binary"
	"hash/crc32"
	"image"
	"image/color"
	"image/gif"
	"image/jpeg"
	"image/png"
	"mime/multipart"
	"net/http/httptest"
	"os"
	"path/filepath"
	"regexp"
	"strings"
	"testing"
)

func testImage() image.Image {
	img := image.NewRGBA(image.Rect(0, 0, 3, 2))
	img.Set(1, 1, color.RGBA{R: 255, A: 255})
	return img
}

func encoded(t *testing.T, enc func(*bytes.Buffer, image.Image) error) []byte {
	t.Helper()
	var buf bytes.Buffer
	if err := enc(&buf, testImage()); err != nil {
		t.Fatal(err)
	}
	return buf.Bytes()
}

// pngHeader — сигнатура и IHDR с заданными размерами (CRC верный).
func pngHeader(w, h uint32) []byte {
	ihdr := make([]byte, 13)
	binary.BigEndian.PutUint32(ihdr[0:], w)
	binary.BigEndian.PutUint32(ihdr[4:], h)
	ihdr[8], ihdr[9] = 8, 2 // 8 бит, RGB
	var b bytes.Buffer
	b.WriteString("\x89PNG\r\n\x1a\n")
	binary.Write(&b, binary.BigEndian, uint32(len(ihdr)))
	chunk := append([]byte("IHDR"), ihdr...)
	b.Write(chunk)
	binary.Write(&b, binary.BigEndian, crc32.ChecksumIEEE(chunk))
	return b.Bytes()
}

// webp собирает заголовок RIFF/WEBP с первым чанком kind и полезной нагрузкой.
func webp(kind string, payload []byte) []byte {
	var b bytes.Buffer
	b.WriteString("RIFF")
	binary.Write(&b, binary.LittleEndian, uint32(4+8+len(payload)))
	b.WriteString("WEBP" + kind)
	binary.Write(&b, binary.LittleEndian, uint32(len(payload)))
	b.Write(payload)
	return b.Bytes()
}

func TestValidateImage(t *testing.T) {
	pngData := encoded(t, func(b *bytes.Buffer, i image.Image) error { return png.Encode(b, i) })
	jpegData := encoded(t, func(b *bytes.Buffer, i image.Image) error { return jpeg.Encode(b, i, nil) })
	gifData := encoded(t, func(b *bytes.Buffer, i image.Image) error { return gif.Encode(b, i, nil) })
	// VP8L: сигнатура 0x2f, ширина−1 = 99, высота−1 = 49
	bits := uint32(99) | uint32(49)<<14
	vp8l := append([]byte{0x2f}, binary.LittleEndian.AppendUint32(nil, bits)...)
	webpLossless := webp("VP8L", append(vp8l, make([]byte, 8)...))
	// VP8: ключевой кадр, стартовый код, 640×480
	vp8 := []byte{0x00, 0x00, 0x00, 0x9d, 0x01, 0x2a, 0x80, 0x02, 0xe0, 0x01, 0, 0}
	webpLossy := webp("VP8 ", vp8)
	vp8x := webp("VP8X", []byte{0, 0, 0, 0, 0x3f, 0x01, 0x00, 0xef, 0x00, 0x00})
	heic := append([]byte{0, 0, 0, 0x18}, []byte("ftypheic\x00\x00\x00\x00mif1heic")...)

	const invalid = "Загрузите правильное изображение. Файл, который вы загрузили, поврежден или не является изображением."
	tests := []struct {
		desc     string
		filename string
		size     int64 // 0 — len(data)
		data     []byte
		want     []string
	}{
		{"png", "photo.png", 0, pngData, nil},
		{"jpeg with upper-case extension", "photo.JPG", 0, jpegData, nil},
		{"gif", "a.gif", 0, gifData, nil},
		{"png named .jpg (Pillow also accepts)", "photo.jpg", 0, pngData, nil},
		{"webp lossless", "photo.webp", 0, webpLossless, nil},
		{"webp lossy", "photo.webp", 0, webpLossy, nil},
		{"webp extended", "photo.webp", 0, vp8x, nil},
		{"exactly 15 MB", "photo.png", MaxImageBytes, pngData, nil},
		{"over 15 MB", "photo.png", MaxImageBytes + 1, pngData, []string{"Файл больше 15 МБ"}},
		{"text", "photo.png", 0, []byte("hello, not an image at all"), []string{invalid}},
		{"html disguised as image", "x.jpg", 0, []byte("<html><script>alert(1)</script></html>"), []string{invalid}},
		{"truncated png", "photo.png", 0, pngData[:20], []string{invalid}},
		{"heic is not supported (as in Django without pillow-heif)", "photo.heic", 0, heic, []string{invalid}},
		{"broken webp chunk", "photo.webp", 0, webp("VP8L", make([]byte, 16)), []string{invalid}},
		{"decompression bomb", "bomb.png", 0, pngHeader(20000, 20000), []string{invalid}},
		{"empty", "photo.png", 0, nil, []string{"Отправленный файл пуст."}},
		{"no filename", "", 0, pngData, []string{"Невозможно определить имя файла."}},
		{"long filename", strings.Repeat("я", 97) + ".png", 0, pngData,
			[]string{"Убедитесь, что это имя файла содержит не более 100 символов (сейчас 101)."}},
		{"wrong extension", "photo.svg", 0, pngData, []string{
			"Расширение файлов “svg” не поддерживается. Разрешенные расширения: jpg, jpeg, jpe, jfif, png, apng, gif, webp."}},
		{"no extension and too big: both model validators", "photo", MaxImageBytes + 1, pngData, []string{
			"Расширение файлов “” не поддерживается. Разрешенные расширения: jpg, jpeg, jpe, jfif, png, apng, gif, webp.",
			"Файл больше 15 МБ"}},
		{"dot file has no extension", ".png", 0, pngData, []string{
			"Расширение файлов “” не поддерживается. Разрешенные расширения: jpg, jpeg, jpe, jfif, png, apng, gif, webp."}},
	}
	for _, tt := range tests {
		t.Run(tt.desc, func(t *testing.T) {
			size := tt.size
			if size == 0 {
				size = int64(len(tt.data))
			}
			got := ValidateImage(tt.filename, size, bytes.NewReader(tt.data))
			if strings.Join(got, "|") != strings.Join(tt.want, "|") {
				t.Fatalf("got  %q\nwant %q", got, tt.want)
			}
		})
	}
}

func TestName(t *testing.T) {
	tests := []struct{ filename, ext string }{
		{"photo.JPG", ".jpg"},
		{"photo.jpeg", ".jpeg"},
		{"../../etc/passwd.png", ".png"},
		{`C:\Users\me\shot.PNG`, ".png"},
		{"archive.tar.gz", ".gz"},
		{".png", ""},
		{"noext", ""},
		{"x.abcdefghijklmn", ".abcdefghi"}, // [:10] вместе с точкой, как в UploadTo
	}
	re := regexp.MustCompile(`^places/[0-9a-f]{32}(\..*)?$`)
	seen := map[string]bool{}
	for _, tt := range tests {
		name := Name("/places/", tt.filename)
		if !re.MatchString(name) || !strings.HasSuffix(name, tt.ext) || len(name) != len("places/")+32+len(tt.ext) {
			t.Errorf("Name(%q) = %q, want ext %q", tt.filename, name, tt.ext)
		}
		if seen[name] {
			t.Errorf("duplicate name %s", name)
		}
		seen[name] = true
	}
}

func TestStorageSaveAndRemove(t *testing.T) {
	root := t.TempDir()
	data := encoded(t, func(b *bytes.Buffer, i image.Image) error { return png.Encode(b, i) })
	fh := fileHeader(t, "photo.PNG", data)

	s := Storage{Root: root}
	name, err := s.Save("places", fh)
	if err != nil {
		t.Fatal(err)
	}
	if !strings.HasPrefix(name, "places/") || !strings.HasSuffix(name, ".png") {
		t.Fatalf("name = %s", name)
	}
	got, err := os.ReadFile(filepath.Join(root, name))
	if err != nil || !bytes.Equal(got, data) {
		t.Fatalf("saved file differs: %v", err)
	}
	if err := s.Remove(name); err != nil {
		t.Fatal(err)
	}
	if _, err := os.Stat(filepath.Join(root, name)); !os.IsNotExist(err) {
		t.Fatalf("file still exists: %v", err)
	}
	if err := s.Remove("../outside.png"); err == nil {
		t.Fatal("Remove accepted ..")
	}
}

// testdata/*.webp созданы Pillow: Image.new("RGB", (37, 21)).save(..., lossless=…/save_all=…).
func TestWebPFromPillow(t *testing.T) {
	for _, name := range []string{"lossy.webp", "lossless.webp", "animated.webp"} {
		data, err := os.ReadFile(filepath.Join("testdata", name))
		if err != nil {
			t.Fatal(err)
		}
		if w, h, ok := webpSize(data); !ok || w != 37 || h != 21 {
			t.Errorf("%s: %dx%d ok=%v", name, w, h, ok)
		}
		if errs := ValidateImage(name, int64(len(data)), bytes.NewReader(data)); len(errs) != 0 {
			t.Errorf("%s: %v", name, errs)
		}
	}
}

// fileHeader разбирает настоящий multipart-запрос, чтобы получить *multipart.FileHeader.
func fileHeader(t *testing.T, filename string, data []byte) *multipart.FileHeader {
	t.Helper()
	var body bytes.Buffer
	mw := multipart.NewWriter(&body)
	fw, _ := mw.CreateFormFile("photo", filename)
	fw.Write(data)
	mw.Close()
	r := httptest.NewRequest("POST", "/", &body)
	r.Header.Set("Content-Type", mw.FormDataContentType())
	if err := r.ParseMultipartForm(1 << 20); err != nil {
		t.Fatal(err)
	}
	return r.MultipartForm.File["photo"][0]
}
